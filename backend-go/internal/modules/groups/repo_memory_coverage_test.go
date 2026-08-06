package groups

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryAnnouncementAndApplicationLifecycle(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// 创建群（owner）
	created, err := repo.Create(ctx, "u1", CreateGroupRequest{Name: "测试群", JoinMethod: GroupJoinMethodDirect})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	// 成员加入
	if _, err := repo.JoinGroup(ctx, "u2", created.Group.GroupID, nil, nil); err != nil {
		t.Fatalf("join: %v", err)
	}

	// 公告 CRUD
	ann, err := repo.CreateAnnouncement(ctx, "u1", CreateGroupAnnouncementRequest{GroupID: created.Group.GroupID, Title: "公告", Content: "内容"})
	if err != nil || ann.AnnouncementID == "" {
		t.Fatalf("create announcement: %#v err=%v", ann, err)
	}
	anns, err := repo.ListAnnouncements(ctx, created.Group.GroupID)
	if err != nil {
		t.Fatalf("list announcements: %#v err=%v", anns, err)
	}
	if _, err := repo.ListAnnouncements(ctx, "missing"); err != ErrGroupNotFound {
		t.Fatalf("list announcements missing group: expected not found, got %v", err)
	}
	title := "更新"
	if err := repo.UpdateAnnouncement(ctx, created.Group.GroupID, UpdateGroupAnnouncementRequest{AnnouncementID: ann.AnnouncementID, Title: &title}); err != nil {
		t.Fatalf("update announcement: %v", err)
	}
	if err := repo.DeleteAnnouncement(ctx, created.Group.GroupID, ann.AnnouncementID); err != nil {
		t.Fatalf("delete announcement: %v", err)
	}
	// 群不存在 → 错误
	if err := repo.UpdateAnnouncement(ctx, "missing", UpdateGroupAnnouncementRequest{AnnouncementID: "x"}); err != ErrGroupNotFound {
		t.Fatalf("update announcement missing group: expected not found, got %v", err)
	}
	if err := repo.DeleteAnnouncement(ctx, "missing", "x"); err != ErrGroupNotFound {
		t.Fatalf("delete announcement missing group: expected not found, got %v", err)
	}

	// 成员管理
	if err := repo.SetMemberRole(ctx, "u1", SetMemberRoleRequest{GroupID: created.Group.GroupID, UserID: "u2", Role: GroupRoleAdmin}); err != nil {
		t.Fatalf("set role: %v", err)
	}
	if err := repo.MuteMember(ctx, "u1", MuteMemberRequest{GroupID: created.Group.GroupID, UserID: "u2", Active: true, DurationMinutes: 60}); err != nil {
		t.Fatalf("mute: %v", err)
	}
	// 退出群：内存 repo 的 MyState 恒为 owner 状态，任何成员退出都返回 ErrAccessRestricted
	if err := repo.LeaveGroup(ctx, "u2", created.Group.GroupID); err != ErrAccessRestricted {
		t.Fatalf("leave non-owner: expected access restricted, got %v", err)
	}
	if err := repo.LeaveGroup(ctx, "u1", created.Group.GroupID); err != ErrAccessRestricted {
		t.Fatalf("leave owner: expected access restricted, got %v", err)
	}
	if err := repo.LeaveGroup(ctx, "u1", "missing"); err != ErrGroupNotFound {
		t.Fatalf("leave missing group: expected not found, got %v", err)
	}
}

func TestMemoryInviteLifecycle(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateGroupRequest{Name: "邀请群", JoinMethod: GroupJoinMethodInviteOnly})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	invite, err := repo.InviteMember(ctx, "u1", InviteMemberRequest{GroupID: created.Group.GroupID, UserID: "u2", Message: nil})
	if err != nil || invite.InviteID == "" {
		t.Fatalf("invite: %#v err=%v", invite, err)
	}

	sent, _, err := repo.ListSentInvites(ctx, created.Group.GroupID, nil, 10, "")
	if err != nil || len(sent) != 1 {
		t.Fatalf("list sent invites: %#v err=%v", sent, err)
	}

	if err := repo.AcceptInvite(ctx, "u2", invite.InviteID); err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	if err := repo.AcceptInvite(ctx, "u2", invite.InviteID); err == nil {
		t.Fatal("double accept must fail")
	}
	if err := repo.AcceptInvite(ctx, "u2", "missing"); err == nil {
		t.Fatal("accept missing must fail")
	}

	// 新邀请 → 拒绝
	invite2, err := repo.InviteMember(ctx, "u1", InviteMemberRequest{GroupID: created.Group.GroupID, UserID: "u3"})
	if err != nil {
		t.Fatalf("invite2: %v", err)
	}
	if err := repo.RejectInvite(ctx, invite2.InviteID); err != nil {
		t.Fatalf("reject invite: %v", err)
	}
	if err := repo.RejectInvite(ctx, invite2.InviteID); err == nil {
		t.Fatal("double reject must fail")
	}

	// 撤销邀请
	invite3, err := repo.InviteMember(ctx, "u1", InviteMemberRequest{GroupID: created.Group.GroupID, UserID: "u4"})
	if err != nil {
		t.Fatalf("invite3: %v", err)
	}
	if err := repo.RevokeInvite(ctx, created.Group.GroupID, invite3.InviteID); err != nil {
		t.Fatalf("revoke invite: %v", err)
	}
	// 不存在的邀请撤销为 no-op（幂等）
	if err := repo.RevokeInvite(ctx, created.Group.GroupID, "missing"); err != nil {
		t.Fatalf("revoke missing invite: %v", err)
	}
	if err := repo.RevokeInvite(ctx, "missing", "x"); err != ErrGroupNotFound {
		t.Fatalf("revoke missing group: expected not found, got %v", err)
	}
}

func TestMemoryMySettingsAndCancelApplication(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	created, err := repo.Create(ctx, "u1", CreateGroupRequest{Name: "设置群", JoinMethod: GroupJoinMethodDirect})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// 非成员更新设置 → ErrNotMember（与 postgres 语义一致）
	if err := repo.UpdateMySettings(ctx, "u2", UpdateGroupMySettingsRequest{GroupID: created.Group.GroupID, IsMuted: boolPtr(true)}); !errors.Is(err, ErrNotMember) {
		t.Fatalf("non-member settings update: %v", err)
	}
	// 成员更新设置
	if _, err := repo.JoinGroup(ctx, "u2", created.Group.GroupID, nil, nil); err != nil {
		t.Fatalf("join: %v", err)
	}
	if err := repo.UpdateMySettings(ctx, "u2", UpdateGroupMySettingsRequest{GroupID: created.Group.GroupID, IsMuted: boolPtr(true)}); err != nil {
		t.Fatalf("member settings update: %v", err)
	}

	// 申请-取消：requires_approval 群 JoinGroup 创建申请
	repo2 := NewInMemoryRepository()
	approval := true
	created2, err := repo2.Create(ctx, "u1", CreateGroupRequest{Name: "申请群", JoinMethod: GroupJoinMethodDirect, RequiresApproval: &approval})
	if err != nil {
		t.Fatalf("create2: %v", err)
	}
	joinRes, err := repo2.JoinGroup(ctx, "u2", created2.Group.GroupID, nil, nil)
	if err != nil || joinRes.Status != JoinGroupStatusPendingApproval || joinRes.Application == nil {
		t.Fatalf("join approval group: %#v err=%v", joinRes, err)
	}
	apps, _, err := repo2.ListApplications(ctx, created2.Group.GroupID, nil, 10, "")
	if err != nil || len(apps) != 1 {
		t.Fatalf("list applications: %#v err=%v", apps, err)
	}
	// 取消申请（CancelApplication 幂等，重复取消仍为 nil）
	if err := repo2.CancelApplication(ctx, apps[0].ApplicationID); err != nil {
		t.Fatalf("cancel application: %v", err)
	}
	if err := repo2.CancelApplication(ctx, apps[0].ApplicationID); err != nil {
		t.Fatalf("double cancel: %v", err)
	}
	if err := repo2.CancelApplication(ctx, "missing"); err != ErrGroupNotFound {
		t.Fatalf("cancel missing: expected not found, got %v", err)
	}
	// 批准/拒绝已取消的申请 → 错误
	if err := repo2.ApproveApplication(ctx, "u1", apps[0].ApplicationID); err != ErrAlreadyProcessed {
		t.Fatalf("approve cancelled: expected already processed, got %v", err)
	}
	if err := repo2.RejectApplication(ctx, apps[0].ApplicationID, nil); err != ErrAlreadyProcessed {
		t.Fatalf("reject cancelled: expected already processed, got %v", err)
	}
}

func boolPtr(v bool) *bool { return &v }
