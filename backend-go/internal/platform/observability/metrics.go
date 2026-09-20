package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics.go — Prometheus 指标最小闭环（/metrics + 请求指标中间件）。
//
// 采集面：Go 运行时/进程默认采集器 + HTTP 请求计数/耗时（按路由模板分维度，
// chi 的 RoutePattern 是模板（如 /api/v1/groups/{groupId}），404 等未匹配
// 请求归入 "unmatched"，杜绝路径参数造成的高基数标签）。WS 连接与 DB 连接池
// 指标由 realtime / bootstrap 侧经 Register() 挂载自定义采集器。
//
// 暴露：/metrics 挂在主路由（bootstrap/http.go），nginx 不反代该路径——外部
// 访问只会拿到 SPA 兜底，抓取仅发生在 compose 网络内（prometheus 容器）。
// 可选采集栈：仓库根 `docker compose --profile observability up -d`
//（prometheus + grafana，默认不启动，见 docker-compose.yml）。

var registry = prometheus.NewRegistry()

func init() {
	// Go 运行时 + 进程指标（内存/GC/fd/CPU 等）。
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

var (
	httpRequestsTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数（method/route/status 三维；route 为 chi 路由模板）",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP 请求耗时分布（method/route 两维）",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"method", "route"},
	)
)

// Registry 返回指标注册表，供其他包挂载自定义采集器
// （如 realtime 连接数、pgxpool 统计；不要使用 prometheus.DefaultRegisterer）。
func Registry() *prometheus.Registry { return registry }

// Register 把自定义采集器挂到指标注册表（panic 即装配错误，fail-fast 与
// MustRegister 同语义；仅限启动装配路径调用）。
func Register(cs ...prometheus.Collector) { registry.MustRegister(cs...) }

// MetricsHandler 渲染 /metrics（promhttp 文本协议）。
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// RequestMetricsMiddleware 统计每个请求的计数与耗时。放在 httpx.Recoverer
// 之后挂载，panic 恢复为 500 的请求也会被计数。
func RequestMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		// 路由在 chi 匹配后可取到模板；未匹配（404/405）归入 unmatched，
		// 避免把任意请求路径变成标签值（基数爆炸）。
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		method := r.Method
		if r.Method == http.MethodHead {
			method = http.MethodGet // HEAD 与 GET 同模板合并，减一维基数
		}
		status := strconv.Itoa(ww.Status())
		httpRequestsTotal.WithLabelValues(method, route, status).Inc()
		httpRequestDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
	})
}
