package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/sergeyzalunin/grpc-go-course/greet/greetpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func main() {
	fmt.Println("Hi from client")

	opts := getTLSClientOptions(true)

	cc, err := grpc.Dial("localhost:50051", opts)
	if err != nil {
		log.Fatal(err)
	}

	defer cc.Close()

	c := greetpb.NewGreetServiceClient(cc)

	// doUnary(c)
	// doServerStreaming(c)
	// doClientStreaming(c)
	// doBiDiStreaming(c)
	doGreetWithDeadline(c)
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

func doUnary(c greetpb.GreetServiceClient) {
	req := &greetpb.GreetRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Sergey",
			LastName:  "Budko",
		},
	}

	res, err := c.Greet(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(res.Result)
}

func doServerStreaming(c greetpb.GreetServiceClient) {
	fmt.Println("Requesting results from a stream....")

	req := &greetpb.GreetManyTimesRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Sergey",
			LastName:  "Budko",
		},
	}

	resStream, err := c.GreetManyTimes(context.Background(), req)
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

		result := res.GetResult()
		fmt.Println(result)
	}
}

func doClientStreaming(c greetpb.GreetServiceClient) {
	fmt.Println("Sending the request as a stream....")

	requests := []*greetpb.LongGreetRequest{
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Sergey",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Alexandr",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Vladimir",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Olga",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Gerasim",
			},
		},
	}

	stream, err := c.LongGreet(context.Background())
	if err != nil {
		log.Fatalf("error while calling LongGreet: %v", err)
	}

	for i := range requests {
		fmt.Printf("Sending request: %v\n", requests[i])

		err := stream.Send(requests[i])
		if err != nil {
			fmt.Printf("Error while sending requests: %v", err)
		}

		time.Sleep(1 * time.Second)
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("error while receiving response from LongGreet: %v", err)
	}

	fmt.Println(res.GetResult())
}

func doBiDiStreaming(c greetpb.GreetServiceClient) {
	fmt.Println("Starting to do BiDI streaming...")

	// we create a stream by invoking a client
	stream, err := c.GreetEveryone(context.Background())
	if err != nil {
		log.Fatalf("Error while creating a stream: %v\n", err)

		return
	}

	requests := []*greetpb.GreetEveryoneRequest{
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Sergey",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Alexandr",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Vladimir",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Olga",
			},
		},
		{
			Greeting: &greetpb.Greeting{ //nolint
				FirstName: "Gerasim",
			},
		},
	}

	waitc := make(chan struct{})

	// we send a bunch of messages to the server (go routine)
	go func() {
		// function to send a bunch of messages
		for i := range requests {
			fmt.Printf("Sending a request: %v\n", requests[i])
			err := stream.Send(requests[i])
			if err != nil {
				log.Fatalf("Error while sending request: %v\n", err)

				return
			}

			time.Sleep(1 * time.Second)
		}

		err := stream.CloseSend()
		if err != nil {
			log.Fatalf("error while closing the stream")
		}
	}()

	// we receive a bunch of messages from the server (go routine)
	go func() {
		// function to receive a bunch of messages
		for {
			res, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				log.Fatalf("error while receiving a greeting: %v\n", err)

				break
			}

			fmt.Println(res.GetResult())
		}
		close(waitc)
	}()

	// block until everything is done
	<-waitc
}

func doGreetWithDeadline(c greetpb.GreetServiceClient) {
	doGreetWithDeadlineDuration(c, 5*time.Second)
	doGreetWithDeadlineDuration(c, 1*time.Second)
}

func doGreetWithDeadlineDuration(c greetpb.GreetServiceClient, d time.Duration) {
	fmt.Println("Starting to do UnaryWithDeadline...")

	req := &greetpb.GreetWithDedlineRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Sergey",
			LastName:  "Budko",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	res, err := c.GreetWithDedline(ctx, req)
	if err != nil {
		errState, ok := status.FromError(err)
		if ok {
			if errState.Code() == codes.DeadlineExceeded {
				fmt.Println("Timeout was hit! Deadline was exceeded!")
			} else {
				fmt.Printf("unexcepted error: %v\n", errState.Err())
			}
		} else {
			fmt.Printf("Error while calling GreetWithDeadline: %v\n", err)
		}

		return
	}

	fmt.Printf("Response from GreetWithDedline: %v\n", res.Result)
}
