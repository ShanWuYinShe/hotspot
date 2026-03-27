package ratelimit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
)

func resetStores() {
	storeMu.Lock()
	defer storeMu.Unlock()
	windows = map[string][]time.Time{}
	tokens = map[string]*tokenBucket{}
}

func newTestSource(apply func(*appconfig.AppConfig)) appconfig.Source {
	cfg := appconfig.NewStaticSource(config.Config{})
	if apply == nil {
		return cfg
	}
	_, _ = cfg.Update(nil, func(current appconfig.AppConfig) appconfig.AppConfig {
		apply(&current)
		return current
	})
	return cfg
}

func decodeCode(t *testing.T, rr *httptest.ResponseRecorder) int {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.Code
}

func TestAuthRegisterRateLimitByIP(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.GlobalPerHour = 0
		cfg.RateLimits.AuthRegisterPer10Minutes = 1
	}))
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ok", http.StatusOK)
	}))
	first := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	first.Header.Set("CF-Connecting-IP", "1.1.1.1")
	handler.ServeHTTP(httptest.NewRecorder(), first)
	second := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	second.Header.Set("CF-Connecting-IP", "1.1.1.1")
	secondRR := httptest.NewRecorder()
	handler.ServeHTTP(secondRR, second)
	if code := decodeCode(t, secondRR); code != 10500 {
		t.Fatalf("unexpected code: %d", code)
	}
	if secondRR.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestSearchRateLimitUsesUserWhenAuthenticated(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.GlobalPerHour = 0
		cfg.RateLimits.UserPerHour = 0
		cfg.RateLimits.SearchPerMinute = 1
	}))
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ok", http.StatusOK)
	}))
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/suggest", nil)
		req.Header.Set("CF-Connecting-IP", "2.2.2.2")
		req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice"}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if i == 1 && decodeCode(t, rr) != 10500 {
			t.Fatal("expected authenticated user limit")
		}
	}
}

func TestAdminReadWriteUseDifferentLimits(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.GlobalPerHour = 0
		cfg.RateLimits.UserPerHour = 0
		cfg.RateLimits.AdminReadPerMinute = 1
		cfg.RateLimits.AdminWritePerMinute = 1
	}))
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ok", http.StatusOK)
	}))
	readReq1 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/audit/list", nil)
	readReq1 = readReq1.WithContext(platformauth.WithPrincipal(readReq1.Context(), platformauth.Principal{UserID: "admin_1"}))
	handler.ServeHTTP(httptest.NewRecorder(), readReq1)
	readReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/audit/list", nil)
	readReq2 = readReq2.WithContext(platformauth.WithPrincipal(readReq2.Context(), platformauth.Principal{UserID: "admin_1"}))
	readRR2 := httptest.NewRecorder()
	handler.ServeHTTP(readRR2, readReq2)
	if decodeCode(t, readRR2) != 10500 {
		t.Fatal("expected admin read limit")
	}
	writeReq1 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/ban", nil)
	writeReq1 = writeReq1.WithContext(platformauth.WithPrincipal(writeReq1.Context(), platformauth.Principal{UserID: "admin_2"}))
	handler.ServeHTTP(httptest.NewRecorder(), writeReq1)
	writeReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/ban", nil)
	writeReq2 = writeReq2.WithContext(platformauth.WithPrincipal(writeReq2.Context(), platformauth.Principal{UserID: "admin_2"}))
	writeRR2 := httptest.NewRecorder()
	handler.ServeHTTP(writeRR2, writeReq2)
	if decodeCode(t, writeRR2) != 10500 {
		t.Fatal("expected admin write limit")
	}
}

func TestSlidingWindowExpires(t *testing.T) {
	resetStores()
	storeMu.Lock()
	windows["test::ip::1.1.1.1"] = []time.Time{time.Now().UTC().Add(-2 * time.Minute)}
	storeMu.Unlock()
	if retry, limited := allow(time.Now().UTC(), rule{name: "test", strategy: "sliding", limit: 1, window: time.Minute, scope: "ip", key: "1.1.1.1"}); limited || retry != 0 {
		t.Fatalf("expected expired window to allow request, retry=%d limited=%v", retry, limited)
	}
}

func TestAllowWebsocketSubscribeRespectsRateConfig(t *testing.T) {
	resetStores()
	SetSource(newTestSource(func(cfg *appconfig.AppConfig) {
		cfg.RateLimits.WSSubscribePerSecond = 1
	}))
	if retry, limited := AllowWebsocketSubscribe(context.Background(), "user_alice", "conn-1"); limited || retry != 0 {
		t.Fatalf("unexpected first websocket limit result: retry=%d limited=%v", retry, limited)
	}
	if retry, limited := AllowWebsocketSubscribe(context.Background(), "user_alice", "conn-1"); !limited || retry <= 0 {
		t.Fatalf("expected websocket limit, retry=%d limited=%v", retry, limited)
	}
}
