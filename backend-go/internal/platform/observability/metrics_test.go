package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestRequestMetricsMiddlewareCountsAndLabels：请求经中间件后
// http_requests_total 按 method/route/status 计数，未匹配路由归入
// unmatched（路径参数不得成为标签值）。
func TestRequestMetricsMiddlewareCountsAndLabels(t *testing.T) {
	r := chi.NewRouter()
	r.Use(RequestMetricsMiddleware)
	r.Post("/api/v1/things/{thingId}/like", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	// 命中路由两次（不同路径参数，同一模板）
	for _, id := range []string{"aaa", "bbb"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/things/"+id+"/like", http.NoBody)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rr.Code)
		}
	}
	// 未匹配一次（404）
	req404 := httptest.NewRequest(http.MethodPost, "/elsewhere", http.NoBody)
	rr404 := httptest.NewRecorder()
	r.ServeHTTP(rr404, req404)

	out := MetricsHandler().ServeHTTP
	rec := httptest.NewRecorder()
	out(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	body := rec.Body.String()

	if !strings.Contains(body, `http_requests_total{method="POST",route="/api/v1/things/{thingId}/like",status="201"} 2`) {
		t.Fatalf("missing routed counter: %s", body)
	}
	if !strings.Contains(body, `route="unmatched"`) {
		t.Fatalf("missing unmatched counter: %s", body)
	}
	if !strings.Contains(body, "http_request_duration_seconds") {
		t.Fatalf("missing duration histogram: %s", body)
	}
}

// TestMetricsHandlerExposedCollectors：默认采集器（Go 运行时/进程）在
// /metrics 输出中可见——prometheus 抓取的第一屏内容。
func TestMetricsHandlerExposedCollectors(t *testing.T) {
	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	body := rec.Body.String()
	if !strings.Contains(body, "go_goroutines") || !strings.Contains(body, "process_resident_memory_bytes") {
		t.Fatalf("default go/process collectors missing from /metrics: %s", body[:min(len(body), 400)])
	}
}
