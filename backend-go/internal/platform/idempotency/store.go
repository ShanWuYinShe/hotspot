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
	// Reserve atomically inserts a "processing" record when none exists; it
	// returns the existing record if one is already present. This lets the
	// replay path decide replay-vs-in-flight without a separate check-then-act
	// (and without unsafe type assertions on the concrete store type).
	Reserve(scope, key string, expiresAt time.Time) (Record, bool)
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

func (s *InMemoryStore) Reserve(scope, key string, expiresAt time.Time) (Record, bool) {
	k := composite(scope, key)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[k]; ok {
		return existing, false
	}
	rec := Record{Key: key, Scope: scope, Status: StatusProcessing, ExpiresAt: expiresAt}
	s.records[k] = rec
	return rec, true
}

func (s *InMemoryStore) Save(record Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[composite(record.Scope, record.Key)] = record
}
