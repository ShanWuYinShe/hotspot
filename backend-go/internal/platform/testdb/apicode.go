package testdb

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// DecodeAPICode decodes the business error code (httpx.Code) from a handler
// test response body. Shared by all modules' handler coverage tests; keep
// module-local decode helpers out of the codebase.
func DecodeAPICode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.Code
}
