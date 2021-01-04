package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/sergeyzalunin/grpc-go-course/blog/blogpb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var collection *mongo.Collection

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("Blog server started...")

	listener, err := net.Listen("tcp", "localhost:50052")
	if err != nil {
		fmt.Println(err)
	}

	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://root:example@172.28.1.3:27017"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database("testing").Collection("numbers")

	opts := getTLSServerOptions(true)
	s := grpc.NewServer(opts...)
	blogpb.RegisterBlogServiceServer(s, &server{})
	reflection.Register(s)

	go func() {
		if err = s.Serve(listener); err != nil {
			fmt.Print(err)
		}

		fmt.Println("Closing Mongo DB connection")
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block until a signal is received
	<-ch
	fmt.Println("Stopping the server")
	s.GracefulStop()

	fmt.Println("Closing the listener")
	listener.Close()

	fmt.Println("End of Program")
}

// getTLSServerOptions returns already setup TLS server option
// when tls parameter is true
// to use it, start gen_cert.sh manualy to generate certificates
func getTLSServerOptions(tls bool) []grpc.ServerOption {
	opts := []grpc.ServerOption{}

	if tls {
		certFile := "../../ssl/localhost/cert.pem"
		keyFile := "../../ssl/localhost/key.pem"

		creds, sslErr := credentials.NewServerTLSFromFile(certFile, keyFile)
		if sslErr == nil {
			opts = append(opts, grpc.Creds(creds))
		} else {
			log.Fatalf("Filed loading certificates: %v", sslErr)
		}
	}

	return opts
}

type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id"`
	Title    string             `bson:"title"`
	Content  string             `bson:"content"`
}

type server struct {
}

func (s *server) CreateBlog(
	ctx context.Context,
	req *blogpb.CreateBlogRequest,
) (*blogpb.CreateBlogResponse, error) {
	fmt.Println("Create blog")

	blog := req.GetBlog()

	data := blogItem{
		AuthorID: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}

	res, err := collection.InsertOne(ctx, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"internal error: %v",
			err,
		)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			"Cannot conver to OID: %v",
			err,
		)
	}

	blog.Id = oid.Hex()

	return &blogpb.CreateBlogResponse{
		Blog: blog,
	}, nil
}

func (s *server) ReadBlog(
	ctx context.Context,
	req *blogpb.ReadBlogRequest,
) (*blogpb.ReadBlogResponse, error) {
	fmt.Println("Read blog")

	blogId := req.GetBlogId()

	oid, err := primitive.ObjectIDFromHex(blogId)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot parse ID: %v",
			blogId,
		)
	}

	// create an empty document
	data := &blogItem{}
	filter := bson.M{"_id": oid}

	err = collection.FindOne(ctx, filter).Decode(data)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Errorf(
				codes.NotFound,
				"There is no blog with id: %s",
				blogId,
			)
		}

		return nil, status.Errorf(
			codes.Internal,
			"Something went wrong: %v",
			err,
		)
	}

	return &blogpb.ReadBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil
}

func (s *server) UpdateBlog(
	ctx context.Context,
	req *blogpb.UpdateBlogRequest,
) (*blogpb.UpdateBlogResponse, error) {
	fmt.Println("Update blog")

	blog := req.GetBlog()

	oid, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot parse ID: %v",
			blog.GetId(),
		)
	}

	filter := bson.M{
		"_id": oid,
	}

	update := bson.M{
		"$set": bson.M{
			"author_id": blog.GetAuthorId(),
			"title":     blog.GetTitle(),
			"content":   blog.GetContent(),
		},
	}

	opts := &options.FindOneAndUpdateOptions{}
	opts.SetReturnDocument(options.After)

	data := &blogItem{}

	err = collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Something went wrong: %v",
			err,
		)
	}

	return &blogpb.UpdateBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil
}

func dataToBlogPb(data *blogItem) *blogpb.Blog {
	return &blogpb.Blog{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorID,
		Title:    data.Title,
		Content:  data.Content,
	}
}

func (s *server) DeleteBlog(
	ctx context.Context,
	req *blogpb.DeleteBlogRequest,
) (*blogpb.DeleteBlogResponse, error) {
	fmt.Println("Delete blog")

	blog := req.GetBlog()

	oid, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot parse ID: %v",
			blog.GetId(),
		)
	}

	filter := bson.M{
		"_id": oid,
	}

	data := &blogItem{}

	err = collection.FindOneAndDelete(ctx, filter).Decode(data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Something went wrong: %v",
			err,
		)
	}

	return &blogpb.DeleteBlogResponse{
		BlogId: data.ID.Hex(),
	}, nil
}

func (s *server) ListBlog(
	_ *blogpb.ListBlogRequest,
	stream blogpb.BlogService_ListBlogServer,
) error {
	fmt.Println("List of blogs")

	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			"Unexpected err: %v",
			err,
		)
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		blog := &blogItem{}

		err := cur.Decode(blog)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				"Error while decoding data: %v",
				err,
			)
		}

		err = stream.Send(&blogpb.ListBlogResponse{
			Blog: dataToBlogPb(blog),
		})

		if err != nil {
			return status.Errorf(
				codes.Internal,
				"Error while sending data: %v",
				err,
			)
		}
	}

	return nil
}
