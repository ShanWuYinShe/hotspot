package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

func testCacheConfig() config.Config {
	return config.Config{AuthTokenSecret: strings.Repeat("x", 32)}
}

// TestServiceInvalidateSessionCache：admin 封禁等绕过 SessionStore 的调用方通过
// Service.InvalidateSessionCache 让缓存立即失效（否则被撤会话在 TTL 内仍可认证）。
func TestServiceInvalidateSessionCache(t *testing.T) {
	ctx := context.Background()
	inner := newCountingSessionStore()
	if err := inner.Save(ctx, SessionRecord{SessionID: "s1", UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	svc := NewServiceWithStores(testCacheConfig(), cache, nil)
	if _, _, err := svc.GetSession(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.GetSession(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if got := inner.getCount(); got != 1 {
		t.Fatalf("inner Get calls = %d, want 1 (second served from cache)", got)
	}
	svc.InvalidateSessionCache("s1")
	if _, _, err := svc.GetSession(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if got := inner.getCount(); got != 2 {
		t.Fatalf("inner Get calls = %d, want 2 after InvalidateSessionCache", got)
	}
}

// TestServiceInvalidateSessionCacheNoopWithoutCache：未包装缓存时调用不应 panic。
func TestServiceInvalidateSessionCacheNoopWithoutCache(t *testing.T) {
	inner := NewInMemorySessionStore()
	svc := NewServiceWithStores(testCacheConfig(), inner, nil)
	svc.InvalidateSessionCache("s1", "s2")
}
