package auth

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

// P2-4：宽限期外重用已轮换的旧 refresh token = 盗用强信号，
// 必须吊销整个会话（整条 refresh 链 + access token 同源 SessionID），
// 而非单点拒绝后放任链继续有效。宽限期内复用仍合法返回新链 token。
func TestRefreshReuseOutsideGraceRevokesSessionChain(t *testing.T) {
	svc := NewServiceWithStores(config.Config{}, NewInMemorySessionStore(), NewRefreshStore())
	ctx := context.Background()

	issued, err := svc.IssueSession(ctx, "user_1")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	refreshes := svc.refreshes.(*RefreshStore)
	sessions := svc.sessions.(*InMemorySessionStore)

	// 第一次刷新：旧 token 进入 30s 宽限期，返回新 token。
	first, err := svc.RefreshSession(ctx, issued.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	// 宽限期内重用旧 token：合法并发场景，返回新链 token，会话不受影响。
	replayed, err := svc.RefreshSession(ctx, issued.RefreshToken)
	if err != nil {
		t.Fatalf("in-grace replay must succeed: %v", err)
	}
	if replayed.RefreshToken != first.RefreshToken {
		t.Fatal("in-grace replay must return the winning chain token")
	}

	// 宽限期过期后重用旧 token：盗用信号 → 吊销会话并拒绝。
	refreshes.mu.Lock()
	stale := refreshes.items[issued.RefreshToken]
	stale.GraceUntil = time.Now().UTC().Add(-time.Second)
	refreshes.items[issued.RefreshToken] = stale
	refreshes.mu.Unlock()

	if _, err := svc.RefreshSession(ctx, issued.RefreshToken); err == nil {
		t.Fatal("out-of-grace reuse must be rejected")
	}
	session, ok, err := sessions.Get(ctx, first.Principal.SessionID)
	if err != nil || !ok {
		t.Fatalf("session should still be readable: ok=%v err=%v", ok, err)
	}
	if session.RevokedAt == nil {
		t.Fatal("out-of-grace reuse must revoke the session chain")
	}

	// 链上后续 token 也一并失效（revoked session 检查在 refresh 入口）。
	if _, err := svc.RefreshSession(ctx, first.RefreshToken); err == nil {
		t.Fatal("refresh via chain token must fail after session revocation")
	}
	// access token 校验同样拒绝（AuthenticateRequest 走同一 session 记录）。
	if _, _, ok := svc.AuthenticateRequest(ctx, "Bearer "+first.AccessToken, ""); ok {
		t.Fatal("access token must be invalid after session revocation")
	}
}
