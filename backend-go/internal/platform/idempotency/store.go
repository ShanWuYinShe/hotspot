package idempotency

import (
	"sync"
	"time"
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
)

type Record struct {
	Key       string
	Scope     string
	Status    Status
	Response  []byte
	ExpiresAt time.Time
}

type Store interface {
	Get(scope, key string) (Record, bool)
	Save(record Record)
}

type InMemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{records: make(map[string]Record)} }

func composite(scope, key string) string { return scope + "::" + key }

func (s *InMemoryStore) Get(scope, key string) (Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[composite(scope, key)]
	return record, ok
}
func (s *InMemoryStore) Save(record Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[composite(record.Scope, record.Key)] = record
}
