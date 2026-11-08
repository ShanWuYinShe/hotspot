package privatechats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestCoverageG5ModuleWiring 覆盖模块装配、Service accessor 与路由挂载。
// 路由注册本身执行 Mount 的全部语句；未装配认证依赖时 RequiredAuth 按
// fail-closed 返回 10600，可证明四个端点均已登记（而非落到 chi 的 404）。
func TestCoverageG5ModuleWiring(t *testing.T) {
	source := appconfig.NewStaticSource(config.Load())
	module := NewModule(persistence.NewFactory(config.Config{}), source, nil, nil)
	if module == nil || module.Service() == nil {
		t.Fatal("expected module to expose privatechats service")
	}
	router := chi.NewRouter()
	module.Mount(router)

	paths := []string{
		"/api/v1/private_chats/fetch",
		"/api/v1/private_chats/open_the_door",
		"/api/v1/private_chats/update_preferences",
		"/api/v1/private_chats/delete",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			var resp struct {
				Code httpx.Code `json:"code"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if resp.Code != httpx.CodeInternalUnavailable {
				t.Fatalf("route not mounted or auth not fail-closed: code=%d want %d", resp.Code, httpx.CodeInternalUnavailable)
			}
		})
	}
}
