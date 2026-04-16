package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIErrorError(t *testing.T) {
	err := &APIError{Code: 10000, Message: "参数错误"}
	if got := err.Error(); got != "api error 10000: 参数错误" {
		t.Fatalf("unexpected error string: %s", got)
	}
}

func TestSuccessAndErrorHelpers(t *testing.T) {
	success := Success(map[string]any{"ok": true})
	if success.Code != 0 || success.Data == nil || (*success.Data)["ok"] != true {
		t.Fatalf("unexpected success payload: %#v", success)
	}
	empty := SuccessEmpty()
	if empty.Code != 0 || empty.Data == nil {
		t.Fatalf("unexpected empty success payload: %#v", empty)
	}
	failure := Error(10300, "不存在")
	if failure.Code != 10300 || failure.Message != "不存在" {
		t.Fatalf("unexpected error payload: %#v", failure)
	}
}

func TestWriteJSONWritesStatusHeadersAndBody(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusAccepted, Success(map[string]string{"id": "1"}))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	var resp ApiResponse[map[string]string]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 0 || resp.Data == nil || (*resp.Data)["id"] != "1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestDecodeStrictJSONRejectsUnknownFields(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok","extra":1}`))
	var out payload
	if err := DecodeStrictJSON(req, &out); err == nil || err.Code != 10000 {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestDecodeStrictJSONRejectsTrailingJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"} {}`))
	var out payload
	if err := DecodeStrictJSON(req, &out); err == nil || err.Code != 10000 {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestDecodeStrictJSONHappyPath(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	var out payload
	if err := DecodeStrictJSON(req, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "ok" {
		t.Fatalf("expected ok, got %#v", out)
	}
}

func TestRequireEmptyObject(t *testing.T) {
	good := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	if err := RequireEmptyObject(good); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bad := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"x":1}`))
	if err := RequireEmptyObject(bad); err == nil || err.Code != 10000 {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
