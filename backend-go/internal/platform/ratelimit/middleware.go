package ratelimit

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/requestmeta"
)

var (
	sourceMu      sync.RWMutex
	defaultSource appconfig.Source = appconfig.NewStaticSource(config.Config{})
	storeMu       sync.Mutex
	windows       = map[string][]time.Time{}
	tokens        = map[string]*tokenBucket{}
)

type tokenBucket struct {
	Tokens     float64
	LastRefill time.Time
}

type rule struct {
	name     string
	strategy string
	limit    int
	window   time.Duration
	scope    string
	key      string
}

type routeClass struct {
	adminOnly bool
	readOnly  bool
}

func SetSource(source appconfig.Source) {
	if source == nil {
		return
	}
	sourceMu.Lock()
	defer sourceMu.Unlock()
	defaultSource = source
}

func CurrentRateLimits(ctx context.Context) appconfig.RateLimitConfig {
	return currentSource().Get(ctx).RateLimits
}

func AllowWebsocketSubscribe(ctx context.Context, userID, connectionID string) (int, bool) {
	if userID == "" {
		return 0, false
	}
	limit := CurrentRateLimits(ctx).WSSubscribePerSecond
	if limit <= 0 {
		return 0, false
	}
	key := connectionID
	if key == "" {
		key = userID
	}
	return allow(time.Now().UTC(), rule{name: "ws_subscribe_per_second", strategy: "fixed", limit: limit, window: time.Second, scope: "connection", key: key})
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rules := classifyRules(r, CurrentRateLimits(r.Context()))
		now := time.Now().UTC()
		for _, rl := range rules {
			if retryAfter, limited := allow(now, rl); limited {
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10500, "请求过于频繁"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func currentSource() appconfig.Source {
	sourceMu.RLock()
	defer sourceMu.RUnlock()
	return defaultSource
}

func classifyRules(r *http.Request, cfg appconfig.RateLimitConfig) []rule {
	path := r.URL.Path
	principal, hasPrincipal := platformauth.PrincipalFromContext(r.Context())
	ip := requestmeta.ClientIP(r)
	route, ok := classifyRoute(path)
	if !ok {
		return nil
	}
	rules := make([]rule, 0, 4)
	if cfg.GlobalPerHour > 0 && ip != "" {
		rules = append(rules, rule{name: "global_per_hour", strategy: "sliding", limit: cfg.GlobalPerHour, window: time.Hour, scope: "ip", key: ip})
	}
	if cfg.UserPerHour > 0 && hasPrincipal && principal.UserID != "" {
		rules = append(rules, rule{name: "user_per_hour", strategy: "sliding", limit: cfg.UserPerHour, window: time.Hour, scope: "user", key: principal.UserID})
	}
	if matched := appendAuthRules(&rules, path, ip, cfg); matched {
		return rules
	}
	if route.adminOnly {
		if !hasPrincipal || principal.UserID == "" {
			return rules
		}
		if route.readOnly && cfg.AdminReadPerMinute > 0 {
			rules = append(rules, rule{name: "admin_read_per_minute", strategy: "token_bucket", limit: cfg.AdminReadPerMinute, window: time.Minute, scope: "user", key: principal.UserID})
		}
		if !route.readOnly && cfg.AdminWritePerMinute > 0 {
			rules = append(rules, rule{name: "admin_write_per_minute", strategy: "sliding", limit: cfg.AdminWritePerMinute, window: time.Minute, scope: "user", key: principal.UserID})
		}
		return rules
	}
	if strings.HasPrefix(path, "/api/v1/search/") {
		key := subjectKey(ip, principal, hasPrincipal)
		if key != "" && cfg.SearchPerMinute > 0 {
			rules = append(rules, rule{name: "search_per_minute", strategy: "token_bucket", limit: cfg.SearchPerMinute, window: time.Minute, scope: subjectScope(ip, principal, hasPrincipal), key: key})
		}
		return rules
	}
	if !route.readOnly {
		key := subjectKey(ip, principal, hasPrincipal)
		if key != "" && cfg.WritePerMinute > 0 {
			rules = append(rules, rule{name: "write_per_minute", strategy: "sliding", limit: cfg.WritePerMinute, window: time.Minute, scope: subjectScope(ip, principal, hasPrincipal), key: key})
		}
	}
	if hasPrincipal && principal.UserID != "" {
		switch path {
		case "/api/v1/moments/create":
			if cfg.MomentsCreatePerMinute > 0 {
				rules = append(rules, rule{name: "moments_create_per_minute", strategy: "sliding", limit: cfg.MomentsCreatePerMinute, window: time.Minute, scope: "user", key: principal.UserID})
			}
		case "/api/v1/comments/create":
			if cfg.CommentsCreatePerMinute > 0 {
				rules = append(rules, rule{name: "comments_create_per_minute", strategy: "sliding", limit: cfg.CommentsCreatePerMinute, window: time.Minute, scope: "user", key: principal.UserID})
			}
		case "/api/v1/reports/create":
			if cfg.ReportsCreatePerHour > 0 {
				rules = append(rules, rule{name: "reports_create_per_hour", strategy: "sliding", limit: cfg.ReportsCreatePerHour, window: time.Hour, scope: "user", key: principal.UserID})
			}
		case "/api/v1/media/presign_upload", "/api/v1/media/batch_presign_upload":
			if cfg.UploadPresignPerHour > 0 {
				rules = append(rules, rule{name: "upload_presign_per_hour", strategy: "sliding", limit: cfg.UploadPresignPerHour, window: time.Hour, scope: "user", key: principal.UserID})
			}
		}
	}
	return rules
}

func appendAuthRules(rules *[]rule, path, ip string, cfg appconfig.RateLimitConfig) bool {
	if !strings.HasPrefix(path, "/api/v1/auth/") || ip == "" {
		return false
	}
	switch path {
	case "/api/v1/auth/register":
		if cfg.AuthRegisterPer10Minutes > 0 {
			*rules = append(*rules, rule{name: "auth_register_per_10minutes", strategy: "sliding", limit: cfg.AuthRegisterPer10Minutes, window: 10 * time.Minute, scope: "ip", key: ip})
		}
		return true
	case "/api/v1/auth/login":
		if cfg.AuthLoginPer10Minutes > 0 {
			*rules = append(*rules, rule{name: "auth_login_per_10minutes", strategy: "sliding", limit: cfg.AuthLoginPer10Minutes, window: 10 * time.Minute, scope: "ip", key: ip})
		}
		return true
	case "/api/v1/auth/guest":
		if cfg.AuthGuestPer10Minutes > 0 {
			*rules = append(*rules, rule{name: "auth_guest_per_10minutes", strategy: "sliding", limit: cfg.AuthGuestPer10Minutes, window: 10 * time.Minute, scope: "ip", key: ip})
		}
		return true
	case "/api/v1/auth/refresh_token":
		if cfg.AuthRefreshPer10Minutes > 0 {
			*rules = append(*rules, rule{name: "auth_refresh_per_10minutes", strategy: "sliding", limit: cfg.AuthRefreshPer10Minutes, window: 10 * time.Minute, scope: "ip", key: ip})
		}
		return true
	case "/api/v1/auth/request_password_reset", "/api/v1/auth/reset_password":
		if cfg.AuthResetPer10Minutes > 0 {
			*rules = append(*rules, rule{name: "auth_reset_per_10minutes", strategy: "sliding", limit: cfg.AuthResetPer10Minutes, window: 10 * time.Minute, scope: "ip", key: ip})
		}
		return true
	default:
		return false
	}
}

func classifyRoute(path string) (routeClass, bool) {
	switch {
	case strings.HasPrefix(path, "/api/v1/admin/"):
		return routeClass{adminOnly: true, readOnly: isAdminReadPath(path)}, true
	case strings.HasPrefix(path, "/api/v1/system/"):
		return routeClass{readOnly: true}, true
	case strings.HasPrefix(path, "/api/v1/feed/"):
		return routeClass{readOnly: true}, true
	case strings.HasPrefix(path, "/api/v1/search/"):
		return routeClass{readOnly: true}, true
	case strings.HasPrefix(path, "/api/v1/auth/"):
		return routeClass{readOnly: isAuthReadPath(path)}, true
	case strings.HasPrefix(path, "/api/v1/media/"):
		return routeClass{readOnly: false}, true
	case strings.HasPrefix(path, "/api/v1/reports/"):
		return routeClass{readOnly: strings.HasSuffix(path, "/me/list")}, true
	case strings.HasPrefix(path, "/api/v1/moments/"):
		return routeClass{readOnly: !strings.HasSuffix(path, "/create")}, true
	case strings.HasPrefix(path, "/api/v1/comments/"):
		return routeClass{readOnly: !strings.HasSuffix(path, "/create")}, true
	case strings.HasPrefix(path, "/api/v1/groups/"):
		return routeClass{readOnly: strings.HasSuffix(path, "/search") || strings.HasSuffix(path, "/open_the_door")}, true
	case strings.HasPrefix(path, "/api/v1/users/"):
		return routeClass{readOnly: isUsersReadPath(path)}, true
	case strings.HasPrefix(path, "/api/v1/private_chats/"):
		return routeClass{readOnly: true}, true
	case strings.HasPrefix(path, "/api/v1/private_messages/"):
		return routeClass{readOnly: strings.HasSuffix(path, "/get_history")}, true
	case strings.HasPrefix(path, "/api/v1/group_messages/"):
		return routeClass{readOnly: strings.HasSuffix(path, "/get_history")}, true
	case strings.HasPrefix(path, "/api/v1/notifications/"):
		return routeClass{readOnly: strings.HasSuffix(path, "/list") || strings.HasSuffix(path, "/unread_count")}, true
	case strings.HasPrefix(path, "/api/v1/share/"):
		return routeClass{readOnly: strings.HasSuffix(path, "/get") || strings.HasSuffix(path, "/me/list")}, true
	case strings.HasPrefix(path, "/api/v1/map/"):
		return routeClass{readOnly: true}, true
	default:
		return routeClass{}, false
	}
}

func isAuthReadPath(path string) bool {
	switch path {
	case "/api/v1/auth/check_status", "/api/v1/auth/me/sessions/list":
		return true
	default:
		return false
	}
}

func isAdminReadPath(path string) bool {
	return strings.Contains(path, "/get") || strings.Contains(path, "/list") || strings.Contains(path, "/search") || strings.Contains(path, "/open_the_door")
}

func isUsersReadPath(path string) bool {
	switch path {
	case "/api/v1/users/search", "/api/v1/users/open_the_door", "/api/v1/users/me/profile/get", "/api/v1/users/me/settings/get", "/api/v1/users/me/list_blocked":
		return true
	default:
		return false
	}
}

func subjectKey(ip string, principal platformauth.Principal, hasPrincipal bool) string {
	if hasPrincipal && principal.UserID != "" {
		return principal.UserID
	}
	return ip
}

func subjectScope(ip string, principal platformauth.Principal, hasPrincipal bool) string {
	if hasPrincipal && principal.UserID != "" {
		return "user"
	}
	if ip != "" {
		return "ip"
	}
	return ""
}

func allow(now time.Time, rl rule) (int, bool) {
	if rl.limit <= 0 || rl.key == "" {
		return 0, false
	}
	storeMu.Lock()
	defer storeMu.Unlock()
	composite := rl.name + "::" + rl.scope + "::" + rl.key
	switch rl.strategy {
	case "token_bucket":
		return allowTokenBucket(now, composite, rl.limit, rl.window)
	case "fixed":
		return allowFixedWindow(now, composite, rl.limit, rl.window)
	default:
		return allowSlidingWindow(now, composite, rl.limit, rl.window)
	}
}

func allowSlidingWindow(now time.Time, key string, limit int, window time.Duration) (int, bool) {
	cutoff := now.Add(-window)
	events := windows[key][:0]
	for _, ts := range windows[key] {
		if ts.After(cutoff) {
			events = append(events, ts)
		}
	}
	if len(events) >= limit {
		windows[key] = events
		retry := int(math.Ceil(events[0].Add(window).Sub(now).Seconds()))
		if retry < 1 {
			retry = 1
		}
		return retry, true
	}
	windows[key] = append(events, now)
	return 0, false
}

func allowFixedWindow(now time.Time, key string, limit int, window time.Duration) (int, bool) {
	windowStart := now.Truncate(window)
	cutoff := windowStart.Add(-window)
	events := windows[key][:0]
	for _, ts := range windows[key] {
		if !ts.Before(cutoff) {
			events = append(events, ts)
		}
	}
	count := 0
	for _, ts := range events {
		if !ts.Before(windowStart) {
			count++
		}
	}
	if count >= limit {
		windows[key] = events
		retry := int(math.Ceil(windowStart.Add(window).Sub(now).Seconds()))
		if retry < 1 {
			retry = 1
		}
		return retry, true
	}
	windows[key] = append(events, now)
	return 0, false
}

func allowTokenBucket(now time.Time, key string, limit int, window time.Duration) (int, bool) {
	bucket := tokens[key]
	if bucket == nil {
		bucket = &tokenBucket{Tokens: float64(limit), LastRefill: now}
		tokens[key] = bucket
	}
	rate := float64(limit) / window.Seconds()
	elapsed := now.Sub(bucket.LastRefill).Seconds()
	bucket.Tokens = math.Min(float64(limit), bucket.Tokens+elapsed*rate)
	bucket.LastRefill = now
	if bucket.Tokens < 1 {
		retry := int(math.Ceil((1 - bucket.Tokens) / rate))
		if retry < 1 {
			retry = 1
		}
		return retry, true
	}
	bucket.Tokens -= 1
	return 0, false
}
