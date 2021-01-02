package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	pb "github.com/sergeyzalunin/grpc-go-course/calculator/calculatorpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	fmt.Println("Hi from client")

	cc, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	defer cc.Close()

	c := pb.NewCalculatorServiceClient(cc)

	// doUnary(c)
	// doServerStreaming(c)
	// doClientStreaming(c)
	// doBiDiStreaming(c)
	doErrorUnary(c)
}

func doUnary(c pb.CalculatorServiceClient) {
	req := &pb.SumRequest{
		FirstNumber:  2.0, //nolint
		SecondNumber: 3.3, //nolint
	}

	res, err := c.Sum(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(strconv.FormatFloat(res.SumResult, 'f', 6, 64))
}

func doServerStreaming(c pb.CalculatorServiceClient) {
	req := &pb.PrimeNumberDecompositionRequest{
		Number: 1200000, //nolint
	}

	resStream, err := c.PrimeNumberDecomposition(context.Background(), req)
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

		result := res.GetPrimeFactor()
		fmt.Println(result)
	}
}

func doClientStreaming(c pb.CalculatorServiceClient) {
	requests := []*pb.ComputeAverageRequest{
		{
			Number: 1,
		},
		{
			Number: 2,
		},
		{
			Number: 3,
		},
		{
			Number: 4,
		},
		{
			Number: 99,
		},
	}

	stream, err := c.ComputeAverage(context.Background())
	if err != nil {
		log.Fatalf("error while calling ComputeAverage: %v", err)
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
		log.Fatalf("error while receiving response from CalculateAverage: %v", err)
	}

	fmt.Println(res.GetAverage())
}

func doBiDiStreaming(c pb.CalculatorServiceClient) {
	fmt.Println("Starting to do BiDI streaming...")

	// we create a stream by invoking a client
	stream, err := c.FindMaximum(context.Background())
	if err != nil {
		log.Fatalf("Error while creating a stream: %v\n", err)

		return
	}

	requests := []*pb.FindMaximumRequest{
		{
			Number: 2,
		},
		{
			Number: 1,
		},
		{
			Number: 3,
		},
		{
			Number: 2,
		},
		{
			Number: 99,
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

			fmt.Println(res.GetMaximum())
		}
		close(waitc)
	}()

	// block until everything is done
	<-waitc
}

func doErrorUnary(c pb.CalculatorServiceClient) {
	//fmt.Println("Do error handling...")

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		doSquareRootCalc(c, 12)
		wg.Done()
	}()

	go func() {
		doSquareRootCalc(c, -12)
		wg.Done()
	}()

	wg.Wait()
}

func doSquareRootCalc(c pb.CalculatorServiceClient, n int32) {
	fmt.Printf("Do error handling of %d\n", n)

	req := &pb.SquareRootRequest{
		Number: n,
	}

	res, err := c.SquareRoot(context.Background(), req)
	if err != nil {
		state, ok := status.FromError(err)
		if ok {
			// actual error from gRPC (user error)
			fmt.Println(state.Message())
			fmt.Println(state.Err())

			if state.Code() == codes.InvalidArgument {
				fmt.Println("We probably sent a negative number")
			}
		} else {
			fmt.Printf("Big Error calling Square Root: %v\n", err)
		}

		return
	}

	fmt.Printf("Result of square root of %v: %v\n", n, res.GetNumberRoot())
}
