package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/observability"
)

// TestWriteServerErrorRecordsMetricAndResponse：10600 统一出口必须同时
// 输出契约错误响应并记录内部错误指标（route 用 chi 路由模板）。
func TestWriteServerErrorRecordsMetricAndResponse(t *testing.T) {
	router := chi.NewRouter()
	router.Post("/api/v1/things/{thingId}/fail", func(w http.ResponseWriter, r *http.Request) {
		WriteServerError(w, r, errors.New("boom"))
	})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/things/t1/fail", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"code":10600`) {
		t.Fatalf("unexpected response: code=%d body=%s", rr.Code, rr.Body.String())
	}

	rec := httptest.NewRecorder()
	observability.MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `hotspot_internal_errors_total{method="POST",route="/api/v1/things/{thingId}/fail"} 1`) {
		t.Fatalf("internal error metric missing route template: %s", body[:min(len(body), 800)])
	}
}
