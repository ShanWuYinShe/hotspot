package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestNewHTTPRouterMountsSystemRoutes(t *testing.T) {
	router := NewHTTPRouter(BuildModules(config.Load(), persistence.NewFactory(config.Load())))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/get_meta", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected code: %d", resp.Code)
	}
	if resp.Data.Version != "v1" {
		t.Fatalf("unexpected version: %s", resp.Data.Version)
	}
}
