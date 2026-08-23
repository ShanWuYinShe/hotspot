package idempotency

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

type dummyResult struct{ Value string }

func TestReplayOrStoreCachesResponse(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req = req.WithContext(WithKey(req.Context(), "abcd1234abcd1234abcd1234abcd1234"))
	rr1 := httptest.NewRecorder()
	count := 0
	ReplayOrStore(rr1, req, "scope", func() httpx.ApiResponse[dummyResult] {
		count++
		return httpx.Success(dummyResult{Value: "ok"})
	})
	rr2 := httptest.NewRecorder()
	ReplayOrStore(rr2, req, "scope", func() httpx.ApiResponse[dummyResult] {
		count++
		return httpx.Success(dummyResult{Value: "changed"})
	})
	if count != 1 {
		t.Fatalf("expected single producer call, got %d", count)
	}
	var first map[string]any
	var second map[string]any
	_ = json.Unmarshal(rr1.Body.Bytes(), &first)
	_ = json.Unmarshal(rr2.Body.Bytes(), &second)
	if first["data"].(map[string]any)["Value"] != second["data"].(map[string]any)["Value"] {
		t.Fatalf("expected replayed response")
	}
}

func TestReplayOrStoreEmptyKeyExecutesProducer(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	rr := httptest.NewRecorder()
	count := 0
	ReplayOrStore(rr, req, "scope", func() httpx.ApiResponse[dummyResult] {
		count++
		return httpx.Success(dummyResult{Value: "yes"})
	})
	if count != 1 {
		t.Fatalf("expected producer called once for empty key, got %d", count)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["code"].(float64) != 0 {
		t.Fatalf("expected success, got %v", body)
	}
}

// spec api_spec_base.md「幂等性」：仅缓存成功响应（code == 0）。
// 失败响应不得缓存——同一 key 的重试必须真正再次执行 producer。
func TestReplayOrStoreDoesNotCacheFailure(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req = req.WithContext(WithKey(req.Context(), "fail0123fail0123fail0123fail0123"))
	rr1 := httptest.NewRecorder()
	calls := 0
	ReplayOrStore(rr1, req, "failure_scope", func() httpx.ApiResponse[dummyResult] {
		calls++
		return httpx.ApiResponse[dummyResult]{Code: httpx.CodeInvalidOldCredential, Message: "验证码错误或已过期"}
	})
	if calls != 1 {
		t.Fatalf("expected first producer call, got %d", calls)
	}
	var first map[string]any
	_ = json.Unmarshal(rr1.Body.Bytes(), &first)
	if first["code"].(float64) != float64(httpx.CodeInvalidOldCredential) {
		t.Fatalf("expected failure envelope on first call, got %v", first)
	}
	// 重试：producer 必须再次执行（失败未被缓存），且本次成功可被缓存。
	rr2 := httptest.NewRecorder()
	ReplayOrStore(rr2, req, "failure_scope", func() httpx.ApiResponse[dummyResult] {
		calls++
		return httpx.Success(dummyResult{Value: "recovered"})
	})
	if calls != 2 {
		t.Fatalf("expected producer to re-execute after cached-failure, got %d calls", calls)
	}
	// 第三次：命中第二次的成功缓存。
	rr3 := httptest.NewRecorder()
	ReplayOrStore(rr3, req, "failure_scope", func() httpx.ApiResponse[dummyResult] {
		calls++
		return httpx.Success(dummyResult{Value: "should-not-run"})
	})
	if calls != 2 {
		t.Fatalf("expected success replay on third call, got %d calls", calls)
	}
	var third map[string]any
	_ = json.Unmarshal(rr3.Body.Bytes(), &third)
	if third["data"].(map[string]any)["Value"] != "recovered" {
		t.Fatalf("expected replayed success payload, got %v", third)
	}
}

func TestReplayOrStoreConcurrentRaceCondition(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	key := "concurrentkey1234567890123456789012"
	reqBase := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	reqBase = reqBase.WithContext(WithKey(reqBase.Context(), key))

	var mu sync.Mutex
	producerCalls := 0

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/", http.NoBody).WithContext(WithKey(reqBase.Context(), key))
			rr := httptest.NewRecorder()
			ReplayOrStore(rr, req, "race_scope", func() httpx.ApiResponse[dummyResult] {
				mu.Lock()
				producerCalls++
				mu.Unlock()
				return httpx.Success(dummyResult{Value: "call"})
			})
		}()
	}
	wg.Wait()
	if producerCalls != 1 {
		t.Errorf("expected exactly 1 producer call (race condition), got %d", producerCalls)
	}
}

func TestMiddlewareReplaysCompleted(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.Header.Set("Idempotency-Key", "middleware1234567890123456789012345")

	handlerCalls := 0
	producerCalls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalls++
		ReplayOrStore(w, r, "test_scope", func() httpx.ApiResponse[dummyResult] {
			producerCalls++
			return httpx.Success(dummyResult{Value: "yes"})
		})
	})
	mw := Middleware(handler)

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if producerCalls != 1 {
		t.Fatalf("first request: expected producer called once, got %d", producerCalls)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr.Code)
	}

	producerCalls = 0
	handlerCalls = 0
	rr2 := httptest.NewRecorder()
	mw.ServeHTTP(rr2, req)
	if producerCalls != 0 {
		t.Fatalf("second request: expected producer NOT called (replayed), got %d", producerCalls)
	}
	if handlerCalls != 1 {
		t.Fatalf("second request: handler called %d times", handlerCalls)
	}
	var body map[string]any
	_ = json.Unmarshal(rr2.Body.Bytes(), &body)
	if body["code"].(float64) != 0 {
		t.Fatalf("expected success replay, got %v", body)
	}
}

func TestMiddlewareNoKeyPassesThrough(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(dummyResult{Value: "no_key"}))
	})
	mw := Middleware(handler)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if !called {
		t.Fatal("handler should be called when no idempotency key present")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMiddlewareInvalidKeyPassesThrough(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.Header.Set("Idempotency-Key", "too_short")
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(dummyResult{Value: "invalid_key_handled"}))
	})
	mw := Middleware(handler)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if !called {
		t.Fatal("handler should be called when invalid idempotency key present")
	}
}

func TestReplayOrStoreProcessingStatus22002(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	store.Save(Record{
		Key: "u:u1::inprogress", Scope: "test_scope",
		Status:    StatusProcessing,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))
	req = req.WithContext(WithKey(req.Context(), "inprogress"))
	rr := httptest.NewRecorder()
	producerCalled := false
	ReplayOrStore(rr, req, "test_scope", func() httpx.ApiResponse[dummyResult] {
		producerCalled = true
		return httpx.Success(dummyResult{Value: "executed"})
	})
	if producerCalled {
		t.Fatal("producer should NOT be called when StatusProcessing")
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["code"].(float64) != 22002 {
		t.Fatalf("expected 22002 (processing), got %v", body["code"])
	}
}
