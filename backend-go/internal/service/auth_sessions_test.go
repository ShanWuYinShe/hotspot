package service

import (
	"context"
	"testing"
	"time"

	"backend-go/internal/auth"
	"backend-go/internal/domain"
	apierr "backend-go/internal/errors"
)

// --- Logout & session management ---

func TestAuthService_Logout_BlacklistsTokensAndRevokesSession(t *testing.T) {
	deps := newAuthServiceTestDeps()
	ctx := context.Background()

	// 构造一个已有会话，便于验证 RevokeSession 被调用
	now := time.Now().UTC()
	sess := domain.Session{
		SessionID:    "sess-1",
		UserID:       "user-1",
		CreatedAt:    now,
		LastActiveAt: now,
	}
	deps.sessions.sessions = append(deps.sessions.sessions, sess)

	// 注入当前请求的 AuthInfo，包含 access/refresh token 与过期时间
	accessExp := now.Add(15 * time.Minute)
	refreshExp := now.Add(24 * time.Hour)
	ctx = auth.WithAuthInfo(ctx, &auth.AuthInfo{
		UserID:           "user-1",
		SessionID:        "sess-1",
		AccessToken:      "access-token-current",
		RefreshToken:     "refresh-token-current",
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	})

	if err := deps.svc.Logout(ctx); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	// 当前会话应被标记 revoked
	if deps.sessions.sessions[0].RevokedAt == nil {
		t.Fatalf("expected session to be revoked after Logout")
	}

	// access / refresh token 应被写入黑名单
	var tokens []string
	for _, entry := range deps.black.blacklisted {
		tokens = append(tokens, entry.token)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens blacklisted, got %d (%v)", len(tokens), tokens)
	}
}

func TestAuthService_Logout_WithoutAuthInfoReturns10100(t *testing.T) {
	deps := newAuthServiceTestDeps()
	ctx := context.Background()
	if err := deps.svc.Logout(ctx); err == nil {
		t.Fatalf("expected error when Logout called without auth context, got nil")
	} else {
		if e, ok := err.(*apierr.APIError); ok {
			if e.Code != 10100 {
				t.Fatalf("expected code 10100 for missing auth, got %d", e.Code)
			}
		}
	}
}

func TestAuthService_ListSessions_FlagsCurrentSessionAndFiltersRevoked(t *testing.T) {
	deps := newAuthServiceTestDeps()
	ctx := context.Background()

	now := time.Now().UTC()
	revokedAt := now
	deps.sessions.sessions = []domain.Session{
		{
			SessionID:    "sess-current",
			UserID:       "user-1",
			CreatedAt:    now.Add(-2 * time.Hour),
			LastActiveAt: now,
		},
		{
			SessionID:    "sess-revoked",
			UserID:       "user-1",
			CreatedAt:    now.Add(-3 * time.Hour),
			LastActiveAt: now.Add(-1 * time.Hour),
			RevokedAt:    &revokedAt,
		},
	}

	ctx = auth.WithAuthInfo(ctx, &auth.AuthInfo{
		UserID:    "user-1",
		SessionID: "sess-current",
	})

	items, err := deps.svc.ListSessions(ctx, "")
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected only active session to be returned, got %d", len(items))
	}
	if items[0].SessionID != "sess-current" || !items[0].IsCurrent {
		t.Fatalf("expected current session flagged as is_current, got %+v", items[0])
	}
}

func TestAuthService_RevokeAllSessions_RevokesOthersAndReturnsCount(t *testing.T) {
	deps := newAuthServiceTestDeps()
	ctx := context.Background()

	now := time.Now().UTC()
	deps.sessions.sessions = []domain.Session{
		{SessionID: "sess-current", UserID: "user-1", CreatedAt: now, LastActiveAt: now},
		{SessionID: "sess-old-1", UserID: "user-1", CreatedAt: now, LastActiveAt: now},
		{SessionID: "sess-old-2", UserID: "user-1", CreatedAt: now, LastActiveAt: now},
	}

	ctx = auth.WithAuthInfo(ctx, &auth.AuthInfo{
		UserID:    "user-1",
		SessionID: "sess-current",
	})

	res, err := deps.svc.RevokeAllSessions(ctx, "")
	if err != nil {
		t.Fatalf("RevokeAllSessions returned error: %v", err)
	}
	if res == nil || res.RevokedCount != 2 {
		t.Fatalf("expected revoked_count=2, got %+v", res)
	}

	var revoked, active int
	for _, s := range deps.sessions.sessions {
		if s.RevokedAt != nil {
			revoked++
		} else {
			active++
		}
	}
	if revoked != 2 || active != 1 {
		t.Fatalf("expected 2 revoked and 1 active, got revoked=%d active=%d", revoked, active)
	}
}
