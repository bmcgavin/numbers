package main

import (
	"context"
	"flag"
	"fmt"
	hash "hash/crc32"
	"io"
	"log"
	"math/rand"
	"time"

	pb "github.com/bmcgavin/numbers"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"google.golang.org/grpc/credentials/insecure"
)

var (
	port             = flag.Int("port", 12345, "the port to connect to")
	count            = flag.Uint64("count", 0, "number of messages to receive")
	overrideClientID = flag.String("clientID", "", "[debug] override clientID'")
	pauseAfter       = flag.Int("pauseAfter", 0, "[debug] pause after this many messages")
	pauseFor         = flag.Int64("pauseFor", 3, "[debug] pause for this many seconds if pauseAfter != 0")
)

func connect(ctx context.Context, addr string) (*grpc.ClientConn, error) {

	cp := keepalive.ClientParameters{
		Time: 1 * time.Second,
	}
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials(),
			grpc.WithClientParameters(cp)),
	)
	return conn, err
}

func main() {
	flag.Parse()
	addr := fmt.Sprintf("localhost:%d", *port)
	if *count == 0 {
		rand.Seed(time.Now().UnixNano())
		*count = rand.Uint64() % 0xffff
	}
	fmt.Printf("%d", *count)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := connect(ctx, addr)
	if err != nil {
		log.Fatalf("connect error %v", err)
	}
	defer conn.Close()
	c := pb.NewNumbersClient(conn)

	var u uuid.UUID
	if *overrideClientID != "" {
		u, err = uuid.Parse(*(overrideClientID))
		if err != nil {
			log.Fatalf("supplied clientID %s is not a UUID, %v", *overrideClientID, err)
		}
	} else {
		u = uuid.New()
	}
	nr := pb.NumbersRequest{UUID: u.String(), Count: count}

	stream, err := c.GetNumbers(ctx, &nr)
	if err != nil {
		log.Fatalf("couldn't get numbers %v", err)
	}
	b := []byte{}
	crc := uint32(0)
	numReceived := 0
	cNum := make(chan *pb.NumbersResponse, 1)
	cErr := make(chan error, 1)
	done := false
	for {
		go func() {
			number, err := stream.Recv()
			if err != nil {
				cErr <- err
			} else {
				cNum <- number
			}
		}()
		select {
		case number := <-cNum:
			numReceived++
			b = append(b, byte(number.Number))
			log.Printf("number %v", number)
			if number.Checksum != nil {
				crc = *number.Checksum
			}
		case err := <-cErr:
			if err == io.EOF {
				done = true
				break
			}
			if err != nil {
				log.Printf("error %v", err)
				ctx, cancel = context.WithTimeout(context.Background(), 6*time.Second)
				connect(ctx, addr)

				// time.Sleep(5 * time.Second)
			}
		case <-time.After(2 * time.Second):
			log.Printf("timeout received")
		}

		if done {
			break
		}

		if *pauseAfter > 0 && *pauseAfter == numReceived {
			cancel()
		}
	}
	log.Printf("Checksum received: %v", crc)
	log.Printf("Checksum calculated: %v", hash.Checksum(b, hash.IEEETable))
}
