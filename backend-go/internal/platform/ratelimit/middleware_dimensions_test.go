package ratelimit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

func newRequest(ip, path string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, path, nil)
	if ip != "" {
		request.Header.Set("CF-Connecting-IP", ip)
	}
	return request
}

func withPrincipal(request *http.Request, userID string) *http.Request {
	return request.WithContext(platformauth.WithPrincipal(request.Context(), platformauth.Principal{UserID: userID, SessionID: "sess_1"}))
}

func passThrough() http.Handler {
	return Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

// TestGlobalPerHourAppliesAcrossEndpoints 验证 global_per_hour 是跨端点的
// IP 维度总量闸：不同端点共享同一配额。
func TestGlobalPerHourAppliesAcrossEndpoints(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.GlobalPerHour = 1
	}))
	t.Cleanup(func() { SetSource(newTestSource(nil)) })
	handler := passThrough()
	first := serveLimited(handler, newRequest("1.1.1.1", "/api/v1/moments/fetch_nearby"))
	if first.Code != http.StatusOK {
		t.Fatalf("first request should pass, got %d", first.Code)
	}
	second := serveLimited(handler, newRequest("1.1.1.1", "/api/v1/users/get_account"))
	if code := testdb.DecodeAPICode(t, second); code != httpx.CodeRateLimited {
		t.Fatalf("expected rate limit across endpoints, got code %d (body %s)", code, second.Body.String())
	}
}

// TestUserPerHourRequiresPrincipal 验证 user_per_hour 只对登录请求生效：
// 匿名请求不受用户维度限流影响。
func TestUserPerHourRequiresPrincipal(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.UserPerHour = 1
	}))
	t.Cleanup(func() { SetSource(newTestSource(nil)) })
	handler := passThrough()
	// 匿名连续请求均不受 user 维度限制。
	for i := 0; i < 3; i++ {
		recorder := serveLimited(handler, newRequest("2.2.2.2", "/api/v1/moments/fetch_nearby"))
		if recorder.Code != http.StatusOK {
			t.Fatalf("anonymous request %d should pass, got %d", i, recorder.Code)
		}
	}
	// 登录后受用户维度限制。
	authed := withPrincipal(newRequest("2.2.2.2", "/api/v1/moments/fetch_nearby"), "user_a")
	if recorder := serveLimited(handler, authed); recorder.Code != http.StatusOK {
		t.Fatalf("first authed request should pass, got %d", recorder.Code)
	}
	authedSecond := withPrincipal(newRequest("2.2.2.2", "/api/v1/moments/fetch_nearby"), "user_a")
	if code := testdb.DecodeAPICode(t, serveLimited(handler, authedSecond)); code != httpx.CodeRateLimited {
		t.Fatalf("expected user_per_hour to limit second authed request, got code %d", code)
	}
}

// TestMomentsCreatePerMinuteIsPerUser 验证 moments/create 的专用限流按用户
// 维度隔离：user_a 触发不影响 user_b。
func TestMomentsCreatePerMinuteIsPerUser(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.MomentsCreatePerMinute = 1
	}))
	t.Cleanup(func() { SetSource(newTestSource(nil)) })
	handler := passThrough()
	first := serveLimited(handler, withPrincipal(newRequest("3.3.3.3", "/api/v1/moments/create"), "user_a"))
	if first.Code != http.StatusOK {
		t.Fatalf("first create should pass, got %d", first.Code)
	}
	second := serveLimited(handler, withPrincipal(newRequest("3.3.3.3", "/api/v1/moments/create"), "user_a"))
	if code := testdb.DecodeAPICode(t, second); code != httpx.CodeRateLimited {
		t.Fatalf("expected moments_create limit for user_a, got code %d", code)
	}
	other := serveLimited(handler, withPrincipal(newRequest("3.3.3.4", "/api/v1/moments/create"), "user_b"))
	if other.Code != http.StatusOK {
		t.Fatalf("user_b create should pass, got %d", other.Code)
	}
}

// TestAuthSigninUnlimitedWhenConfigZero 验证限流阈值为 0 表示该维度关闭：
// 未配置的 signin 限流不拦截连续请求。
func TestAuthSigninUnlimitedWhenConfigZero(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.AuthSigninPer10Minutes = 0
	}))
	t.Cleanup(func() { SetSource(newTestSource(nil)) })
	handler := passThrough()
	for i := 0; i < 5; i++ {
		recorder := serveLimited(handler, newRequest("4.4.4.4", "/public/v1/auth/signin"))
		if recorder.Code != http.StatusOK {
			t.Fatalf("signin %d should pass when limit disabled, got %d", i, recorder.Code)
		}
	}
}

// TestWritePerMinuteAppliesToMutatingRoutes 验证 write_per_minute 兜住
// 未命中专用规则的所有写端点（匿名按 IP 维度）。
func TestWritePerMinuteAppliesToMutatingRoutes(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.WritePerMinute = 1
	}))
	t.Cleanup(func() { SetSource(newTestSource(nil)) })
	handler := passThrough()
	first := serveLimited(handler, newRequest("5.5.5.5", "/api/v1/groups/create"))
	if first.Code != http.StatusOK {
		t.Fatalf("first write should pass, got %d", first.Code)
	}
	second := serveLimited(handler, newRequest("5.5.5.5", "/api/v1/groups/update"))
	if code := testdb.DecodeAPICode(t, second); code != httpx.CodeRateLimited {
		t.Fatalf("expected write_per_minute to limit second write, got code %d", code)
	}
}

// serveLimited 用给定 handler 处理单个请求并返回其 recorder。
func serveLimited(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// TestAdminRoutesSkipUserPerHour 验证 /admin/api/v1 请求不叠加 user_per_hour：
// 管理面已有独立读写桶，叠加小时桶会把 admin_read 120/min 名义预算截断成
// ~16/min（user_per_hour 默认 1000/h）。普通登录请求仍受 user_per_hour 限制。
func TestAdminRoutesSkipUserPerHour(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.UserPerHour = 1
		cfg.RateLimits.AdminReadPerMinute = 60
	}))
	t.Cleanup(func() { SetSource(newTestSource(nil)) })
	handler := passThrough()
	for i := 0; i < 3; i++ {
		recorder := serveLimited(handler, withPrincipal(newRequest("7.7.7.7", "/admin/api/v1/users/search"), "admin_a"))
		if recorder.Code != http.StatusOK {
			t.Fatalf("admin read %d should skip user_per_hour, got %d", i, recorder.Code)
		}
	}
	// 对照组：普通登录路由第二次即被 user_per_hour 拦截。
	if recorder := serveLimited(handler, withPrincipal(newRequest("7.7.7.7", "/api/v1/moments/fetch_nearby"), "normal_a")); recorder.Code != http.StatusOK {
		t.Fatalf("first normal request should pass, got %d", recorder.Code)
	}
	second := serveLimited(handler, withPrincipal(newRequest("7.7.7.7", "/api/v1/moments/fetch_nearby"), "normal_a"))
	if code := testdb.DecodeAPICode(t, second); code != httpx.CodeRateLimited {
		t.Fatalf("expected user_per_hour on normal route, got code %d", code)
	}
}

// TestAllowCodeSendPerAccountIsPerAccount 验证验证码发送的账号维度限流：
// 账号键大小写/首尾空格归一后共享配额，不同账号独立计量；limit<=0 不生效。
func TestAllowCodeSendPerAccountIsPerAccount(t *testing.T) {
	resetStores()
	login := fmt.Sprintf("Throttle_%d@Example.com", time.Now().UnixNano())
	if _, limited := AllowCodeSendPerAccount(login, 2); limited {
		t.Fatal("first send should pass")
	}
	if _, limited := AllowCodeSendPerAccount(strings.ToUpper(login), 2); limited {
		t.Fatal("second send (case-normalized) should pass")
	}
	retry, limited := AllowCodeSendPerAccount("  "+login+"  ", 2)
	if !limited {
		t.Fatal("third send (space-normalized) should be limited")
	}
	if retry < 1 {
		t.Fatalf("retry_after should be >= 1, got %d", retry)
	}
	if _, limited := AllowCodeSendPerAccount(login+"x", 2); limited {
		t.Fatal("different account should be independent")
	}
	if _, limited := AllowCodeSendPerAccount(login, 0); limited {
		t.Fatal("limit<=0 should disable per-account limiting")
	}
}
