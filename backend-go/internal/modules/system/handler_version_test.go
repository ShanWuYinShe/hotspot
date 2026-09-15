package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/buildinfo"
	"hotspot/backend-go/internal/platform/httpx"
)

// TestHealthHandlerStampsBuildInfo 验证 health 响应携带构建期注入的版本指纹，
// 且收尾恢复全局变量不污染其他测试。
func TestHealthHandlerStampsBuildInfo(t *testing.T) {
	origVersion, origCommit, origBuildTime := buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime
	buildinfo.Version = "v1.0.0"
	buildinfo.Commit = "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b"
	buildinfo.BuildTime = "2026-09-16T00:00:00Z"
	defer func() {
		buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime = origVersion, origCommit, origBuildTime
	}()

	ts := time.Now().UTC().Format(time.RFC3339)
	health := &HealthStatus{Status: SystemHealthHealthy, Timestamp: ts}
	service := NewServiceWithDeps(&stubRepo{healthCode: 0, health: health}, &stubProvider{})
	handler := NewHandler(service)

	rr := httptest.NewRecorder()
	handler.Health(rr, httptest.NewRequest(http.MethodPost, "/health", strings.NewReader(`{}`)))
	var resp httpx.ApiResponse[HealthStatus]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.Version != "v1.0.0" {
		t.Fatalf("expected version v1.0.0, got %q", resp.Data.Version)
	}
	if resp.Data.Commit != "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b" {
		t.Fatalf("expected injected commit, got %q", resp.Data.Commit)
	}
	if resp.Data.BuildTime != "2026-09-16T00:00:00Z" {
		t.Fatalf("expected injected build time, got %q", resp.Data.BuildTime)
	}
}

// TestHealthHandlerDevBuildKeepsEmptyFields：本地裸构建（未注入）时版本字段
// 仍随契约下发（version=dev，commit/build_time 空串），不缺字段。
func TestHealthHandlerDevBuildKeepsEmptyFields(t *testing.T) {
	origVersion, origCommit, origBuildTime := buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime
	buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime = "dev", "", ""
	defer func() {
		buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime = origVersion, origCommit, origBuildTime
	}()

	health := &HealthStatus{Status: SystemHealthHealthy, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	service := NewServiceWithDeps(&stubRepo{healthCode: 0, health: health}, &stubProvider{})
	handler := NewHandler(service)

	rr := httptest.NewRecorder()
	handler.Health(rr, httptest.NewRequest(http.MethodPost, "/health", strings.NewReader(`{}`)))
	var resp httpx.ApiResponse[HealthStatus]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Version != "dev" || resp.Data.Commit != "" || resp.Data.BuildTime != "" {
		t.Fatalf("expected dev defaults, got %#v", resp.Data)
	}
}
