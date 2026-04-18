package idempotency

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
		Key: "inprogress", Scope: "test_scope",
		Status:    StatusProcessing,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
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
