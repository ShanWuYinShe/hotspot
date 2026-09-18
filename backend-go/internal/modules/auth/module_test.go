package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func newTestModule(t *testing.T) (*Module, context.CancelFunc) {
	t.Helper()
	cfg := config.Config{AuthTokenSecret: "ci-test-secret-key-at-least-32-bytes-long"}
	ctx, cancel := context.WithCancel(context.Background())
	m := NewModule(ctx, cfg, persistence.NewFactory(cfg), appconfig.NewStaticSource(cfg))
	return m, cancel
}

// 模块装配路径：NewModule/SetLoginTracker/Service/Mount/MountPublic。
// 挂载后公开与鉴权路由都必须真实可达（非 404）。
func TestAuthModuleMountsAllRoutes(t *testing.T) {
	m, cancel := newTestModule(t)
	defer cancel()
	if m.Service() == nil {
		t.Fatal("expected Service() to expose the module service")
	}
	m.SetLoginTracker(newInMemoryLoginTracker(context.Background()))

	r := chi.NewRouter()
	m.Mount(r)
	m.MountPublic(r)

	publicRoutes := []string{
		"/auth/register", "/auth/signin", "/auth/guest", "/auth/send_register_code",
		"/auth/refresh_token", "/auth/forgot_password", "/auth/reset_password",
	}
	for _, path := range publicRoutes {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Fatalf("public route %s not mounted (404)", path)
		}
	}

	authenticatedRoutes := []string{
		"/api/v1/auth/session", "/api/v1/auth/signout", "/api/v1/auth/change_password",
		"/api/v1/auth/fetch_sessions", "/api/v1/auth/revoke_session", "/api/v1/auth/revoke_other_sessions",
	}
	for _, path := range authenticatedRoutes {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Fatalf("authenticated route %s not mounted (404)", path)
		}
		if rr.Code != http.StatusOK {
			t.Fatalf("authenticated route %s should reject unauthenticated calls via envelope, got %d", path, rr.Code)
		}
	}
}

// 保留期清理：sweepRetention 同步跑一轮（repo 未接 PG 时静默等待下一轮）；
// StartRetentionSweeper 幂等守卫生效，ctx 取消后幂等标志复位。
func TestAuthModuleStartRetentionSweeperRunsOnceAndStops(t *testing.T) {
	m, cancel := newTestModule(t)
	defer cancel()

	// 同步执行一轮清理（不依赖 goroutine 调度，确定性覆盖）。
	m.sweepRetention(context.Background())

	sweepCtx, sweepCancel := context.WithCancel(context.Background())
	defer sweepCancel()
	m.StartRetentionSweeper(sweepCtx)
	// 重复调用 no-op（幂等守卫分支）。
	m.StartRetentionSweeper(sweepCtx)
	if !retentionSweepStarted {
		t.Fatal("expected retention sweeper started flag to be set")
	}

	sweepCancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && retentionSweepStarted {
		time.Sleep(5 * time.Millisecond)
	}
	if retentionSweepStarted {
		t.Fatal("expected retention sweeper started flag to reset after ctx cancel")
	}
}
