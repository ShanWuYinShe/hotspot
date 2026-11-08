package share

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestShareModuleMountsAllRoutes 覆盖一等模块装配路径 NewModule/Mount/MountPublic：
// 只做路由注册断言（chi Match 不执行根限流中间件，避免单测依赖全局限流状态）。
func TestShareModuleMountsAllRoutes(t *testing.T) {
	mod := NewModule(persistence.NewFactory(config.Config{}))
	if mod == nil || mod.handler == nil || mod.Service == nil {
		t.Fatalf("module must expose handler and service: %#v", mod)
	}

	r := chi.NewRouter()
	mod.Mount(r)
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/share/create"},
		{http.MethodPost, "/api/v1/share/fetch_created"},
		{http.MethodPost, "/api/v1/share/revoke"},
		{http.MethodGet, "/share/ABCDEFGHJKLM"},
	} {
		if !r.Match(chi.NewRouteContext(), tc.method, tc.path) {
			t.Fatalf("%s %s not mounted", tc.method, tc.path)
		}
	}
	if r.Match(chi.NewRouteContext(), http.MethodPost, "/api/v1/share/not_exists") {
		t.Fatal("unexpected route /api/v1/share/not_exists mounted")
	}

	pub := chi.NewRouter()
	mod.MountPublic(pub)
	if !pub.Match(chi.NewRouteContext(), http.MethodPost, "/share/open_the_door") {
		t.Fatal("public share open_the_door route not mounted")
	}
}
