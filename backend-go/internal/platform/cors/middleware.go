package cors

import (
	"net/http"
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
func DefaultConfig() Config {
	return Config{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "X-CSRF-Token", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}
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

			if cfg.AllowCredentials {
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
					w.Header().Set("Access-Control-Max-Age", strings.TrimSpace(strings.Repeat(" ", 0)+intToStr(cfg.MaxAge)))
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

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
