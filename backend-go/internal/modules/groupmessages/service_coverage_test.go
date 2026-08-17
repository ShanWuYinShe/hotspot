package groupmessages

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryRecallAsAdminAndAuxLookups(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// 空群：GetReadCount / LatestMessageID 报 not found
	if _, err := repo.GetReadCount(ctx, "g1", "m1"); err != ErrMessageNotFound {
		t.Fatalf("empty read count: %v", err)
	}
	if _, err := repo.LatestMessageID(ctx, "g1"); err != ErrMessageNotFound {
		t.Fatalf("empty latest: %v", err)
	}

	// 发送两条消息（顺序错开，LatestMessageID 取最新）
	first, err := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "first"})
	if err != nil {
		t.Fatalf("send1: %v", err)
	}
	second, err := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u2", Type: "text", Content: "second"})
	if err != nil {
		t.Fatalf("send2: %v", err)
	}

	latest, err := repo.LatestMessageID(ctx, "g1")
	if err != nil || latest != second.MessageID {
		t.Fatalf("latest: %q err=%v", latest, err)
	}
	if n, err := repo.GetReadCount(ctx, "g1", first.MessageID); err != nil || n != 0 {
		t.Fatalf("read count: %d err=%v", n, err)
	}
	// 群存在但消息不存在 → not found
	if _, err := repo.GetReadCount(ctx, "g1", "missing"); err != ErrMessageNotFound {
		t.Fatalf("read count missing message: %v", err)
	}

	// FilterMemberIDs：只保留真实成员
	repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "again"})
	members, err := repo.FilterMemberIDs(ctx, "g1", []string{"u1", "u2", "nobody"})
	if err != nil || len(members) != 2 || members[0] != "u1" || members[1] != "u2" {
		t.Fatalf("filter members: %v err=%v", members, err)
	}

	// MessageGroupID：存在与缺失
	if gid, found, err := repo.MessageGroupID(ctx, second.MessageID); err != nil || !found || gid != "g1" {
		t.Fatalf("message group id: %q found=%v err=%v", gid, found, err)
	}
	if _, found, err := repo.MessageGroupID(ctx, "missing"); err != nil || found {
		t.Fatalf("missing group id: found=%v err=%v", found, err)
	}

	// RecallAsAdmin：管理员撤回（非发送者也可）
	meta, err := repo.RecallAsAdmin(ctx, second.MessageID)
	if err != nil || meta.MessageID != second.MessageID {
		t.Fatalf("recall as admin: %#v err=%v", meta, err)
	}
	// 已撤回 → ErrMessageRecalled
	if _, err := repo.RecallAsAdmin(ctx, second.MessageID); err != ErrMessageRecalled {
		t.Fatalf("double admin recall: %v", err)
	}
	// 不存在 → ErrMessageNotFound
	if _, err := repo.RecallAsAdmin(ctx, "missing"); err != ErrMessageNotFound {
		t.Fatalf("admin recall missing: %v", err)
	}
	// 普通 Recall 对已撤回消息 → ErrMessageRecalled
	if _, err := repo.Recall(ctx, RecallGroupMessageRequest{MessageID: second.MessageID, UserID: "u2"}); err != ErrMessageRecalled {
		t.Fatalf("recall recalled: %v", err)
	}
}

func TestServiceLookupAndVisibilityHelpers(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := gmAuthCtx("u1")
	repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "hi"})

	svc := NewServiceWithRepoAndUsers(repo, nil)
	// NewServiceWithDeps 构造（mentions/shares nil 路径）
	svc2 := NewServiceWithDeps(repo, nil, nil, nil)
	if svc2 == nil || svc2.repo == nil {
		t.Fatal("NewServiceWithDeps must wire repo")
	}

	// LookupMessage 转发 MessageExists
	sent, _ := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u2", Type: "text", Content: "second"})
	if found, senderID, err := svc.LookupMessage(ctx, sent.MessageID); err != nil || !found || senderID != "u2" {
		t.Fatalf("lookup: found=%v sender=%q err=%v", found, senderID, err)
	}

	// IsReactionTargetVisible：成员可见，非成员不可见
	if found, visible, err := svc.IsReactionTargetVisible(ctx, sent.MessageID); err != nil || !found || !visible {
		t.Fatalf("member visible: found=%v visible=%v err=%v", found, visible, err)
	}
	outsiderCtx := gmAuthCtx("outsider")
	if found, visible, err := svc.IsReactionTargetVisible(outsiderCtx, sent.MessageID); err != nil || !found || visible {
		t.Fatalf("outsider visibility: found=%v visible=%v err=%v", found, visible, err)
	}
	if found, _, err := svc.IsReactionTargetVisible(ctx, "missing"); err != nil || found {
		t.Fatalf("missing visibility: found=%v err=%v", found, err)
	}

	// resolveGroupName：有群名返回群名，无群名回退空串
	repo.SetGroupName("g1", "热群")
	if name := svc.resolveGroupName(ctx, "g1"); name != "热群" {
		t.Fatalf("resolve group name: %q", name)
	}
	if name := svc.resolveGroupName(ctx, "missing"); name != "" {
		t.Fatalf("missing group name: %q", name)
	}
}

func TestServiceIsReactionTargetVisibleRepoError(t *testing.T) {
	repo := &stubRepo{messageGroupIDErr: errors.New("boom")}
	svc := NewServiceWithRepoAndUsers(repo, nil)
	ctx := context.Background()
	if _, _, err := svc.IsReactionTargetVisible(ctx, "m1"); err == nil {
		t.Fatal("repo error must propagate")
	}
}
