package system

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

func benchmarkSystemHandler() *Handler {
	svc := NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(appconfig.NewStaticSource(config.Load())))
	return NewHandler(svc)
}

// BenchmarkBootstrapHandler 覆盖登录版冷启动 envelope 编码 + 版本组装。
func BenchmarkBootstrapHandler(b *testing.B) {
	handler := benchmarkSystemHandler()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/system/bootstrap", strings.NewReader(`{"client_type":"web"}`))
		rr := httptest.NewRecorder()
		handler.Bootstrap(rr, req)
	}
}

// BenchmarkPublicBootstrapHandler 覆盖匿名 PublicBootstrapData 编码（无 profile/
// unread_summary，缓存 miss 路径）。
func BenchmarkPublicBootstrapHandler(b *testing.B) {
	handler := benchmarkSystemHandler()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/public/v1/system/bootstrap", strings.NewReader(`{"client_type":"web"}`))
		rr := httptest.NewRecorder()
		handler.PublicBootstrap(rr, req)
	}
}
