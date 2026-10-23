package groups

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/hotcache"
)

// TestServiceAcceptInviteInvalidatesMemberCache 防回归（审查 P1-1）：
// 接受邀请入群成功后必须失效群成员热缓存。ensureMember（groupmessages/
// realtime 消费同一键族）会把「非成员」结果以 "0" 负缓存（防穿透），漏删
// 会让受邀者在 TTL 内发消息/订阅仍被判非成员——JoinGroup/ApproveApplication
// 两条入群路径均已失效，本测试钉住 AcceptInvite 的同款行为。
func TestServiceAcceptInviteInvalidatesMemberCache(t *testing.T) {
	cache := hotcache.New(nil)
	defer cache.Close()
	repo := &stubGroupsRepo{
		getInvite: func(inviteID string) (GroupInviteItem, string, bool) {
			return GroupInviteItem{
				InviteID: "inv1",
				Group:    GroupInfo{GroupID: "g1", Name: "test", MembersCount: 1},
				Inviter:  UserInfo{UserID: "owner1"},
				Status:   ApplicationStatusPending,
			}, "u2", true
		},
	}
	svc := NewServiceWithRepo(repo)
	svc.SetMemberCache(cache)

	// 模拟受邀者在接受邀请前尝试过发消息：ensureMember 写入 "0" 负缓存。
	key := hotcache.GroupMemberKey("g1", "u2")
	cache.Set(context.Background(), key, "0", hotcache.GroupMemberTTL)
	if v, ok := cache.Get(context.Background(), key); !ok || v != "0" {
		t.Fatalf(`precondition failed: expected cached "0", got %q ok=%v`, v, ok)
	}

	if err := svc.AcceptInvite(grpCtx("u2"), AcceptInviteRequest{InviteID: "inv1"}); err != nil {
		t.Fatalf("AcceptInvite failed: %v", err)
	}
	if v, ok := cache.Get(context.Background(), key); ok {
		t.Fatalf("member cache should be invalidated after accepting invite, got cached value %q", v)
	}
}
