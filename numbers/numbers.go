package numbers

import (
	"errors"
	hash "hash/crc32"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var (
	ErrStale               = errors.New("stale clientID")
	ErrRefusingConnections = errors.New("server refusing connections")
)

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

func MakeNumbersEntry(clientID uuid.UUID, count uint64) NumbersEntry {
	nums := make([]uint32, count)
	log.Printf("generating %d for %v", count, clientID)
	for i := 0; i < int(count); i++ {
		nums[i] = rand.Uint32()
	}

	return NumbersEntry{
		ClientID:       clientID,
		Numbers:        nums,
		PositionToSend: 0,
		LastSeen:       time.Now(),
		Checksum:       Checksum(nums),
	}
}

func Checksum(nums []uint32) uint32 {
	bytesForChecksum := []byte{}
	for _, num := range nums {
		bytesOfNumber := []byte(strconv.FormatInt(int64(num), 10))
		bytesForChecksum = append(bytesForChecksum, bytesOfNumber...)

	}

	return hash.Checksum(bytesForChecksum, hash.IEEETable)
}
