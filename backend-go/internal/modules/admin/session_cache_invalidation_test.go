package admin

import (
	"context"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

// TestSetUserBanInvalidatesSessionCache：admin 封禁直接 UPDATE auth_sessions 表，
// 不经平台 auth 的 SessionStore 装饰器，必须显式失效会话缓存（否则被撤会话在
// 缓存 TTL 内仍可通过认证）。
func TestSetUserBanInvalidatesSessionCache(t *testing.T) {
	svc, repo, _, _ := newTestService()
	previousHub := realtime.DefaultHub()
	realtime.SetDefaultHub(realtime.NewHub())
	t.Cleanup(func() { realtime.SetDefaultHub(previousHub) })

	var invalidated []string
	svc.SetSessionCacheInvalidator(func(sessionIDs ...string) {
		invalidated = append(invalidated, sessionIDs...)
	})

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "admin", SessionID: "sess_admin"})
	apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam"})
	if err != nil {
		t.Fatalf("SetUserBan err: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("SetUserBan apiErr: %+v", apiErr)
	}

	if sessions, listErr := repo.ListUserSessions(ctx, "user_bob"); listErr != nil || len(sessions) != 0 {
		t.Fatalf("sessions after ban = %v err=%v, want none", sessions, listErr)
	}
	found := false
	for _, id := range invalidated {
		if id == "sess_user_bob_1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("session cache invalidator got %v, want sess_user_bob_1", invalidated)
	}
}
