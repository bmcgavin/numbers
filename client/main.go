package main

import (
	"context"
	"io"
	"log"

	pb "github.com/bmcgavin/numbers"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	//TODO cli args
	addr := "localhost:12345"
	var count uint32 = 5

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect error %v", err)
	}
	defer conn.Close()
	c := pb.NewNumbersClient(conn)

	u := uuid.New()
	nr := pb.NumbersRequest{UUID: u.String(), Count: &count}

	ctx := context.Background()
	stream, err := c.GetNumbers(ctx, &nr)
	if err != nil {
		log.Fatalf("couldn't get numbers %v", err)
	}
	for {
		number, err := stream.Recv()
		if err == io.EOF {
			log.Printf("io error %v", err)
			break
		}
		if err != nil {
			log.Printf("error %v", err)
		}
		log.Printf("number %v", number)
	}
}
