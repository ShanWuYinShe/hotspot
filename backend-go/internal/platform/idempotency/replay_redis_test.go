package idempotency

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/requestmeta"
)

// 回放中间件 + RedisStore 的组合是多实例部署的真实生产路径（单实例用
// InMemoryStore，已有 replay_test.go 覆盖）。上一轮修复了 Redis 记录过期
// 即删键导致旧 key 静默重新执行的问题；本文件锁定「过期命中 → 22001」
// 必须通过完整 ReplayOrStore 链路成立，而不只是 Store 单测。
func withRedisStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewRedisStore(rdb)
	SetStore(store)
	t.Cleanup(func() { SetStore(NewInMemoryStore()) })
	return store, mr
}

func newReplayRequest(key string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a"}`))
	return req.WithContext(WithKey(req.Context(), key))
}

func replayCode(t *testing.T, recorder *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var body struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return body.Code
}

func TestReplayOrStoreRedisCachesAndReplays(t *testing.T) {
	withRedisStore(t)
	key := "redisreplay1234567890abcdefghij"
	req := newReplayRequest(key)

	calls := 0
	first := httptest.NewRecorder()
	ReplayOrStore(first, req, "redis/scope", func() httpx.ApiResponse[dummyResult] {
		calls++
		http.SetCookie(first, &http.Cookie{Name: "session", Value: "s1", Path: "/", HttpOnly: true})
		return httpx.Success(dummyResult{Value: "first"})
	})
	if calls != 1 || replayCode(t, first) != httpx.CodeOK {
		t.Fatalf("first call calls=%d code=%d", calls, replayCode(t, first))
	}

	// 相同请求体 + 相同 key：命中 Redis 完成态记录，producer 不执行、Set-Cookie 重发。
	second := httptest.NewRecorder()
	ReplayOrStore(second, newReplayRequest(key), "redis/scope", func() httpx.ApiResponse[dummyResult] {
		calls++
		return httpx.Success(dummyResult{Value: "second"})
	})
	if calls != 1 {
		t.Fatalf("replay must not run producer, calls=%d", calls)
	}
	var replayed struct {
		Data dummyResult `json:"data"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &replayed); err != nil {
		t.Fatalf("decode replayed body: %v", err)
	}
	if replayed.Data.Value != "first" {
		t.Fatalf("replayed data = %#v, want first", replayed.Data)
	}
	if cookies := second.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != "session" {
		t.Fatalf("replayed cookies = %+v", cookies)
	}
}

func TestReplayOrStoreRedisBodyMismatchConflicts(t *testing.T) {
	withRedisStore(t)
	key := "redisconflict1234567890abcdefgh"
	first := httptest.NewRecorder()
	ReplayOrStore(first, newReplayRequest(key), "redis/scope", func() httpx.ApiResponse[dummyResult] {
		return httpx.Success(dummyResult{Value: "ok"})
	})
	if replayCode(t, first) != httpx.CodeOK {
		t.Fatalf("first code = %d", replayCode(t, first))
	}

	// 同一 key 换请求体：必须 22000（幂等上下文不匹配），不得回放首次响应。
	other := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"b"}`))
	other = other.WithContext(WithKey(other.Context(), key))
	mismatch := httptest.NewRecorder()
	ran := false
	ReplayOrStore(mismatch, other, "redis/scope", func() httpx.ApiResponse[dummyResult] {
		ran = true
		return httpx.Success(dummyResult{Value: "should-not-run"})
	})
	if ran {
		t.Fatal("body mismatch must not run producer")
	}
	if code := replayCode(t, mismatch); code != httpx.CodeIdempotencyConflict {
		t.Fatalf("body mismatch code = %d, want %d", code, httpx.CodeIdempotencyConflict)
	}
}

func TestReplayOrStoreRedisExpiredReturns22001(t *testing.T) {
	store, _ := withRedisStore(t)
	key := "redisexpired1234567890abcdefgh"
	req := newReplayRequest(key)
	// 模拟「24h 前写入、已过期」的记录：按中间件的 scopedKey 口径预置一条 ExpiresAt 已过的记录
	// （业务 TTL 由 ReplayOrStore 固定 24h，无法在测试里真实等待）。
	scopedKey := "ip:" + requestmeta.ClientIP(req) + "::" + key
	past := time.Now().Add(-time.Minute)
	if _, claimed := store.Reserve("redis/scope", scopedKey, past); !claimed {
		t.Fatal("seed expired record: reserve must claim")
	}

	expired := httptest.NewRecorder()
	ran := false
	ReplayOrStore(expired, req, "redis/scope", func() httpx.ApiResponse[dummyResult] {
		ran = true
		return httpx.Success(dummyResult{Value: "new"})
	})
	if ran {
		t.Fatal("expired key must not re-execute request")
	}
	if code := replayCode(t, expired); code != httpx.CodeIdempotencyExpired {
		t.Fatalf("expired code = %d, want %d (22001)", code, httpx.CodeIdempotencyExpired)
	}
}

func TestReplayOrStoreRedisPanicRollsBackKey(t *testing.T) {
	withRedisStore(t)
	key := "redispanic1234567890abcdefghij"
	req := newReplayRequest(key)

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected producer panic to propagate")
			}
		}()
		ReplayOrStore(httptest.NewRecorder(), req, "redis/scope", func() httpx.ApiResponse[dummyResult] {
			panic("boom")
		})
	}()

	// panic 回滚必须硬删除 processing 记录：同一 key 可立即重新执行（不返回 22002）。
	retry := httptest.NewRecorder()
	ran := false
	ReplayOrStore(retry, newReplayRequest(key), "redis/scope", func() httpx.ApiResponse[dummyResult] {
		ran = true
		return httpx.Success(dummyResult{Value: "retried"})
	})
	if !ran {
		t.Fatal("after panic rollback the same key must be retryable")
	}
	if code := replayCode(t, retry); code != httpx.CodeOK {
		t.Fatalf("retry code = %d", code)
	}
}
