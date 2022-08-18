package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/bmcgavin/numbers"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"google.golang.org/grpc/credentials/insecure"
)

var (
	port             = flag.Int("port", 12345, "the port to connect to")
	count            = flag.Uint64("count", 0, "number of messages to receive")
	overrideClientID = flag.String("clientID", "", "[debug] override clientID'")

	clientID = uuid.UUID{}
	addr     = ""
)

func connectGrpc(ctx context.Context, addr string) (*grpc.ClientConn, error) {

	ka := keepalive.ClientParameters{
		Time:    500 * time.Millisecond,
		Timeout: 100 * time.Millisecond,
	}
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(ka),
	)
	return conn, err
}

func connectStream(ctx context.Context, conn *grpc.ClientConn, nr numbers.NumbersRequest) (numbers.Numbers_GetNumbersClient, error) {

	c := numbers.NewNumbersClient(conn)

	return c.GetNumbers(ctx, &nr)
}

func cli() {
	flag.Parse()
	addr = fmt.Sprintf("localhost:%d", *port)
	if *count == 0 {
		rand.Seed(time.Now().UnixNano())
		*count = rand.Uint64() % 0xffff
	}
	if *overrideClientID != "" {
		var err error
		clientID, err = uuid.Parse(*(overrideClientID))
		if err != nil {
			log.Fatalf("supplied clientID %s is not a UUID, %v", *overrideClientID, err)
		}
	} else {
		clientID = uuid.New()
	}
}

func reconnect(ctx context.Context, nr numbers.NumbersRequest) (*grpc.ClientConn, numbers.Numbers_GetNumbersClient, error) {
	initialDelay := 1.1
	maxRetries := 10
	var stream numbers.Numbers_GetNumbersClient
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < maxRetries; i++ {
		// backoff
		sleepFor := math.Pow(float64(initialDelay), float64(i))
		log.Printf("sleeping for %v", sleepFor)
		time.Sleep(time.Duration(sleepFor) * time.Second)

		// grpc connection
		conn, err = connectGrpc(ctx, addr)
		if err != nil {
			log.Printf("failed to reconnect grpc %v", err)
			continue
		}

		// stream connection
		stream, err = connectStream(ctx, conn, nr)
		if err != nil {
			log.Printf("failed to reconnect stream %v", err)
			continue
		}
		if stream != nil {
			break
		}
	}
	if err != nil {
		return nil, nil, err
	}
	return conn, stream, nil
}

func main() {
	cli()

	// ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	// defer cancel()
	ctx := context.Background()

	conn, err := connectGrpc(ctx, addr)
	if err != nil {
		log.Fatalf("connect error %v", err)
	}
	defer conn.Close()

	nr := numbers.NumbersRequest{UUID: clientID.String(), Count: count}

	stream, err := connectStream(ctx, conn, nr)
	if err != nil {
		log.Fatalf("couldn't connect to the stream %v", err)
	}
	nums := []uint32{}
	crc := uint32(0)
	numReceived := 0
	cNum := make(chan *numbers.NumbersResponse, 1)
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
			nums = append(nums, number.Number)
			log.Printf("number %v", number)
			if number.Checksum != nil {
				crc = *number.Checksum
			}
		case err := <-cErr:
			if err == io.EOF {
				done = true
				break
			}
			// this doesn't work but should be an RPC return code
			if err == numbers.ErrStale {
				log.Fatalf("%v", err)
			}
			if err != nil {
				log.Printf("error %v", err)
				// ctx, cancel = context.WithTimeout(context.Background(), 6*time.Second)
				conn, stream, err = reconnect(ctx, nr)
				if err != nil {
					log.Fatalf("couldn't reconnect %v", err)
				}
				defer conn.Close()
			}
		// may be useless with the retry pinging, there's also a grpc native server config that could be used
		case <-time.After(2 * time.Second):
			log.Printf("timeout received")
		}

		if done {
			break
		}
	}
	if crc == numbers.Checksum(nums) {
		log.Printf("checksums match")
	} else {
		log.Printf("checksums do not match, got %v expected %v", crc, numbers.Checksum(nums))
	}

}
