package idempotency

import (
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
