package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/media"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/buildinfo"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/cors"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
	"hotspot/backend-go/internal/platform/routemeta"
)

func NewHTTPRouter(modules []modulekit.Module) http.Handler {
	return newHTTPRouter(modules, nil, nil, nil, idempotency.NewInMemoryStore(), nil, nil, nil)
}

func NewHTTPRouterWithLogger(modules []modulekit.Module, logger *observability.Logger) http.Handler {
	return newHTTPRouter(modules, nil, nil, nil, idempotency.NewInMemoryStore(), nil, nil, nil)
}

// NewHTTPRouterWithReadiness 在生产装配下注入就绪探针（DB 必查、Redis 降级
// 仅告警不摘流量），供容器 healthcheck/LB 使用。
func NewHTTPRouterWithReadiness(modules []modulekit.Module, logger *observability.Logger, readiness http.Handler, guardChecker resourceguard.Checker, idemStore idempotency.Store, platformAuth *platformauth.Service, adminChecker platformauth.AdminChecker, hub *realtime.Hub) http.Handler {
	return newHTTPRouter(modules, logger, readiness, guardChecker, idemStore, platformAuth, adminChecker, hub)
}

func realtimeHubMiddleware(hub *realtime.Hub) func(http.Handler) http.Handler {
	// nil 仅供单测/轻量装配回退到进程级 Hub；生产 NewApp 恒传应用级实例。
	if hub == nil {
		hub = realtime.DefaultHub()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(realtime.WithHub(r.Context(), hub)))
		})
	}
}

func newHTTPRouter(modules []modulekit.Module, logger *observability.Logger, readiness http.Handler, guardChecker resourceguard.Checker, idemStore idempotency.Store, platformAuth *platformauth.Service, adminChecker platformauth.AdminChecker, hub *realtime.Hub) http.Handler {
	r := chi.NewRouter()
	// 媒体客户端端点：按请求入口 origin 注入 context，媒体模块据此生成与页面
	// 同源的直传/读 URL（跨源会被 CORS/LNA 拦截，见 media/client_endpoint.go）。
	r.Use(media.ClientEndpointMiddleware)
	// CORS 白名单经 CORS_ALLOWED_ORIGINS 装配：同源部署默认为空（不响应跨源）。
	corsCfg := cors.DefaultConfig().WithAllowedOrigins(config.Load().CorsAllowedOrigins)
	r.Use(cors.Middleware(corsCfg))
	r.Use(observability.RequestIDMiddleware)
	// 应用级 Logger 注入请求 context：模块内统一 observability.FromContext(ctx)，
	// 不再跨包直取进程级 defaultLogger（与 realtime Hub 同一 DI 模式）。
	r.Use(observability.WithLoggerMiddleware(logger))
	if logger != nil {
		r.Use(observability.RequestLoggerMiddleware(logger))
	}
	// Catch panics into a JSON envelope (never leak a stack trace) and cap
	// request bodies before they reach handlers.
	r.Use(httpx.Recoverer)
	r.Use(httpx.LimitRequestBody)
	// 请求计数/耗时指标（chi 路由模板维度）。放在 Recoverer 之后：panic
	// 恢复为 500 的请求同样被计数。
	r.Use(observability.RequestMetricsMiddleware)
	// 实时 Hub 显式注入请求 context（替代模块内 80+ 处 DefaultHub 直取的隐式
	// 全局依赖）：请求路径统一从 ctx 取 Hub，组合根在这里注入一次。
	r.Use(realtimeHubMiddleware(hub))
	// 资源守卫 checker 显式注入请求 context（替代进程级 SetDefaultChecker）：
	// 守卫中间件与写入方 Record* 统一从 ctx 取实例，同进程可装配多套 App。
	r.Use(resourceguard.WithChecker(guardChecker))
	// 幂等 Store 同样显式注入（替代进程级 SetStore）：路由树任意位置都能读到，
	// 多套 App 各自持有独立 store。
	if idemStore == nil {
		idemStore = idempotency.NewInMemoryStore()
	}
	r.Use(idempotency.WithStore(idemStore))
	// 平台认证依赖显式注入（替代进程级 DefaultService/AdminChecker 全局）：
	// RequiredAuth/OptionalAuth/AdminAccess 统一从 ctx 读取。
	r.Use(platformauth.WithDependencies(platformAuth, adminChecker))
	r.NotFound(httpx.NotFoundHandler)
	r.MethodNotAllowed(httpx.MethodNotAllowedHandler)
	// Prometheus 抓取端点：仅 compose 网络内可达（nginx 不反代 /metrics，
	// 外部访问落到 SPA 兜底）；可选采集栈见 docker-compose.yml observability
	// profile。不挂限流——15s 抓取节奏不会触发限流桶，也不应被其干扰。
	r.Method(http.MethodGet, "/metrics", observability.MetricsHandler())
	// 存活探针（全站约定 GET /healthz）：无鉴权、无限流、无请求体要求，
	// 只回 status/version/commit/build_time，供 LB/容器健康检查与 ops 绿灯
	// 聚合。业务健康（DB/Redis 连通性）仍走 POST /public/v1/system/health。
	r.Method(http.MethodGet, "/healthz", http.HandlerFunc(healthzProbe))
	// 就绪探针（GET /readyz）：与存活探针分离——DB 不可达返回 503（实例不可
	// 接新流量），Redis 不可达仅标记 degraded（限流/幂等 fail-open，仍可服务）。
	// 未注入依赖时（单元测试装配）按 ready 处理，不阻塞路由构建。
	if readiness == nil {
		readiness = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]any{"status": "ok", "checks": map[string]string{}}))
		})
	}
	r.Method(http.MethodGet, "/readyz", readiness)
	for _, module := range modules {
		module.Mount(r)
	}
	// /public/v1 匿名节点（spec api_spec_public.md）：纯匿名、不解析 Token，
	// 限流按 IP 维度（ratelimit 内部按路径分类），无资源守卫。
	publicRouter := chi.NewRouter()
	publicRouter.Use(ratelimit.Middleware)
	publicRouter.Use(idempotency.Middleware)
	publicRouter.NotFound(httpx.NotFoundHandler)
	publicRouter.MethodNotAllowed(httpx.MethodNotAllowedHandler)
	for _, module := range modules {
		if pm, ok := module.(modulekit.PublicMounter); ok {
			pm.MountPublic(publicRouter)
		}
	}
	r.Mount("/public/v1", publicRouter)
	// 路由元数据一致性审计（审查 P2-架构）：限流分类与资源守卫规则统一声明
	// 在 platform/routemeta，此处对「已挂载路由 ↔ 元数据声明」做双向校验并
	// 以 Warn 暴露漂移（新端点漏分类 / 守卫规则成了永不命中的死配置）。
	// CI 侧由 bootstrap/route_metadata_test.go 升级为硬断言。
	if audit := routemeta.AuditRoutes(collectRoutePatterns(r)); !audit.OK() {
		warn := observability.L().Warn
		if logger != nil {
			warn = logger.Warn
		}
		if len(audit.Unclassified) > 0 {
			warn("route metadata: 路由缺少限流分类", "paths", audit.Unclassified)
		}
		if len(audit.DeadGuardRules) > 0 {
			warn("route metadata: 资源守卫规则未命中任何路由（死配置）", "suffixes", audit.DeadGuardRules)
		}
	}
	return r
}

// collectRoutePatterns 枚举 chi 路由树，返回收敛后的完整路径（如
// /api/v1/moments/create）。Mount 贡献的 "/*" 分隔符会被去掉；GET /share/{code}
// 这类带参数的路径保持原样。
func collectRoutePatterns(r *chi.Mux) []string {
	var out []string
	var walk func(routes []chi.Route, prefix string)
	walk = func(routes []chi.Route, prefix string) {
		for _, rt := range routes {
			full := prefix + rt.Pattern
			if rt.SubRoutes != nil {
				walk(rt.SubRoutes.Routes(), full)
				continue
			}
			out = append(out, strings.ReplaceAll(full, "/*", ""))
		}
	}
	walk(r.Routes(), "")
	return out
}

// healthzProbe 存活探针（全站约定 GET /healthz）
// 无鉴权、无限流、无请求体要求。
func healthzProbe(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]string{
		"status":     "ok",
		"version":    buildinfo.Version,
		"commit":     buildinfo.Commit,
		"build_time": buildinfo.BuildTime,
	}))
}

// readinessProbe 就绪探针：数据库为硬依赖（失败即 503），Redis 为降级依赖
// （失败仅 degraded，实例继续接流量）。检查总预算 2s，避免探针拖垮进程。
type readinessProbe struct {
	checks []readinessCheck
}

type readinessCheck struct {
	name string
	// required 为 true 时失败导致整体 503；否则只降级。
	required bool
	ping     func(context.Context) error
}

func newReadinessProbe(checks ...readinessCheck) http.Handler {
	return readinessProbe{checks: checks}
}

func (p readinessProbe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	results := make(map[string]string, len(p.checks))
	status := "ok"
	httpStatus := http.StatusOK
	for _, check := range p.checks {
		if check.ping == nil || check.ping(ctx) == nil {
			results[check.name] = "ok"
			continue
		}
		results[check.name] = "down"
		if check.required {
			httpStatus = http.StatusServiceUnavailable
			status = "unavailable"
		} else if status == "ok" {
			status = "degraded"
		}
	}
	// 就绪探针不是业务接口：直接返回 {status,checks}，LB/容器只认 HTTP 状态码。
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "checks": results})
}
