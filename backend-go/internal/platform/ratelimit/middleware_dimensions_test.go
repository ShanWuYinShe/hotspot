package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
