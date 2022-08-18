package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/bmcgavin/numbers"
	"github.com/bmcgavin/repository"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

var (
	port         = flag.Int("port", 12345, "the port to connect to")
	failEvery    = flag.Uint("failEvery", 0, "[debug] failEvery so many messages")
	goOfflineFor = flag.Int64("goOfflineFor", 0, "[debug] refuse connections for this long after failEvery triggers")
	testCase     = flag.Int("testCase", 0, "[debug] test case to run")

	goOfflineUntil = time.Now()
)

// sporadic failures based on https://github.com/grpc/grpc-go/blob/v1.48.0/examples/features/retry/server/main.go
type server struct {
	r  repository.Repository
	mu sync.Mutex

	reqCounter     uint
	reqModulo      uint
	goOfflineUntil time.Time
	goOfflineFor   int64
	numbers.UnimplementedNumbersServer
}

// this method will succees on reqModulo - 1 times RPCs
// and fail (return status code Unavailable) on reqModulo || reqModuloTwo times.
func (s *server) maybeFailRequest() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqCounter++
	if (s.reqModulo > 0) && (s.reqCounter%s.reqModulo == 0) {
		return status.Errorf(codes.Unavailable, "maybeFailRequest: failing it")
	}

	return nil

}

func (s *server) GetNumbers(in *numbers.NumbersRequest, stream numbers.Numbers_GetNumbersServer) error {

	// refuse connections if we need to
	if time.Now().Before(s.goOfflineUntil) {
		return numbers.ErrRefusingConnections
	}

	// validate client
	clientID, err := uuid.Parse(in.GetUUID())
	if err != nil {
		log.Printf("received invalid uuid %v", in.GetUUID())
		return nil
	}
	//get from cache
	ne, err := getFromCache(s.r, clientID)
	if err == numbers.ErrStale {
		// e := "stale clientID %v, please regenerate and retry"
		return err
	}
	if err != nil {
		log.Printf("could not read cache %v", err)
	}

	// found in cache
	if !ne.IsZero() {
		log.Printf("using cached entry for %v", in.GetUUID())
	} else {
		// new client
		ne = numbers.MakeNumbersEntry(clientID, in.GetCount())
		//store
		s.r.Put(clientID, ne)
	}

	//stream
	for i := ne.PositionToSend; i < uint(len(ne.Numbers)); i++ {
		n := numbers.NumbersResponse{Number: ne.Numbers[ne.PositionToSend]}
		//final message
		if ne.PositionToSend == uint(len(ne.Numbers))-1 {
			n.Checksum = &ne.Checksum
		}
		if err := s.maybeFailRequest(); err != nil {
			//get a length of time to fail for
			if s.goOfflineFor > 0 {
				s.goOfflineUntil = time.Now().Add(time.Duration(s.goOfflineFor) * time.Second)
			}
			//store for later
			ne.LastSeen = time.Now()
			s.r.Put(clientID, ne)
			return err
		}
		if err := stream.Send(&n); err != nil {
			log.Printf("could not send %v", err)
			//store for later
			ne.LastSeen = time.Now()
			s.r.Put(clientID, ne)
			break
		}
		log.Printf("sent %v, PTS %d", n.Number, ne.PositionToSend)
		ne.PositionToSend++
		time.Sleep(1 * time.Second)
	}

	return nil
}

func getFromCache(r repository.Repository, key uuid.UUID) (numbers.NumbersEntry, error) {
	ne := r.Get(key)
	if !ne.IsZero() {
		//in cache, check staleness
		if time.Since(ne.LastSeen) > 30*time.Second {
			//in cache but stale
			// > any subsequent reconnection attempt for that client id must be rejected with a suitable error
			return ne, numbers.ErrStale

		}
	}
	return ne, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Couldn't listen to port %d: %v", *port, err)
	}
	sp := keepalive.ServerParameters{
		Time: 1 * time.Second,
	}
	kp := grpc.KeepaliveParams(sp)

	s := grpc.NewServer(kp)

	r := repository.MemoryRepository{}
	err = r.Init()
	if err != nil {
		log.Fatalf("couldn't initiate repository %v", err)
	}
	pbs := &server{
		reqCounter:     0,
		reqModulo:      *failEvery,
		goOfflineUntil: goOfflineUntil,
		goOfflineFor:   *goOfflineFor,
	}
	pbs.r = &r
	numbers.RegisterNumbersServer(s, pbs)

	log.Printf("server starting on %d", *port)
	if err := s.Serve(listener); err != nil {
		log.Fatalf("error serving %v", err)
	}

}
