package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	pb "github.com/sergeyzalunin/grpc-go-course/calculator/calculatorpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

func main() {
	fmt.Println("Hi")

	listener, err := net.Listen("tcp", "0.0.0.0:50051") //nolint
	if err != nil {
		fmt.Println(err)
	}

	s := grpc.NewServer()
	pb.RegisterCalculatorServiceServer(s, &server{})
	reflection.Register(s)

	if err = s.Serve(listener); err != nil {
		fmt.Print(err)
	}
}

type server struct{}

func (s *server) Sum(_ context.Context, req *pb.SumRequest) (*pb.SumResponse, error) {
	fnum := req.FirstNumber
	snum := req.SecondNumber

	result := &pb.SumResponse{
		SumResult: fnum + snum,
	}

	return result, nil
}

func (s *server) PrimeNumberDecomposition(
	req *pb.PrimeNumberDecompositionRequest,
	stream pb.CalculatorService_PrimeNumberDecompositionServer,
) error {
	fmt.Printf("Received PrimeNumberDecomposed %v\n", req)

	number := req.GetNumber()
	divisor := int64(2)

	for number > 1 {
		if number%divisor == 0 {
			stream.Send(&pb.PrimeNumberDecompositionResponse{
				PrimeFactor: divisor,
			})

			number /= divisor
		} else {
			divisor++
			fmt.Println("Divisor increased")
		}
	}

	return nil
}

func (s *server) ComputeAverage(stream pb.CalculatorService_ComputeAverageServer) error {
	fmt.Println("Compute Average was invoked by stream")

	var result, count int32

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// we've finished the reading a client stream
			var average float64

			if count > 0 {
				average = float64(result) / float64(count)
			}

			return stream.SendAndClose(&pb.ComputeAverageResponse{
				Average: average,
			})
		}

		if err != nil {
			log.Fatal(err)
		}

		number := req.GetNumber()
		result += number
		count++
	}
}

// BiDi Streaming
func (s *server) FindMaximum(stream pb.CalculatorService_FindMaximumServer) error {
	fmt.Println("BiDi streaming started... Let's find the maximum!")

	var max int32

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			log.Fatalf("Error while receiving request: %v\n", err)

			return err
		}

		number := req.GetNumber()
		if number > max {
			max = number
		}

		err = stream.Send(&pb.FindMaximumResponse{
			Maximum: max,
		})

		if err != nil {
			log.Fatalf("Error while sending response: %v\n", err)

			return err
		}
	}
}

// error handling
// this RPC throw an exception if the sent number is negative
// the error being sent is of type INVALID_ARGUMENT
func (s *server) SquareRoot(
	_ context.Context,
	req *pb.SquareRootRequest,
) (*pb.SquareRootResponse, error) {
	fmt.Println("Received SquareRoot RPC...")

	number := req.GetNumber()

	if number < 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Received negative number: %v",
			number,
		)
	}

	return &pb.SquareRootResponse{
		NumberRoot: float64(number * number),
	}, nil
}
