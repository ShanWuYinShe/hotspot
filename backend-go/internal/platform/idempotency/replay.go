package idempotency

import (
	"encoding/json"
	"net/http"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

var defaultStore Store = NewInMemoryStore()

func SetStore(store Store) {
	if store != nil {
		defaultStore = store
	}
}

func ReplayOrStore[T any](w http.ResponseWriter, r *http.Request, scope string, producer func() httpx.ApiResponse[T]) bool {
	key := KeyFromContext(r.Context())
	if key == "" {
		payload := producer()
		httpx.WriteJSON(w, http.StatusOK, payload)
		return true
	}
	if record, ok := defaultStore.Get(scope, key); ok && record.Status == StatusCompleted {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(record.Response)
		return true
	}
	payload := producer()
	blob, _ := json.Marshal(payload)
	defaultStore.Save(Record{Key: key, Scope: scope, Status: StatusCompleted, Response: append(blob, '\n'), ExpiresAt: time.Now().Add(24 * time.Hour)})
	httpx.WriteJSON(w, http.StatusOK, payload)
	return true
}
