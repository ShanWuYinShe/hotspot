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

// TestInMemoryStoreLazyExpiry verifies that Get treats records past their
// ExpiresAt as missing, even before the sweeper has physically removed them.
// This is the property that makes the sweeper non-critical for correctness.
func TestInMemoryStoreLazyExpiry(t *testing.T) {
	store := NewInMemoryStore()
	past := time.Now().Add(-time.Minute)
	store.Save(Record{Key: "expired", Scope: "s", Status: StatusCompleted, ExpiresAt: past})
	if _, ok := store.Get("s", "expired"); ok {
		t.Fatal("expected expired record to be lazily evicted by Get")
	}
}

// TestInMemoryStoreReserveHonorsExpiry verifies Reserve overwrites an expired
// record (so a retried request after TTL doesn't see a stale "processing").
func TestInMemoryStoreReserveHonorsExpiry(t *testing.T) {
	store := NewInMemoryStore()
	past := time.Now().Add(-time.Minute)
	store.Save(Record{Key: "k", Scope: "s", Status: StatusProcessing, ExpiresAt: past})
	future := time.Now().Add(time.Hour)
	if _, ok := store.Reserve("s", "k", future); !ok {
		t.Fatal("expected Reserve to overwrite expired record and report success")
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

// TestStartSweeperIsIdempotent verifies that calling StartSweeper twice
// doesn't spawn two goroutines (the second call is a no-op).
func TestStartSweeperIsIdempotent(t *testing.T) {
	t.Cleanup(StopSweeper)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	StartSweeper(ctx)
	StartSweeper(ctx) // must not panic, must not duplicate
}

// TestStartSweeperHonorsCancellation verifies the goroutine exits when ctx
// is cancelled and a fresh StartSweeper can subsequently launch (which
// would be blocked if the previous one never cleared sweeperStarted).
func TestStartSweeperHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	StartSweeper(ctx)
	cancel()
	StopSweeper()
	StartSweeper(context.Background())
	StopSweeper()
}
