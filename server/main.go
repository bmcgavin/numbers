package main

import (
	"flag"
	"fmt"
	hash "hash/crc32"
	"log"
	"math/rand"
	"net"
	"time"

	pb "github.com/bmcgavin/numbers"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type server struct {
	r Repository
	pb.UnimplementedNumbersServer
}

func (s *server) GetNumbers(in *pb.NumbersRequest, stream pb.Numbers_GetNumbersServer) error {

	log.Printf("generating %d for %v", in.GetCount(), in.GetUUID())
	///generate n numbers
	count := in.GetCount()

	clientID, err := uuid.Parse(in.GetUUID())
	if err != nil {
		log.Printf("received invalid uuid %v", in.GetUUID())
		return nil
	}
	//get from cache
	ne := s.r.Get(clientID)
	if !ne.IsZero() {
		//in cache, check staleness
		if time.Since(ne.LastSeen) > 30*time.Second {
			//in cache but stale
			// > any subsequent reconnection attempt for that client id must be rejected with a suitable error
			e := "Stale clientID %v, please regenerate and retry"
			n := pb.NumbersResponse{Number: 0, Error: &e}
			if err := stream.Send(&n); err != nil {
				log.Printf("could not send error response %v", err)
			}
			return nil
		}
	} else {
		ne = NumbersEntry{
			ClientID:       clientID,
			Numbers:        make([]uint32, count),
			PositionToSend: 0,
			LastSeen:       time.Now(),
			Checksum:       0,
		}
	}

	b := []byte{}
	for i := 0; i < int(count); i++ {
		ne.Numbers[i] = rand.Uint32()
		b = append(b, byte(ne.Numbers[i]))
	}
	ne.Checksum = hash.Checksum(b, hash.IEEETable)
	//store
	s.r.Put(clientID, ne)
	//stream
	for i := ne.PositionToSend; i < uint(len(ne.Numbers)); i++ {
		n := pb.NumbersResponse{Number: ne.Numbers[ne.PositionToSend]}
		//last message
		if ne.PositionToSend == uint(len(ne.Numbers))-1 {
			n.Checksum = &ne.Checksum
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
	//on error

	// wait up to 30s
	// dump or resume
	return nil
}

type NumbersEntry struct {
	ClientID       uuid.UUID
	Numbers        []uint32
	PositionToSend uint
	LastSeen       time.Time
	Checksum       uint32
}

var NilNumbersEntry NumbersEntry = NumbersEntry{
	ClientID:       uuid.Nil,
	Numbers:        []uint32{},
	PositionToSend: 0,
	LastSeen:       time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
	Checksum:       0,
}

func (ne NumbersEntry) IsZero() bool {
	return ne.ClientID == NilNumbersEntry.ClientID &&
		len(ne.Numbers) == len(NilNumbersEntry.Numbers) &&
		ne.PositionToSend == NilNumbersEntry.PositionToSend &&
		ne.LastSeen == NilNumbersEntry.LastSeen &&
		ne.Checksum == NilNumbersEntry.Checksum
}

type MemoryRepository struct {
	entries map[uuid.UUID]NumbersEntry
}

func MakeMemoryRepository() MemoryRepository {
	e := make(map[uuid.UUID]NumbersEntry)
	return MemoryRepository{entries: e}
}

func (r *MemoryRepository) Get(key uuid.UUID) NumbersEntry {
	ne, ok := r.entries[key]
	if !ok {
		return NilNumbersEntry
	}
	return ne
}

func (r *MemoryRepository) Put(key uuid.UUID, val NumbersEntry) {
	r.entries[key] = val
}

type Repository interface {
	Get(key uuid.UUID) NumbersEntry
	Put(key uuid.UUID, val NumbersEntry)
}

var (
	port = flag.Int("port", 12345, "the port to connect to")
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Couldn't listen to port %d: %v", *port, err)
	}
	sp := keepalive.ServerParameters{
		Time: 1 * time.Second,
	}
	s := grpc.NewServer(sp)

	pbs := &server{}
	r := MakeMemoryRepository()
	pbs.r = &r
	pb.RegisterNumbersServer(s, pbs)
	log.Printf("server listening on %d", *port)
	if err := s.Serve(listener); err != nil {
		log.Fatalf("error serving %v", err)
	}

}
