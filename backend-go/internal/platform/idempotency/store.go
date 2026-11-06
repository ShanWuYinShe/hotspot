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
	// BodyHash 是请求体的 SHA-256（幂等上下文一致性校验，spec api_spec_errors.md
	// 22000「幂等上下文不匹配」）。存量记录为空时跳过比对（灰度兼容）。
	BodyHash string `json:"body_hash,omitempty"`
	// Headers 是 producer 在写响应体前设置的、需要在回放时原样重发的响应头。
	// 当前只保留 Set-Cookie：web 端 register/guest 的会话 cookie 由 producer
	// 在闭包内下发，而回放命中时 producer 不执行——不重发会让「网络超时后用
	// 同一 Idempotency-Key 重试」的客户端拿到 code=0 却没有会话（账号已建但
	// 处于未登录态）。存量记录为空 → 回放不补头（灰度兼容）。
	Headers map[string][]string `json:"headers,omitempty"`
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

// StartSweeper 为指定 Store 启动后台清扫（InMemoryStore 支持 Sweep；Redis
// 由服务端 TTL 清理，直接返回 no-op stop）。显式传入 Store，不再依赖全局
// store 单例；返回的 stop 幂等，可安全在测试/停机时调用。
func StartSweeper(ctx context.Context, store Store) (stop func()) {
	sweeper, ok := store.(interface{ Sweep(time.Time) })
	if !ok {
		return func() {}
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	worker.GoRestartable("idempotency.sweeper", func() { runSweeper(sweepCtx, sweeper) })
	return cancel
}

func runSweeper(ctx context.Context, store interface{ Sweep(time.Time) }) {
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
