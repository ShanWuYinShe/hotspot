package feed

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListTrendingReturnsBundle(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(stubRepository{
		listTrendingResult: TrendingBundle{TrendingTags: []map[string]any{{"name": "hot"}}},
	}))
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

func TestListTrendingReturnsServiceUnavailableOnRepositoryError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(stubRepository{err: errors.New("boom")}))
	req := httptest.NewRequest(http.MethodPost, "/list_trending", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.ListTrending(rr, req)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10600 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
