package comments

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestCommentsModuleMountRegistersAllRoutes 覆盖 module.go 的装配与挂载：
// NewModule 组装 Postgres 仓储 + 事务运行器，Mount/MountPublic 注册全部评论
// 端点（认证分组内 5 个 + 匿名 3 个）。空配置的 factory 走「未配置 DB」分支，
// 挂载阶段不会触碰仓储，因此无需数据库即可验证路由表。
func TestCommentsModuleMountRegistersAllRoutes(t *testing.T) {
	factory := persistence.NewFactory(config.Config{})
	mod := NewModule(factory, nil, nil, nil, nil)
	if mod == nil || mod.Handler == nil || mod.Service == nil {
		t.Fatalf("NewModule must wire handler and service: %#v", mod)
	}
	if mod.Handler.service != mod.Service {
		t.Fatal("handler must share the module service")
	}
	if mod.Service.repo == nil {
		t.Fatal("NewModule must wire the postgres repository")
	}
	if mod.Service.runInTx == nil {
		t.Fatal("NewModule must wire the transaction runner")
	}

	router := chi.NewRouter()
	mod.Mount(router)
	authRoutes := []string{
		"/api/v1/comments/fetch",
		"/api/v1/comments/fetch_replies",
		"/api/v1/comments/open_the_door",
		"/api/v1/comments/create",
		"/api/v1/comments/delete",
	}
	for _, path := range authRoutes {
		if !router.Match(chi.NewRouteContext(), http.MethodPost, path) {
			t.Fatalf("authenticated route %s must be mounted", path)
		}
		if router.Match(chi.NewRouteContext(), http.MethodGet, path) {
			t.Fatalf("route %s must only accept POST", path)
		}
	}
	if router.Match(chi.NewRouteContext(), http.MethodPost, "/api/v1/comments/unknown") {
		t.Fatal("unknown comments path must not match")
	}

	public := chi.NewRouter()
	mod.MountPublic(public)
	for _, path := range []string{"/comments/fetch", "/comments/fetch_replies", "/comments/open_the_door"} {
		if !public.Match(chi.NewRouteContext(), http.MethodPost, path) {
			t.Fatalf("public route %s must be mounted", path)
		}
		if public.Match(chi.NewRouteContext(), http.MethodPost, path+"/extra") {
			t.Fatalf("public route %s must not match sub-paths", path)
		}
	}
}
