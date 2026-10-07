package cors

import (
	"net/http"
	"strconv"
	"strings"
)

// Config 定义 CORS 中间件的配置。
type Config struct {
	// AllowedOrigins 允许的来源列表，"*" 表示允许所有。
	AllowedOrigins []string
	// AllowedMethods 允许的 HTTP 方法。
	AllowedMethods []string
	// AllowedHeaders 允许的请求头。
	AllowedHeaders []string
	// ExposedHeaders 暴露给前端的响应头。
	ExposedHeaders []string
	// AllowCredentials 是否允许携带凭证（Cookie 等）。
	AllowCredentials bool
	// MaxAge 预检请求缓存时间（秒）。
	MaxAge int
}

// DefaultConfig 返回适合本项目开发/生产环境的默认 CORS 配置。
// 同源部署（nginx 同一 server 服务前端与 API）：跨源请求只可能来自
// 恶意页面，默认不响应任何跨源（不发 Access-Control-Allow-Origin）；
// 确有跨源消费方时经 CORS_ALLOWED_ORIGINS 显式配置白名单。
// 此前的全局 "*" 是多余暴露面——一旦未来有人顺手打开
// AllowCredentials 即成为「任意站点带凭证跨源」的高危组合。
func DefaultConfig() Config {
	return Config{
		AllowedOrigins:   nil,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}
}

// WithAllowedOrigins 设置允许的跨源白名单（供 bootstrap 经环境变量装配）。
func (c Config) WithAllowedOrigins(origins []string) Config {
	c.AllowedOrigins = origins
	return c
}

// Middleware 返回一个符合 chi 中间件签名的 CORS 处理函数。
func Middleware(cfg Config) func(http.Handler) http.Handler {
	// 预计算常用值
	allowMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	allowAllOrigins := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 判断是否允许该 Origin
			if allowAllOrigins {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if isOriginAllowed(cfg.AllowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			} else {
				next.ServeHTTP(w, r)
				return
			}

			// "*" 与凭证的组合被 Fetch 规范禁止（浏览器会直接拒绝该响应），
			// 且一旦有人把白名单配成 "*" 又打开凭证，就等价于「任意站点带凭证跨源」。
			// 这里选择失败安全：通配模式下不发放凭证头，凭证只在显式白名单下生效。
			if cfg.AllowCredentials && !allowAllOrigins {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if exposeHeaders != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
			}

			// 处理预检请求
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", allowMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(allowed []string, origin string) bool {
	for _, o := range allowed {
		if o == origin {
			return true
		}
	}
	return false
}
