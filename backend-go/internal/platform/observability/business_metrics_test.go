package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestBusinessErrorMetricRecordsEnvelopeErrors：契约错误走 HTTP 200 + envelope
// code!=0，HTTP 指标看不到；统一日志出口必须同时记录业务错误指标（含路由模板）。
func TestBusinessErrorMetricRecordsEnvelopeErrors(t *testing.T) {
	router := chi.NewRouter()
	router.Post("/api/v1/things/{thingId}/fail", func(_ http.ResponseWriter, r *http.Request) {
		LogBusinessError(r, "ThingsFail", 10400, "参数错误")
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/things/t1/fail", nil))

	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `hotspot_business_errors_total{code="10400",method="POST",route="/api/v1/things/{thingId}/fail"} 1`) {
		t.Fatalf("business error metric missing route template: %s", body[:min(len(body), 800)])
	}
}
