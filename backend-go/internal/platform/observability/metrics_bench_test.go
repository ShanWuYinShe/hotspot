package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// BenchmarkRequestMetricsMiddleware 测每个请求的指标采集开销
// （counter/histogram observe + 路由模板解析），是全部业务请求的固定税。
func BenchmarkRequestMetricsMiddleware(b *testing.B) {
	r := chi.NewRouter()
	r.Use(RequestMetricsMiddleware)
	r.Get("/api/v1/moments/{momentId}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/moments/m_bench", nil)
	rr := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr.Body.Reset()
		rr.Code = http.StatusOK
		r.ServeHTTP(rr, req)
	}
}
