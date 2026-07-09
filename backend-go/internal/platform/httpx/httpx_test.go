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
		t.Fatalf("unexpected error for empty object: %v", err)
	}
	// Per api_spec_base.md, empty-object endpoints must silently ignore unknown
	// fields (SDK telemetry/gateway-injected keys), so {"x":1} is accepted.
	withUnknown := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"x":1,"telemetry":"abc"}`))
	if err := RequireEmptyObject(withUnknown); err != nil {
		t.Fatalf("unknown fields should be ignored, got %v", err)
	}
	// Non-object bodies (arrays, strings, numbers) are rejected.
	for _, body := range []string{`[]`, `"str"`, `42`} {
		bad := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		if err := RequireEmptyObject(bad); err == nil || err.Code != 10000 {
			t.Fatalf("body %q should be rejected as non-object, got %v", body, err)
		}
	}
}

func TestRecovererReturnsEnvelopeOnPanic(t *testing.T) {
	handler := Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 envelope, got status %d", rr.Code)
	}
	var resp ApiResponse[struct{}]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 10600 {
		t.Fatalf("expected code 10600, got %d", resp.Code)
	}
	if strings.Contains(rr.Body.String(), "runtime/debug") || strings.Contains(rr.Body.String(), "goroutine") {
		t.Fatalf("response leaked stack trace: %s", rr.Body.String())
	}
}

func TestNotFoundHandlerReturnsEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	NotFoundHandler(rr, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp ApiResponse[struct{}]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 10300 {
		t.Fatalf("expected code 10300, got %d", resp.Code)
	}
}

func TestLimitRequestBodyRejectsOversize(t *testing.T) {
	handler := LimitRequestBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Attempting to read past the limit must surface http.MaxBytesError.
		buf := make([]byte, MaxRequestBodyBytes+1)
		if _, err := r.Body.Read(buf); err == nil {
			t.Fatal("expected error reading oversize body, got nil")
		}
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", MaxRequestBodyBytes+1024)))
	handler.ServeHTTP(rr, req)
}
