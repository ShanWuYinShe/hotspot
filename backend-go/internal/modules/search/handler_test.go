package search

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuggestReturnsItems(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(stubRepository{
		response: SearchSuggestResponse{Items: []SearchSuggestItem{{Type: "tag", Payload: map[string]any{"name": "hot"}}}},
	}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":1,"longitude":2}}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code int                   `json:"code"`
		Data SearchSuggestResponse `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) == 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestReturnsServiceUnavailableOnRepositoryError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(stubRepository{err: errors.New("boom")}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot"}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

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
