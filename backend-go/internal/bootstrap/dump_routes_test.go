package bootstrap

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestDumpAllRoutes 打印全部已注册路由（API 审计用：spec 端点 vs 实现全量比对）。
// 路由枚举复用生产侧 collectRoutePatterns（与启动审计同一实现，避免两份
// walk 逻辑各自漂移）。
func TestDumpAllRoutes(t *testing.T) {
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
	lines := strings.Join(collectRoutePatterns(mux), "\n") + "\n"
	os.WriteFile("/tmp/backend_routes_dump.txt", []byte(lines), 0o644)
	fmt.Print(lines)
}
