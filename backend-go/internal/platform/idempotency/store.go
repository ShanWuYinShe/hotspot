package idempotency

import (
	"context"
	"sync"
	"time"
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
)

// sweepCadence is how often the background sweeper physically removes
// expired records from the in-memory store. Get already applies lazy
// expiry (callers see expired records as missing), but without sweeping
// the map grows unboundedly in long-lived processes.
const sweepCadence = 5 * time.Minute

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

// Get applies lazy expiry: records past their ExpiresAt are reported as
// missing even before the sweeper physically removes them.
func (s *InMemoryStore) Get(scope, key string) (Record, bool) {
	now := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[composite(scope, key)]
	if !ok {
		return Record{}, false
	}
	if now.After(record.ExpiresAt) {
		return Record{}, false
	}
	return record, true
}

func (s *InMemoryStore) Reserve(scope, key string, expiresAt time.Time) (Record, bool) {
	k := composite(scope, key)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[k]; ok && now.Before(existing.ExpiresAt) {
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

// Sweep physically removes records whose ExpiresAt is in the past. Intended
// to be called by the background sweeper started via StartSweeper.
func (s *InMemoryStore) Sweep(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, rec := range s.records {
		if now.After(rec.ExpiresAt) {
			delete(s.records, k)
		}
	}
}

// sweeper state mirrors the pattern in ratelimit/middleware.go and
// platform/auth/sweeper.go: idempotent Start, ctx-bound goroutine,
// explicit Stop for tests.
var (
	sweeperMu      sync.Mutex
	sweeperCancel  context.CancelFunc
	sweeperStarted bool
)

// StartSweeper launches a background goroutine that periodically prunes
// expired records from the default in-memory store. It is safe to call
// from bootstrap; calling it again while a sweeper is already running is a
// no-op. The goroutine exits when ctx is cancelled.
//
// Get already applies lazy expiry so correctness does not depend on the
// sweeper; it just keeps the map from growing forever.
func StartSweeper(ctx context.Context) {
	sweeperMu.Lock()
	defer sweeperMu.Unlock()
	if sweeperStarted {
		return
	}
	store := defaultStore.Load()
	if store == nil {
		return
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	sweeperCancel = cancel
	sweeperStarted = true
	go runSweeper(sweepCtx, store)
}

// StopSweeper cancels the running sweeper if any. Intended for tests.
func StopSweeper() {
	sweeperMu.Lock()
	if sweeperCancel == nil {
		sweeperMu.Unlock()
		return
	}
	sweeperCancel()
	sweeperCancel = nil
	sweeperStarted = false
	sweeperMu.Unlock()
}

func runSweeper(ctx context.Context, store *InMemoryStore) {
	defer func() {
		sweeperMu.Lock()
		sweeperCancel = nil
		sweeperStarted = false
		sweeperMu.Unlock()
	}()
	ticker := time.NewTicker(sweepCadence)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			store.Sweep(time.Now())
		}
	}
}
