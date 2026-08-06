package search

import (
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/httpx"
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
		Code httpx.Code            `json:"code"`
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
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestRejectsEmptyPrefix(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"","reference_point":{"latitude":1,"longitude":2}}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestAcceptsMissingReferencePoint(t *testing.T) {
	// reference_point 现在可选:缺失时服务端按回退链解析(见 api_spec_types.md)
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot"}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("missing reference_point should be accepted (fallback), got code %d", resp.Code)
	}
}

func TestSuggestAcceptsPartialReferencePoint(t *testing.T) {
	// 部分 reference_point(缺经度或纬度)也走回退,不报错
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	cases := []string{
		`{"prefix":"hot","reference_point":{}}`,
		`{"prefix":"hot","reference_point":{"latitude":1}}`,
		`{"prefix":"hot","reference_point":{"longitude":2}}`,
	}

	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.Suggest(rr, req)

		var resp struct {
			Code httpx.Code `json:"code"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Code != 0 {
			t.Fatalf("partial reference_point should be accepted (fallback) for %s, got code %d", body, resp.Code)
		}
	}
}

func TestSuggestRejectsInvalidReferencePoint(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":999,"longitude":2}}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestRejectsShortRadius(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":1,"longitude":2},"radius":99}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSuggestAllowsTooLargeRadiusForServiceClamp(t *testing.T) {
	repo := &stubRepository{response: SearchSuggestResponse{Items: []SearchSuggestItem{}}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/suggest", strings.NewReader(`{"prefix":"hot","reference_point":{"latitude":1,"longitude":2},"radius":50001}`))
	rr := httptest.NewRecorder()
	handler.Suggest(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if repo.lastReq.Radius != 50000 {
		t.Fatalf("expected service clamp to 50000, got %#v", repo.lastReq)
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
		Code    httpx.Code `json:"code"`
		Message string     `json:"message"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInternalUnavailable {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
