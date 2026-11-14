package idempotency

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

// —— 容量上限与淘汰 ——
//
// records/order/elems 三个结构必须始终同步：只淘汰 map 而不摘链表会让淘汰
// 顺序结构本身成为第二处无界内存，因此这些用例同时断言三者规模一致。

// storeSizes 读取存储内部规模（测试专用，避免依赖字段名以外的东西）。
func storeSizes(s *InMemoryStore) (records, order, elems int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records), s.order.Len(), len(s.elems)
}

func mustConsistent(t *testing.T, store *InMemoryStore) {
	t.Helper()
	records, order, elems := storeSizes(store)
	if records != order || records != elems {
		t.Fatalf("inconsistent store bookkeeping: records=%d order=%d elems=%d", records, order, elems)
	}
}

// 生产构造必须带默认容量上限（长跑进程内存有界的前提）。
func TestNewInMemoryStoreUsesDefaultCapacity(t *testing.T) {
	if got := NewInMemoryStore().maxEntries; got != maxInMemoryRecords {
		t.Fatalf("expected default capacity %d, got %d", maxInMemoryRecords, got)
	}
}

// 容量满后按「最早写入」淘汰：最老记录被移除（视为从未存在），新记录正常写入，
// 规模恒定不再增长；被淘汰的 key 可以重新占用（Reserve 成功）。
func TestInMemoryStoreEvictsEarliestWhenFull(t *testing.T) {
	store := newInMemoryStore(3)
	now := time.Now()
	future := now.Add(time.Hour)
	for i := 0; i < 3; i++ {
		store.Save(Record{Key: fmt.Sprintf("k%d", i), Scope: "s", Status: StatusCompleted, ExpiresAt: future})
	}
	store.Save(Record{Key: "new", Scope: "s", Status: StatusCompleted, ExpiresAt: future})

	if size, _, _ := storeSizes(store); size != 3 {
		t.Fatalf("expected store to stay at capacity 3, got %d", size)
	}
	if _, ok := store.Get("s", "k0"); ok {
		t.Fatal("expected earliest written key to be evicted")
	}
	for _, key := range []string{"k1", "k2", "new"} {
		if _, ok := store.Get("s", key); !ok {
			t.Fatalf("expected %q to survive eviction", key)
		}
	}
	mustConsistent(t, store)

	if record, claimed := store.Reserve("s", "k0", future); !claimed || record.Status != StatusProcessing {
		t.Fatalf("expected evicted key to be claimable again, claimed=%v status=%q", claimed, record.Status)
	}
	mustConsistent(t, store)
}

// 容量满时优先物理清理过期记录（不同端点 TTL 不同，过期条目未必在队首）：
// 已过期记录先被移除，有效记录不因这次插入而被淘汰。
func TestInMemoryStorePrefersExpiredOnOverflow(t *testing.T) {
	store := newInMemoryStore(2)
	now := time.Now()
	future := now.Add(time.Hour)
	store.Save(Record{Key: "expired", Scope: "s", Status: StatusCompleted, ExpiresAt: now.Add(-time.Minute)})
	store.Save(Record{Key: "live", Scope: "s", Status: StatusCompleted, ExpiresAt: future})
	// 触发容量检查：过期记录被物理清理，两个有效记录都在。
	store.Save(Record{Key: "fresh", Scope: "s", Status: StatusCompleted, ExpiresAt: future})

	if _, ok := store.Get("s", "expired"); ok {
		t.Fatal("expected expired record to be swept instead of a live record evicted")
	}
	for _, key := range []string{"live", "fresh"} {
		if _, ok := store.Get("s", key); !ok {
			t.Fatalf("expected live record %q to survive overflow cleanup", key)
		}
	}
	mustConsistent(t, store)
}

// Delete/Sweep 必须同时摘除 FIFO 链表节点，否则淘汰顺序结构会在长期运行中
// 无限积累（上限就失效了）；删除后的 key 再次写入应复用同一条链且不重复。
func TestInMemoryStoreEvictionBookkeepingStaysBounded(t *testing.T) {
	store := newInMemoryStore(2)
	now := time.Now()
	future := now.Add(time.Hour)
	store.Save(Record{Key: "a", Scope: "s", Status: StatusCompleted, ExpiresAt: future})
	store.Save(Record{Key: "b", Scope: "s", Status: StatusCompleted, ExpiresAt: future})
	store.Delete("s", "a")
	mustConsistent(t, store)

	store.Save(Record{Key: "a", Scope: "s", Status: StatusProcessing, ExpiresAt: future})
	mustConsistent(t, store)

	store.Save(Record{Key: "expired", Scope: "s", Status: StatusCompleted, ExpiresAt: now.Add(-time.Second)})
	mustConsistent(t, store)

	// Sweep 之后链表与 map 必须一起缩小（过期记录 + 被淘汰记录都不能残留）。
	store.Sweep(now)
	mustConsistent(t, store)
	if records, _, _ := storeSizes(store); records > 2 {
		t.Fatalf("expected store to stay within capacity after sweep, got %d", records)
	}
}

// 幂等语义回归：容量未满时，过期记录仍按「曾存在但已过期」返回 StatusExpired
// （22001），不能被容量逻辑改成「从未存在」。
func TestInMemoryStoreExpirySemanticsUnaffectedByCapacity(t *testing.T) {
	store := newInMemoryStore(maxInMemoryRecords)
	past := time.Now().Add(-time.Minute)
	store.Save(Record{Key: "expired", Scope: "s", Status: StatusCompleted, ExpiresAt: past})
	record, ok := store.Get("s", "expired")
	if !ok || record.Status != StatusExpired {
		t.Fatalf("expected StatusExpired tombstone, ok=%v status=%q", ok, record.Status)
	}
	if record, claimed := store.Reserve("s", "expired", time.Now().Add(time.Hour)); claimed || record.Status != StatusExpired {
		t.Fatalf("expected reserve to report expired (22001), claimed=%v status=%q", claimed, record.Status)
	}
}

// 并发安全（-race）：多 goroutine 同时 Reserve/Save/Get/Delete/Sweep 时，
// 容量上限与 records/order/elems 的一致性不得被破坏。
func TestInMemoryStoreCapacityConcurrentSafety(t *testing.T) {
	const capacity = 32
	store := newInMemoryStore(capacity)
	future := time.Now().Add(time.Hour)

	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				key := fmt.Sprintf("w%d-k%d", worker, i)
				if record, claimed := store.Reserve("s", key, future); claimed {
					record.Status = StatusCompleted
					store.Save(record)
				}
				store.Get("s", key)
				if i%7 == 0 {
					store.Delete("s", key)
				}
				if i%13 == 0 {
					store.Sweep(time.Now())
				}
			}
		}(worker)
	}
	wg.Wait()

	if records, order, elems := storeSizes(store); records > capacity || records != order || records != elems {
		t.Fatalf("capacity/bookkeeping violated after concurrent use: records=%d order=%d elems=%d capacity=%d",
			records, order, elems, capacity)
	}
}
