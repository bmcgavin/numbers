package repository

import (
	"github.com/bmcgavin/numbers"

	"github.com/google/uuid"
)

type MemoryRepository struct {
	entries map[uuid.UUID]numbers.NumbersEntry
}

/**
 * This init is not throwing an err here but allows a more complex cache to do so
 */
func (r *MemoryRepository) Init() error {
	e := make(map[uuid.UUID]numbers.NumbersEntry)
	r.entries = e
	return nil
}

func (r *MemoryRepository) Get(key uuid.UUID) numbers.NumbersEntry {
	ne, ok := r.entries[key]
	if !ok {
		return numbers.NilNumbersEntry
	}
	return ne
}

func (r *MemoryRepository) Put(key uuid.UUID, val numbers.NumbersEntry) {
	r.entries[key] = val
}

type Repository interface {
	Init() error
	Get(key uuid.UUID) numbers.NumbersEntry
	Put(key uuid.UUID, val numbers.NumbersEntry)
}
