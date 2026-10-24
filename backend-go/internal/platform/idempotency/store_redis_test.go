package idempotency

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// RedisStore 是多实例部署下的生产实现（单实例走 InMemoryStore）。本文件锁定
// 其与内存实现必须一致的语义：原子占位、完成态回放、过期命中 22001（墓碑）、
// fail-open，以及 Lua 覆写守卫。
func newTestRedisStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRedisStore(rdb), mr
}

func TestRedisStoreReserveClaimsExactlyOnce(t *testing.T) {
	store, _ := newTestRedisStore(t)
	expiresAt := time.Now().Add(time.Hour)

	claimed, ok := store.Reserve("ep", "k1", expiresAt)
	if !ok || claimed.Status != StatusProcessing || claimed.Key != "k1" || claimed.Scope != "ep" {
		t.Fatalf("first reserve = %#v claimed=%v", claimed, ok)
	}
	// 重放：同一 key 第二次 Reserve 必须看到已有记录且不再声明执行权。
	existing, ok := store.Reserve("ep", "k1", expiresAt)
	if ok {
		t.Fatal("second reserve must not claim")
	}
	if existing.Status != StatusProcessing || existing.ExpiresAt.IsZero() {
		t.Fatalf("existing = %#v", existing)
	}
	// scope 参与 key：不同 scope 的同一 key 互不影响。
	if _, ok := store.Reserve("green", "k1", expiresAt); !ok {
		t.Fatal("different scope must claim independently")
	}
}

func TestRedisStoreReserveIsAtomicUnderConcurrency(t *testing.T) {
	store, _ := newTestRedisStore(t)
	const workers = 24
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		wins int
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, claimed := store.Reserve("ep", "concurrent", time.Now().Add(time.Hour)); claimed {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("claimed %d times, want exactly 1", wins)
	}
}

func TestRedisStoreSaveCompletedAndReplayGuard(t *testing.T) {
	store, _ := newTestRedisStore(t)
	expiresAt := time.Now().Add(time.Hour)
	store.Reserve("ep", "k2", expiresAt)

	first := Record{
		Key:       "k2",
		Scope:     "ep",
		Status:    StatusCompleted,
		Response:  []byte("first-response"),
		ExpiresAt: expiresAt,
		BodyHash:  "hash-1",
		Headers:   map[string][]string{"Set-Cookie": {"a=1"}},
	}
	store.Save(first)
	got, ok := store.Get("ep", "k2")
	if !ok {
		t.Fatal("completed record must be readable")
	}
	if got.Status != StatusCompleted || string(got.Response) != "first-response" || got.BodyHash != "hash-1" {
		t.Fatalf("roundtrip = %#v", got)
	}
	if len(got.Headers["Set-Cookie"]) != 1 || got.Headers["Set-Cookie"][0] != "a=1" {
		t.Fatalf("headers = %#v", got.Headers)
	}
	if replay, ok := store.Reserve("ep", "k2", expiresAt); ok || replay.Status != StatusCompleted {
		t.Fatalf("reserve after completed = %#v claimed=%v", replay, ok)
	}

	// Lua 守卫：完成态记录不得被再次 Save 覆盖（与 InMemoryStore 语义一致）。
	store.Save(Record{Key: "k2", Scope: "ep", Status: StatusCompleted, Response: []byte("second"), ExpiresAt: expiresAt})
	if got, _ := store.Get("ep", "k2"); string(got.Response) != "first-response" {
		t.Fatalf("completed record overwritten: %q", got.Response)
	}
}

func TestRedisStoreExpiredRecordLooksLikeMemoryExpiry(t *testing.T) {
	store, mr := newTestRedisStore(t)
	// 业务有效期很短（200ms）：过期后墓碑窗口内仍应识别为 StatusExpired（22001），
	// 而不是像 TTL 直接删键那样被当成全新请求重新执行。
	expiresAt := time.Now().Add(200 * time.Millisecond)
	if _, ok := store.Reserve("ep", "k3", expiresAt); !ok {
		t.Fatal("initial reserve must claim")
	}
	time.Sleep(300 * time.Millisecond) // 真实时钟越过 ExpiresAt（墓碑键仍在）

	if got, ok := store.Get("ep", "k3"); !ok || got.Status != StatusExpired {
		t.Fatalf("expired get = %#v ok=%v", got, ok)
	}
	expired, ok := store.Reserve("ep", "k3", time.Now().Add(time.Hour))
	if ok || expired.Status != StatusExpired {
		t.Fatalf("expired reserve = %#v claimed=%v, want StatusExpired/not claimed", expired, ok)
	}

	// 超过墓碑窗口后键被 Redis 清理，此时才是真正"从未存在"→ 可重新占位。
	mr.FastForward(expiredRecordGrace + time.Minute)
	if _, ok := store.Reserve("ep", "k3", time.Now().Add(time.Hour)); !ok {
		t.Fatal("after tombstone GC reserve must claim fresh")
	}
}

func TestRedisStoreDeleteAllowsImmediateRetry(t *testing.T) {
	store, _ := newTestRedisStore(t)
	expiresAt := time.Now().Add(time.Hour)
	store.Reserve("ep", "k4", expiresAt)
	store.Delete("ep", "k4")
	if _, ok := store.Reserve("ep", "k4", expiresAt); !ok {
		t.Fatal("delete must allow immediate re-claim (panic rollback path)")
	}
	if _, ok := store.Get("ep", "missing"); ok {
		t.Fatal("missing record must not be found")
	}
}

func TestRedisStoreFailsOpenWhenBackendDown(t *testing.T) {
	store, mr := newTestRedisStore(t)
	mr.Close() // 后端宕机

	// fail-open：限流/幂等故障窗口内直通执行，不阻断全部写端点。
	rec, ok := store.Reserve("ep", "k5", time.Now().Add(time.Hour))
	if !ok || rec.Status != StatusProcessing {
		t.Fatalf("reserve while down = %#v claimed=%v (want fail-open claim)", rec, ok)
	}
	if _, ok := store.Get("ep", "k5"); ok {
		t.Fatal("get while down must miss")
	}
	// Save/Delete 与内存实现同样不返回错误（副作用尽力而为）。
	store.Save(Record{Key: "k5", Scope: "ep", Status: StatusCompleted, ExpiresAt: time.Now().Add(time.Hour)})
	store.Delete("ep", "k5")
}

func TestRedisStoreCallContextHasTimeout(t *testing.T) {
	// 防御性断言：redisCallCtx 必须带超时（避免 Redis 半死时请求挂满 TCP 超时）。
	ctx, cancel := redisCallCtx()
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 5*time.Second {
		t.Fatalf("redis call context deadline = %v ok=%v", deadline, ok)
	}
}
