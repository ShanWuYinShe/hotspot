package feed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListTrendingReturnsBundle(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/list_trending", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.ListTrending(rr, req)

	var resp struct {
		Code int            `json:"code"`
		Data TrendingBundle `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.TrendingTags) == 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
