package media

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

func newMountedMediaModule(t *testing.T) (root http.Handler, public http.Handler) {
	t.Helper()
	module := NewModule(config.Config{
		S3Endpoint:       "http://seaweedfs:8333",
		S3Bucket:         "hotspot",
		S3Region:         "us-east-1",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	}, nil)
	rootRouter := chi.NewRouter()
	module.Mount(rootRouter)
	publicRouter := chi.NewRouter()
	module.MountPublic(publicRouter)
	return rootRouter, publicRouter
}

// Mount 的真实挂载：/api/v1/media/* 必须由 chi 命中（未 404），且未注入认证
// 依赖时 RequiredAuth fail-closed 返回 10600（安全边界）。
func TestMediaModuleMountsAuthenticatedRoutes(t *testing.T) {
	root, _ := newMountedMediaModule(t)
	for _, path := range []string{"/api/v1/media/presign_upload", "/api/v1/media/complete_upload"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
		rr := httptest.NewRecorder()
		root.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: envelope response must be HTTP 200, got %d", path, rr.Code)
		}
		if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
			t.Fatalf("%s: unwired auth deps must fail closed with 10600, got %d", path, code)
		}
	}
}

// MountPublic 的匿名头像代理路由：命中处理器后由 service 判定文件不存在，
// 返回统一 404 envelope（而非路由层 404）。
func TestMediaModuleMountsPublicAvatarProxy(t *testing.T) {
	_, public := newMountedMediaModule(t)
	req := httptest.NewRequest(http.MethodGet, "/media/avatars/file_mediacov_missing", nil)
	rr := httptest.NewRecorder()
	public.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("avatar proxy envelope must be HTTP 200, got %d", rr.Code)
	}
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("expected CodeNotFound for unwired/unknown avatar, got %d", code)
	}
}
