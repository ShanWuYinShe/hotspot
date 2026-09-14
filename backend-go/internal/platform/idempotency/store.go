package idempotency

import (
	"context"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/worker"
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusExpired    Status = "expired"
)

// sweepCadence is how often the background sweeper physically removes
// expired records from the in-memory store. Get/Reserve already apply lazy
// expiry (callers see expired as StatusExpired → 22001), but without sweeping
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
	// Delete removes a record outright — used to roll back the processing
	// marker when the producer panics, so the key isn't poisoned for its TTL.
	Delete(scope, key string)
}

type InMemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{records: make(map[string]Record)} }

func composite(scope, key string) string { return scope + "::" + key }

// Get applies lazy expiry: records past their ExpiresAt are reported as
// expired (StatusExpired) even before the sweeper physically removes them.
// Callers can distinguish "曾存在但已过期" vs "从未存在" by checking
// record.Status == StatusExpired, which the replay layer maps to 22001.
func (s *InMemoryStore) Get(scope, key string) (Record, bool) {
	now := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[composite(scope, key)]
	if !ok {
		return Record{}, false
	}
	if now.After(record.ExpiresAt) {
		return Record{
			Key:       record.Key,
			Scope:     scope,
			Status:    StatusExpired,
			Response:  record.Response,
			ExpiresAt: record.ExpiresAt,
		}, true
	}
	return record, true
}

func (s *InMemoryStore) Reserve(scope, key string, expiresAt time.Time) (Record, bool) {
	k := composite(scope, key)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[k]; ok {
		if now.After(existing.ExpiresAt) {
			// 已存在但已过期：返回 StatusExpired 让上层映射为 CodeIdempotencyExpired (22001)，
			// 而非直接视为可重新占用。保留原记录直到 sweeper 清理，期间同一过期 key 的
			// 重试持续返回 22001，满足"曾存在但已过期" vs "从未存在"的区分。
			return Record{
				Key:       existing.Key,
				Scope:     scope,
				Status:    StatusExpired,
				Response:  existing.Response,
				ExpiresAt: existing.ExpiresAt,
			}, false
		}
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

// Delete removes the record immediately (no lazy expiry), so a retried
// Reserve can claim the key again.
func (s *InMemoryStore) Delete(scope, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, composite(scope, key))
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
// expired records from the default store when it supports sweeping
// (InMemoryStore). Stores with server-side expiry (RedisStore via TTL) are
// left alone. It is safe to call from bootstrap; calling it again while a
// sweeper is already running is a no-op. The goroutine exits when ctx is
// cancelled.
//
// Get/Reserve already apply lazy expiry (StatusExpired → 22001) so correctness
// does not depend on the sweeper; it just keeps the map from growing forever.
func StartSweeper(ctx context.Context) {
	sweeperMu.Lock()
	defer sweeperMu.Unlock()
	if sweeperStarted {
		return
	}
	store, ok := currentStore().(interface{ Sweep(time.Time) })
	if !ok {
		return
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	sweeperCancel = cancel
	sweeperStarted = true
	worker.Go("idempotency.sweeper", func() { runSweeper(sweepCtx, store) })
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

func runSweeper(ctx context.Context, store interface{ Sweep(time.Time) }) {
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
