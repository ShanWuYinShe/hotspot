package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/observability"
)

// TestWriteCodeErrorRecordsMetricAndResponse：业务错误统一出口必须同时输出
// 契约错误响应并记录 hotspot_business_errors_total{code,route,method} 指标
// （此前 221 处裸 WriteJSON(Error) 不进入观测）。
func TestWriteCodeErrorRecordsMetricAndResponse(t *testing.T) {
	router := chi.NewRouter()
	router.Post("/api/v1/business-error-test/{id}", func(w http.ResponseWriter, r *http.Request) {
		WriteCodeError(w, r, CodeInvalidRequest, "参数错误")
	})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/business-error-test/x1", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"code":10000`) {
		t.Fatalf("unexpected response: code=%d body=%s", rr.Code, rr.Body.String())
	}

	rec := httptest.NewRecorder()
	observability.MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	want := `hotspot_business_errors_total{code="10000",method="POST",route="/api/v1/business-error-test/{id}"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("business error metric missing: %s", body[:min(len(body), 800)])
	}
}
