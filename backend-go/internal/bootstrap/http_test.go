package bootstrap

import (
	"encoding/json"
	"hotspot/backend-go/internal/platform/httpx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestNewHTTPRouterMountsSystemRoutes(t *testing.T) {
	cfg := config.Config{
		S3Endpoint:       "http://seaweedfs:8333",
		S3Bucket:         "hotspot",
		S3Region:         "us-east-1",
		S3AccessKey:      "minioadmin",
		S3SecretKey:      "minioadmin123",
		S3ForcePathStyle: true,
	}
	router := NewHTTPRouter(BuildModules(cfg, persistence.NewFactory(cfg)))
	req := httptest.NewRequest(http.MethodPost, "/public/v1/system/meta", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var resp struct {
		Code httpx.Code `json:"code"`
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
