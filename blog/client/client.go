package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/sergeyzalunin/grpc-go-course/blog/blogpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("Blog client")

	opts := getTLSClientOptions(true)

	cc, err := grpc.Dial("localhost:50052", opts)
	if err != nil {
		log.Fatal(err)
	}

	defer cc.Close()

	c := blogpb.NewBlogServiceClient(cc)

	blog := &blogpb.Blog{
		Id:       "",
		AuthorId: "Ninja",
		Title:    "My first blog",
		Content:  "Content of the first blog",
	}

	createNewBlog(c, blog)
	readBlog(c, blog.Id)
	updateBlog(c, blog.Id)
	getListBlog(c)
	deleteBlog(c, blog)
}

// getTLSClientOptions returns already setup TLS dial option
// when tls parameter is true
// to use it, start gen_cert.sh manualy to generate certificates
func getTLSClientOptions(tls bool) grpc.DialOption {
	var opts grpc.DialOption

	if tls {
		certFile := "../../ssl/minica.pem" // Certificate Authority Trust certificate

		creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
		if sslErr == nil {
			opts = grpc.WithTransportCredentials(creds)
		} else {
			log.Fatalf("Filed loading certificates: %v", sslErr)
		}
	} else {
		opts = grpc.WithInsecure()
	}

	return opts
}

// create a new blog.
func createNewBlog(c blogpb.BlogServiceClient, blog *blogpb.Blog) {
	blogRes, err := c.CreateBlog(context.Background(), &blogpb.CreateBlogRequest{
		Blog: blog,
	})
	if err == nil {
		fmt.Printf("%+v\n\n", blogRes.GetBlog())
		blog.Id = blogRes.Blog.Id
	} else {
		fmt.Println(err)
	}
}

// read a blog.
func readBlog(c blogpb.BlogServiceClient, blogID string) {
	blogNext, err := c.ReadBlog(context.Background(), &blogpb.ReadBlogRequest{
		BlogId: blogID,
	})
	if err == nil {
		fmt.Printf("%+v\n\n", blogNext.GetBlog())
	} else {
		fmt.Println(err)
	}
}

// updating a blog by its ID.
func updateBlog(c blogpb.BlogServiceClient, blogID string) {
	updatedBlog := &blogpb.Blog{
		Id:       blogID,
		AuthorId: "TESTER",
		Title:    "Changed Name",
		Content:  "Changed content",
	}

	updBlog, err := c.UpdateBlog(context.Background(), &blogpb.UpdateBlogRequest{
		Blog: updatedBlog,
	})
	if err == nil {
		fmt.Printf("%+v\n\n", updBlog.GetBlog())
	} else {
		fmt.Println(err)
	}
}

func getListBlog(c blogpb.BlogServiceClient) {
	fmt.Println("Requesting blogs from a stream....")

	req := &blogpb.ListBlogRequest{}

	resStream, err := c.ListBlog(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	for {
		res, err := resStream.Recv()
		if errors.Is(err, io.EOF) {
			// we've reached the end of file
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		result := res.GetBlog()
		fmt.Printf("%+v\n\n", result)
	}
}

// deleting blog.
func deleteBlog(c blogpb.BlogServiceClient, blog *blogpb.Blog) {
	delBlog, err := c.DeleteBlog(context.Background(), &blogpb.DeleteBlogRequest{
		Blog: blog,
	})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%v\n\n", delBlog.GetBlogId())
	}
}
