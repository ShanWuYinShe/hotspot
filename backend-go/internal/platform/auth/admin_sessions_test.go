package auth

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

// TestAdminSessionListAndRevoke：管理端列出用户未撤销会话（最近活跃优先）并对
// 任意会话撤销（不校验所有权）；重复撤销返回 false。
func TestAdminSessionListAndRevoke(t *testing.T) {
	store := NewInMemorySessionStore()
	svc := NewServiceWithStores(config.Config{AuthTokenSecret: "test-secret"}, store, NewInMemoryRefreshStore())
	ctx := context.Background()
	now := time.Now().UTC()
	for _, session := range []SessionRecord{
		{SessionID: "s1", UserID: "u1", DeviceID: "d1", CreatedAt: now.Add(-time.Hour), LastActiveAt: now.Add(-time.Minute)},
		{SessionID: "s2", UserID: "u1", DeviceID: "d2", CreatedAt: now.Add(-time.Hour), LastActiveAt: now},
	} {
		if err := store.Save(ctx, session); err != nil {
			t.Fatalf("save session: %v", err)
		}
	}

	ids, err := svc.ListActiveSessionIDsAsAdmin(ctx, "u1")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(ids) != 2 || ids[0] != "s2" || ids[1] != "s1" {
		t.Fatalf("session order = %v, want [s2 s1]", ids)
	}

	revoked, err := svc.RevokeSessionAsAdmin(ctx, "s1")
	if err != nil || !revoked {
		t.Fatalf("revoke s1 = %v err=%v, want true", revoked, err)
	}
	ids, err = svc.ListActiveSessionIDsAsAdmin(ctx, "u1")
	if err != nil || len(ids) != 1 || ids[0] != "s2" {
		t.Fatalf("sessions after revoke = %v err=%v, want [s2]", ids, err)
	}
	revoked, err = svc.RevokeSessionAsAdmin(ctx, "s1")
	if err != nil || revoked {
		t.Fatalf("repeat revoke = %v err=%v, want false/nil", revoked, err)
	}
}
