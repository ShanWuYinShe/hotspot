package idempotency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireKeyStoresContextValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	if err := RequireKey(req); err != nil {
		t.Fatalf("require key: %v", err)
	}
	if key := KeyFromContext(req.Context()); key == "" {
		t.Fatal("expected key in context")
	}
}

func TestInMemoryStoreRoundTrip(t *testing.T) {
	store := NewInMemoryStore()
	store.Save(Record{Key: "k", Scope: "s", Status: StatusProcessing, ExpiresAt: time.Now().Add(time.Hour)})
	record, ok := store.Get("s", "k")
	if !ok || record.Key != "k" {
		t.Fatalf("unexpected record: %#v, ok=%v", record, ok)
	}
}

// TestInMemoryStoreLazyExpiry verifies that Get distinguishes expired records
// from missing ones (StatusExpired vs not found), even before the sweeper has
// physically removed them. The replay layer maps StatusExpired to 22001.
func TestInMemoryStoreLazyExpiry(t *testing.T) {
	store := NewInMemoryStore()
	past := time.Now().Add(-time.Minute)
	store.Save(Record{Key: "expired", Scope: "s", Status: StatusCompleted, ExpiresAt: past})
	record, ok := store.Get("s", "expired")
	if !ok {
		t.Fatal("expected expired record to be returned as StatusExpired (distinguishable from missing)")
	}
	if record.Status != StatusExpired {
		t.Fatalf("expected StatusExpired for expired record, got %q", record.Status)
	}
	// Never-existed key must still be reported as missing.
	if _, ok := store.Get("s", "never_existed"); ok {
		t.Fatal("expected missing key to be not found")
	}
}

// TestInMemoryStoreReserveHonorsExpiry verifies Reserve returns StatusExpired
// for an expired record instead of silently overwriting it (so the replay
// layer can emit CodeIdempotencyExpired 22001 and let the client distinguish
// "曾存在但已过期" vs "从未存在").
func TestInMemoryStoreReserveHonorsExpiry(t *testing.T) {
	store := NewInMemoryStore()
	past := time.Now().Add(-time.Minute)
	store.Save(Record{Key: "k", Scope: "s", Status: StatusProcessing, ExpiresAt: past})
	future := time.Now().Add(time.Hour)
	record, ok := store.Reserve("s", "k", future)
	if ok {
		t.Fatal("expected Reserve to report not claimed for expired record (StatusExpired), not success")
	}
	if record.Status != StatusExpired {
		t.Fatalf("expected StatusExpired for expired Reserve, got %q", record.Status)
	}
}

// TestInMemoryStoreSweepPhysicallyRemovesExpired verifies Sweep deletes the
// underlying map entries for expired records (lazy expiry only hides them).
func TestInMemoryStoreSweepPhysicallyRemovesExpired(t *testing.T) {
	store := NewInMemoryStore()
	now := time.Now()
	store.Save(Record{Key: "expired", Scope: "s", Status: StatusCompleted, ExpiresAt: now.Add(-time.Minute)})
	store.Save(Record{Key: "live", Scope: "s", Status: StatusCompleted, ExpiresAt: now.Add(time.Hour)})
	store.Sweep(now)
	store.mu.RLock()
	_, hasExpired := store.records[composite("s", "expired")]
	_, hasLive := store.records[composite("s", "live")]
	store.mu.RUnlock()
	if hasExpired {
		t.Fatal("expected Sweep to delete expired record")
	}
	if !hasLive {
		t.Fatal("expected Sweep to keep live record")
	}
}

// StartSweeper 现在显式接收 Store：对支持 Sweep 的 store 启动后台清理，
// 对 Redis 等无 Sweep 的 store 返回 no-op stop；ctx 取消后 goroutine 退出。
func TestStartSweeperHonorsCancellation(t *testing.T) {
	store := NewInMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	stop := StartSweeper(ctx, store)
	cancel()
	stop()
	// 再次启动独立 sweeper 不应被上一次状态阻塞（无包级单例）。
	stop2 := StartSweeper(context.Background(), NewInMemoryStore())
	stop2()
}

// 无 Sweep 能力的 Store（如 RedisStore，服务端 TTL 管理）：启动应跳过，
// 返回可用（幂等）的 stop。
func TestStartSweeperSkipsStoreWithoutSweep(t *testing.T) {
	stop := StartSweeper(context.Background(), noopStore{})
	stop()
}

// stubStore 是任意非内存 Store 实现的替身：context 注入必须接受任意 Store。
type stubStore struct{ *InMemoryStore }

// 任意 Store 实现都必须能通过 context 注入并被 ReplayOrStore 读到（此前
// SetStore 只接受 *InMemoryStore，Redis store 被静默忽略）。
func TestWithStoreContextRoundTrip(t *testing.T) {
	custom := &stubStore{NewInMemoryStore()}
	ctx := WithStoreContext(context.Background(), custom)
	if got := storeFromContext(ctx); got != Store(custom) {
		t.Fatalf("expected injected store, got %T", got)
	}
	if got := storeFromContext(context.Background()); got != nil {
		t.Fatalf("expected nil store without injection, got %T", got)
	}
}

// noopStore implements Store without Sweep (like RedisStore, whose expiry
// is handled server-side via TTL): StartSweeper must skip it cleanly.
type noopStore struct{}

func (noopStore) Get(string, string) (Record, bool) { return Record{}, false }
func (noopStore) Reserve(string, string, time.Time) (Record, bool) {
	return Record{}, false
}
func (noopStore) Save(Record)           {}
func (noopStore) Delete(string, string) {}
