package idempotency

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

func TestReplayOrStoreCachesResponse(t *testing.T) {
	store := NewInMemoryStore()
	SetStore(store)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req = req.WithContext(WithKey(req.Context(), "abcd1234abcd1234abcd1234abcd1234"))
	rr1 := httptest.NewRecorder()
	count := 0
	ReplayOrStore(rr1, req, "scope", func() httpx.ApiResponse[map[string]string] {
		count++
		return httpx.Success(map[string]string{"value": "ok"})
	})
	rr2 := httptest.NewRecorder()
	ReplayOrStore(rr2, req, "scope", func() httpx.ApiResponse[map[string]string] {
		count++
		return httpx.Success(map[string]string{"value": "changed"})
	})
	if count != 1 {
		t.Fatalf("expected single producer call, got %d", count)
	}
	var first map[string]any
	var second map[string]any
	_ = json.Unmarshal(rr1.Body.Bytes(), &first)
	_ = json.Unmarshal(rr2.Body.Bytes(), &second)
	if first["data"].(map[string]any)["value"] != second["data"].(map[string]any)["value"] {
		t.Fatalf("expected replayed response, got %v vs %v", first, second)
	}
}
