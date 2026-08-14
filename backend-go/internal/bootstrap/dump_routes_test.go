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

func walkRouteList(routes []chi.Route, prefix string, out *[]string) {
	for _, rt := range routes {
		full := prefix + rt.Pattern
		if rt.SubRoutes != nil {
			walkRouteList(rt.SubRoutes.Routes(), full, out)
			continue
		}
		// chi 挂载层贡献 "/*" 分隔符，收敛为具体路径（/admin/v1/announcements/create）
		clean := strings.ReplaceAll(full, "/*", "")
		*out = append(*out, clean)
	}
}

// TestDumpAllRoutes 打印全部已注册路由（API 审计用：spec 端点 vs 实现全量比对）。
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
	var out []string
	walkRouteList(mux.Routes(), "", &out)
	lines := strings.Join(out, "\n") + "\n"
	os.WriteFile("/tmp/backend_routes_dump.txt", []byte(lines), 0o644)
	fmt.Print(lines)
}
