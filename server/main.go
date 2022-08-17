package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	pb "github.com/bmcgavin/numbers"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type server struct {
	r Repository
	pb.UnimplementedNumbersServer
}

func (s *server) GetNumbers(in *pb.NumbersRequest, stream pb.Numbers_GetNumbersServer) error {
	log.Printf("hi")
	log.Printf("generating %d for %v", in.GetCount(), in.GetUUID())
	///generate n numbers
	count := in.GetCount()
	clientID, err := uuid.Parse(in.GetUUID())
	if err != nil {
		log.Printf("received invalid uuid %v", in.GetUUID())
		return nil
	}
	ne := NumbersEntry{
		ClientID:       clientID,
		Numbers:        make([]uint32, count),
		PositionToSend: 0,
	}
	for i := 0; i < int(count); i++ {
		ne.Numbers[i] = rand.Uint32()
	}
	//store
	s.r.Put(clientID, ne)
	//stream
	for i := ne.PositionToSend; i < uint(len(ne.Numbers)); i++ {
		n := pb.Number{Number: ne.Numbers[ne.PositionToSend]}
		if err := stream.Send(&n); err != nil {
			log.Printf("could not send %v", err)
		}
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
}

type MemoryRepository struct {
	entries map[uuid.UUID]NumbersEntry
}

func MakeMemoryRepository() MemoryRepository {
	e := make(map[uuid.UUID]NumbersEntry)
	return MemoryRepository{entries: e}
}

func (r *MemoryRepository) Get(key uuid.UUID) NumbersEntry {
	ne := r.entries[key]
	return ne
}

func (r *MemoryRepository) Put(key uuid.UUID, val NumbersEntry) {
	r.entries[key] = val
}

type Repository interface {
	Get(key uuid.UUID) NumbersEntry
	Put(key uuid.UUID, val NumbersEntry)
}

func main() {
	port := 12345
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Couldn't listen to port %d: %v", port, err)
	}
	s := grpc.NewServer()

	pbs := &server{}
	r := MakeMemoryRepository()
	pbs.r = &r
	pb.RegisterNumbersServer(s, pbs)
	log.Printf("server listening on %d", port)
	if err := s.Serve(listener); err != nil {
		log.Fatalf("error serving %v", err)
	}

}
