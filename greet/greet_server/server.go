package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"

	"github.com/sergeyzalunin/grpc-go-course/greet/greetpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/reflection"
)

func main() {
	fmt.Println("Hi")

	listener, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		fmt.Println(err)
	}

	opts := getTLSServerOptions(false)
	s := grpc.NewServer(opts...)
	greetpb.RegisterGreetServiceServer(s, &server{})
	reflection.Register(s)
	
	if err = s.Serve(listener); err != nil {
		fmt.Print(err)
	}
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

type server struct{}

// Unary Response.
func (s *server) Greet(ctx context.Context, req *greetpb.GreetRequest) (*greetpb.GreetResponse, error) {
	firstname := req.GetGreeting().GetFirstName()

	result := "Hello " + firstname + " from server"

	return &greetpb.GreetResponse{
		Result: result,
	}, nil
}

// Server Streaming.
func (s *server) GreetManyTimes(
	req *greetpb.GreetManyTimesRequest,
	stream greetpb.GreetService_GreetManyTimesServer,
) error {
	fname := req.GetGreeting().GetFirstName()

	runtime.GC()
	for i := 1; i < 10000000; i++ {
		result := "Hello " + fname + " number " + strconv.Itoa(i)

		response := &greetpb.GreetManyTimesResponse{
			Result: result,
		}

		err := stream.Send(response)
		if err != nil {
			log.Fatal(err)
		}

		//time.Sleep(100 * time.Microsecond) //nolint
	}

	runtime.GC()

	return nil
}

// Client Streaming.
func (s *server) LongGreet(stream greetpb.GreetService_LongGreetServer) error {
	fmt.Println("Long greetings was invoked by stream")

	result := ""

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// we've finished the reading a client stream
			return stream.SendAndClose(&greetpb.LongGreetResponse{
				Result: result,
			})
		}

		if err != nil {
			log.Fatal(err)
		}

		fname := req.GetGreeting().GetFirstName()
		result += "Hello " + fname + "!\n "
	}
}

// BiDi Streaming.
func (s *server) GreetEveryone(stream greetpb.GreetService_GreetEveryoneServer) error {
	fmt.Println("BiDi streaming started... Let's greet everyone!")

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			log.Fatalf("Error while receiving request: %v\n", err)

			return err
		}

		fname := req.GetGreeting().GetFirstName()
		result := "Hello " + fname + "! "

		err = stream.Send(&greetpb.GreetEveryoneResponse{
			Result: result,
		})

		if err != nil {
			log.Fatalf("Error while sending response: %v\n", err)

			return err
		}
	}
}

// Unary with Deadline.
func (s *server) GreetWithDedline(
	ctx context.Context,
	req *greetpb.GreetWithDedlineRequest,
) (*greetpb.GreetWithDedlineResponse, error) {
	fmt.Println("GreetWithDeadline was invoked by RPC...")

	for i := 0; i < 3; i++ {
		if errors.Is(ctx.Err(), context.Canceled) {
			fmt.Println("The request was cancelled")

			return nil, status.Error(codes.Canceled, "The client canceled the request")
		}
		time.Sleep(1 * time.Second)
	}

	firstname := req.GetGreeting().GetFirstName()

	result := "Hello " + firstname + " from server"

	return &greetpb.GreetWithDedlineResponse{
		Result: result,
	}, nil
}
