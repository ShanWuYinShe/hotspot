package bootstrap

import (
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/routemeta"
)

// TestRouteMetadataCoversAllRoutes 路由元数据一致性门禁（审查 P2-架构）：
// 限流分类与资源守卫规则集中在 platform/routemeta，本测试用生产同款装配
// 枚举全部已挂载路由，断言两个方向都不漂移：
//   - 每条业务路由必须被限流分类覆盖（新端点漏分类会让写请求落错限流桶）；
//   - 每条守卫规则必须命中至少一条路由（防止 /groups/update 那类死配置回归）。
func TestRouteMetadataCoversAllRoutes(t *testing.T) {
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
	routes := collectRoutePatterns(mux)
	if len(routes) == 0 {
		t.Fatal("no routes collected: 装配异常，审计无意义")
	}
	audit := routemeta.AuditRoutes(routes)
	if len(audit.Unclassified) > 0 {
		sort.Strings(audit.Unclassified)
		t.Fatalf("以下路由缺少限流分类，请在 platform/routemeta/classify.go 补充:\n%s", strings.Join(audit.Unclassified, "\n"))
	}
	if len(audit.DeadGuardRules) > 0 {
		sort.Strings(audit.DeadGuardRules)
		t.Fatalf("以下资源守卫规则未命中任何已挂载路由（死配置，确认后缀或删除）:\n%s", strings.Join(audit.DeadGuardRules, "\n"))
	}
}
