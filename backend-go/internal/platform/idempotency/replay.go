package idempotency

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

// defaultStore is read on every request and written once at boot, so guard it
// with an atomic pointer instead of a bare global (which would race under
// -race and on reconfigure).
var defaultStore atomic.Pointer[InMemoryStore]

func init() {
	defaultStore.Store(NewInMemoryStore())
}

// currentStore returns the active store. Reserved for future swap-in of a
// distributed store; today it's always the in-memory one.
func currentStore() Store { return defaultStore.Load() }

func SetStore(store Store) {
	if s, ok := store.(*InMemoryStore); ok && s != nil {
		defaultStore.Store(s)
	}
}

func ReplayOrStore[T any](w http.ResponseWriter, r *http.Request, scope string, producer func() httpx.ApiResponse[T]) bool {
	key := KeyFromContext(r.Context())
	if key == "" {
		payload := producer()
		httpx.WriteJSON(w, http.StatusOK, payload)
		return true
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	store := currentStore()

	// Reserve atomically: returns (record, true) when we claimed it (now
	// processing), or (existing, false) when a record already exists.
	existing, claimed := store.Reserve(scope, key, expiresAt)
	if !claimed {
		if existing.Status == StatusCompleted {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(existing.Response)
			return true
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeIdempotencyInFlight, "请求正在处理中，请稍后重试"))
		return true
	}

	payload := producer()
	blob, _ := json.Marshal(payload)
	store.Save(Record{Key: key, Scope: scope, Status: StatusCompleted, Response: append(blob, '\n'), ExpiresAt: expiresAt})
	httpx.WriteJSON(w, http.StatusOK, payload)
	return true
}
