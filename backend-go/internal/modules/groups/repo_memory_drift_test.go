package groups

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 本文件锁定 chat 域审查 P2-8 的内存实现漂移修复（内存实现是「DB 未配置时
// 优雅降级」的生产路径，不是纯测试桩）。

// TestInMemoryDismissClearsBansAndAnnouncements 审查 P2-8-3：解散必须与 PG
// 的六表清理同口径，此前漏清 banned/announcements，同 ID 重建群会读到幽灵数据。
func TestInMemoryDismissClearsBansAndAnnouncements(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	created, err := repo.Create(ctx, "owner1", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	gid := created.Group.GroupID
	if _, err := repo.JoinGroup(ctx, "u2", gid, nil); err != nil {
		t.Fatalf("JoinGroup failed: %v", err)
	}
	reason := "spam"
	if err := repo.BanMember(ctx, "owner1", BanMemberRequest{GroupID: gid, UserID: "u2", Reason: &reason}); err != nil {
		t.Fatalf("BanMember failed: %v", err)
	}
	ann, err := repo.CreateAnnouncement(ctx, "owner1", CreateGroupAnnouncementRequest{GroupID: gid, Title: "t", Content: "c"})
	if err != nil {
		t.Fatalf("CreateAnnouncement failed: %v", err)
	}

	if err := repo.Dismiss(ctx, gid); err != nil {
		t.Fatalf("Dismiss failed: %v", err)
	}
	// 群已解散：列表接口按 PG 口径返回「群组不存在」，故直接核对内存映射无残留。
	if bans, ok := repo.banned[gid]; ok && len(bans) > 0 {
		t.Fatalf("dismiss must clear bans, got %+v", bans)
	}
	if _, ok := repo.announcements[ann.AnnouncementID]; ok {
		t.Fatalf("dismiss must clear announcement %s", ann.AnnouncementID)
	}
	if _, found, err := repo.GetAnnouncement(ctx, gid, ann.AnnouncementID); err != nil || found {
		t.Fatalf("dismissed group announcement still readable: found=%v err=%v", found, err)
	}
}

// TestInMemoryRejectApplicationKeepsMessage 审查 P2-8-2：拒绝原因不得覆盖
// 申请人的原始留言（PG 只更新状态/处理人）。
func TestInMemoryRejectApplicationKeepsMessage(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	created, err := repo.Create(ctx, "owner1", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	gid := created.Group.GroupID
	message := "我想加入"
	repo.applications["app1"] = inMemoryApplicationRecord{
		ApplicationID: "app1",
		GroupID:       gid,
		ApplicantID:   "u2",
		Status:        ApplicationStatusPending,
		Message:       &message,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	reason := "人数已满"
	if err := repo.RejectApplication(ctx, "app1", &reason); err != nil {
		t.Fatalf("RejectApplication failed: %v", err)
	}
	app, found, err := repo.GetApplication(ctx, "app1")
	if err != nil || !found {
		t.Fatalf("GetApplication: found=%v err=%v", found, err)
	}
	if app.Status != ApplicationStatusRejected {
		t.Fatalf("status=%s want rejected", app.Status)
	}
	if app.Message == nil || *app.Message != "我想加入" {
		t.Fatalf("applicant message overwritten: %v", app.Message)
	}
}

// TestInMemorySearchMembersMatchesSubstring 审查 P2-8-7：PG 是 ILIKE 子串
// 匹配，内存实现此前只做精确匹配，单测会掩盖生产行为。
func TestInMemorySearchMembersMatchesSubstring(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	created, err := repo.Create(ctx, "owner1", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	gid := created.Group.GroupID
	if _, err := repo.JoinGroup(ctx, "u2", gid, nil); err != nil {
		t.Fatalf("JoinGroup failed: %v", err)
	}

	members, _, err := repo.SearchMembers(ctx, gid, "OWNER", nil, 10, "")
	if err != nil {
		t.Fatalf("SearchMembers failed: %v", err)
	}
	found := false
	for _, m := range members {
		if m.User.UserID == "owner1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("substring search must match owner1, got %+v", members)
	}
}

// TestInMemoryAcceptInviteCapacityKeepsPending 审查 P2-8-1：容量检查必须先于
// 状态变更，群满时邀请不能被污染成「已接受」。
func TestInMemoryAcceptInviteCapacityKeepsPending(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	created, err := repo.Create(ctx, "owner1", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	gid := created.Group.GroupID
	// 上限 1：owner 已占满。
	group := repo.groups[gid]
	group.MaxMembers = 1
	repo.groups[gid] = group

	inv, err := repo.InviteMember(ctx, "owner1", InviteMemberRequest{GroupID: gid, UserID: "u2"}, nil)
	if err != nil {
		t.Fatalf("InviteMember failed: %v", err)
	}
	inviteID := ""
	if inv.InviteID != nil {
		inviteID = *inv.InviteID
	}
	if err := repo.AcceptInvite(ctx, "u2", inviteID); !errors.Is(err, ErrGroupFull) {
		t.Fatalf("AcceptInvite err=%v want ErrGroupFull", err)
	}
	// 状态必须仍是 pending（未被半提交污染）。
	invites, _, err := repo.ListMyInvites(ctx, "u2", nil, 10, "")
	if err != nil || len(invites) != 1 {
		t.Fatalf("ListMyInvites: n=%d err=%v", len(invites), err)
	}
	if invites[0].Status != ApplicationStatusPending {
		t.Fatalf("invite status=%s want pending", invites[0].Status)
	}
}

// TestInMemoryListMyInvitesPaginates 审查 P2-8-6：内存实现此前忽略
// limit/cursor（全量返回、next_cursor 恒空），与 PG 的 keyset 翻页不一致。
func TestInMemoryListMyInvitesPaginates(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	var gid string
	for i := 0; i < 3; i++ {
		created, err := repo.Create(ctx, "owner1", validCreateReq())
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		gid = created.Group.GroupID
		// 同一被邀请人、三个群，制造 3 条邀请。
		if _, err := repo.InviteMember(ctx, "owner1", InviteMemberRequest{GroupID: gid, UserID: "u2"}, nil); err != nil {
			t.Fatalf("InviteMember failed: %v", err)
		}
		// 保证排序稳定：时间戳递增。
		time.Sleep(2 * time.Millisecond)
	}
	page1, cursor, err := repo.ListMyInvites(ctx, "u2", nil, 2, "")
	if err != nil {
		t.Fatalf("ListMyInvites page1: %v", err)
	}
	if len(page1) != 2 || cursor == "" {
		t.Fatalf("page1 n=%d cursor=%q want 2 items + cursor", len(page1), cursor)
	}
	page2, next, err := repo.ListMyInvites(ctx, "u2", nil, 2, cursor)
	if err != nil {
		t.Fatalf("ListMyInvites page2: %v", err)
	}
	if len(page2) != 1 || next != "" {
		t.Fatalf("page2 n=%d next=%q want 1 item + end", len(page2), next)
	}
	seen := make(map[string]bool)
	for _, item := range append(page1, page2...) {
		if seen[item.InviteID] {
			t.Fatalf("duplicate invite across pages: %s", item.InviteID)
		}
		seen[item.InviteID] = true
	}
	_ = gid
}
