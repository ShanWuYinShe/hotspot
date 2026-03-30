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
	handler := NewHandler(NewServiceWithRepo(&stubRepository{
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

func TestSuggestRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestRejectsEmptyPrefix(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"","reference_point":{"latitude":1,"longitude":2}}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestRejectsMissingReferencePoint(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot"}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestRejectsShortRadius(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":1,"longitude":2},"radius":99}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestRejectsTooLargeRadius(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":1,"longitude":2},"radius":50001}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestPassesTypesToService(t *testing.T) {
	repo := &stubRepository{response: SearchSuggestResponse{Items: []SearchSuggestItem{}}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":1,"longitude":2},"types":["user","group"]}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	if len(repo.lastReq.Types) != 2 || repo.lastReq.Types[0] != "user" || repo.lastReq.Types[1] != "group" {
		t.Fatalf("expected types passthrough, got %#v", repo.lastReq.Types)
	}
}

func TestSuggestReturnsServiceUnavailableOnRepositoryError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: errors.New("boom")}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":1,"longitude":2}}`))
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
