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

	expiresAt := time.Now().Add(24 * time.Hour)
	scopeKey := scope + "::" + key

	store := defaultStore.(*InMemoryStore)
	store.mu.Lock()
	if existing, ok := store.records[scopeKey]; ok {
		store.mu.Unlock()
		if existing.Status == StatusCompleted {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(existing.Response)
			return true
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(22002, "请求正在处理中，请稍后重试"))
		return true
	}
	store.records[scopeKey] = Record{
		Key: key, Scope: scope, Status: StatusProcessing,
		ExpiresAt: expiresAt,
	}
	store.mu.Unlock()

	payload := producer()
	blob, _ := json.Marshal(payload)
	defaultStore.Save(Record{Key: key, Scope: scope, Status: StatusCompleted, Response: append(blob, '\n'), ExpiresAt: expiresAt})
	httpx.WriteJSON(w, http.StatusOK, payload)
	return true
}
