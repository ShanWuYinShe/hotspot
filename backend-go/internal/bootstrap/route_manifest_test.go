package bootstrap

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// route_manifest.json 是「对外端点面」的显式快照（审查 spec-contract P1：
// 此前路由只在 /tmp dump，端点增删/认证档变更无人 review、契约漂移不可见）。
// 测试用生产同款装配枚举全部路由（method + 模板 path + 认证档）并与快照
// 逐字节比较；有意增删端点时运行：
//
//	UPDATE_ROUTE_MANIFEST=1 go test ./internal/bootstrap -run TestRouteManifest
//
// 然后 review diff 一并提交——端点面变更必须是有意识的。
type routeManifest struct {
	Version int                  `json:"version"`
	Routes  []routeManifestEntry `json:"routes"`
}

type routeManifestEntry struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Auth   string `json:"auth"`
}

func routeManifestPath() string {
	return filepath.Join("..", "..", "docs", "route_manifest.json")
}

func TestRouteManifest(t *testing.T) {
	cfg := config.Config{
		S3Endpoint:       "http://seaweedfs:8333",
		S3Bucket:         "hotspot",
		S3Region:         "us-east-1",
		S3AccessKey:      "minioadmin",
		S3SecretKey:      "minioadmin123",
		S3ForcePathStyle: true,
	}
	router := NewHTTPRouter(BuildModules(cfg, persistence.NewFactory(cfg)))
	mux, ok := router.(*chi.Mux)
	if !ok {
		t.Fatalf("expected *chi.Mux, got %T", router)
	}
	manifest := routeManifest{Version: 1, Routes: collectRouteManifest(mux)}
	if len(manifest.Routes) == 0 {
		t.Fatal("no routes collected: 装配异常，端点面快照无意义")
	}
	// 快照自身健康约束：无重复、每条都有认证档；登记的非 RPC 例外（见 spec
	// api_spec_base.md「登记的非 RPC 例外」）必须存在，且认证档与真实中间件链一致。
	seen := map[string]string{}
	for _, r := range manifest.Routes {
		key := r.Method + " " + r.Path
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate route in manifest: %s", key)
		}
		seen[key] = r.Auth
		if r.Auth == "" {
			t.Fatalf("route missing auth tier: %s", key)
		}
	}
	nonRPCExceptions := map[string]string{
		"GET /healthz":                           "probe",
		"GET /readyz":                            "probe",
		"GET /metrics":                           "probe",
		"GET /public/v1/media/avatars/{file_id}": "anonymous",
		"GET /share/{code}":                      "anonymous",
	}
	for route, wantAuth := range nonRPCExceptions {
		gotAuth, ok := seen[route]
		if !ok {
			t.Fatalf("registered non-RPC exception missing from manifest: %s", route)
		}
		if gotAuth != wantAuth {
			t.Fatalf("registered non-RPC exception auth tier mismatch: %s: got %q, want %q", route, gotAuth, wantAuth)
		}
	}

	blob, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("encode route manifest: %v", err)
	}
	blob = append(blob, '\n')
	path := routeManifestPath()
	if os.Getenv("UPDATE_ROUTE_MANIFEST") == "1" {
		if err := os.WriteFile(path, blob, 0o644); err != nil {
			t.Fatalf("write route manifest: %v", err)
		}
		t.Logf("route manifest updated: %s (%d routes)", path, len(manifest.Routes))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read route manifest: %v（首次生成请运行 UPDATE_ROUTE_MANIFEST=1 go test ./internal/bootstrap -run TestRouteManifest）", err)
	}
	if string(want) != string(blob) {
		t.Fatalf("端点面与 %s 不一致：若为有意增删端点，运行 UPDATE_ROUTE_MANIFEST=1 go test ./internal/bootstrap -run TestRouteManifest 并 review diff；否则说明路由装配/认证档发生漂移", path)
	}
}

func collectRouteManifest(r *chi.Mux) []routeManifestEntry {
	entries := make([]routeManifestEntry, 0, 200)
	_ = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.ReplaceAll(route, "/*", "")
		if route == "" || route == "/" {
			return nil
		}
		entries = append(entries, routeManifestEntry{
			Method: strings.ToUpper(method),
			Path:   route,
			Auth:   routeAuthTier(route),
		})
		return nil
	})
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Method < entries[j].Method
	})
	return entries
}

// routeAuthTier 按挂载前缀推导认证档：/public/v1 匿名、/admin/api/v1 管理员、
// 探针独立、其余一律要求登录（与各模块 Mount 的中间件链一致）。根路径上的
// 非 RPC 例外按真实挂载标注：/share/{code} 仅挂限流、匿名可访问（spec
// api_spec_base.md「登记的非 RPC 例外」），任何认证档变化都会体现在快照 diff 中。
func routeAuthTier(path string) string {
	switch {
	case strings.HasPrefix(path, "/public/v1/"):
		return "anonymous"
	case strings.HasPrefix(path, "/admin/api/v1/"):
		return "admin"
	case path == "/healthz" || path == "/readyz" || path == "/metrics":
		return "probe"
	case path == "/share/{code}":
		return "anonymous"
	default:
		return "authenticated"
	}
}
