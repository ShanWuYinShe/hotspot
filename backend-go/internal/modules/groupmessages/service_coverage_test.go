package groupmessages

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/types"
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

	// RecallAsAdmin：管理员撤回（非发送者也可，recalled_by=操作者）
	meta, err := repo.RecallAsAdmin(ctx, second.MessageID, "admin_1")
	if err != nil || meta.MessageID != second.MessageID {
		t.Fatalf("recall as admin: %#v err=%v", meta, err)
	}
	// 已撤回 → ErrMessageRecalled
	if _, err := repo.RecallAsAdmin(ctx, second.MessageID, "admin_1"); err != ErrMessageRecalled {
		t.Fatalf("double admin recall: %v", err)
	}
	// 不存在 → ErrMessageNotFound
	if _, err := repo.RecallAsAdmin(ctx, "missing", "admin_1"); err != ErrMessageNotFound {
		t.Fatalf("admin recall missing: %v", err)
	}
	// 回归（spec api_spec_chat.md GroupMessageItem）：recalled_by 记录撤回
	// 操作者（管理员撤回时非作者本人），mentions 落库并恒为数组。
	hist, err := repo.History(ctx, "g1", nil)
	if err != nil {
		t.Fatalf("history after admin recall: %v", err)
	}
	for _, item := range hist.Items {
		if item.MessageID != second.MessageID {
			continue
		}
		if item.RecalledBy == nil || item.RecalledBy["user_id"] != "admin_1" {
			t.Fatalf("expected recalled_by=admin_1, got %#v", item.RecalledBy)
		}
		if item.Mentions == nil {
			t.Fatal("mentions must always be an array (empty at minimum)")
		}
	}
	// 普通 Recall 对已撤回消息 → ErrMessageRecalled
	if _, err := repo.Recall(ctx, 2*time.Minute, RecallGroupMessageRequest{MessageID: second.MessageID, UserID: "u2"}); err != ErrMessageRecalled {
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
	// 已撤回消息按不存在处理（审计 F-11）：撤回后不再接受新的 reaction。
	if _, err := repo.Recall(ctx, 2*time.Minute, RecallGroupMessageRequest{MessageID: sent.MessageID, UserID: "u2"}); err != nil {
		t.Fatalf("recall for visibility test: %v", err)
	}
	if found, visible, err := svc.IsReactionTargetVisible(ctx, sent.MessageID); err != nil || found || visible {
		t.Fatalf("recalled message must be invisible: found=%v visible=%v err=%v", found, visible, err)
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

type stubMentionNotifier struct {
	appendedCount int
	lastTargetID  string
	returnErr     error
}

func (s *stubMentionNotifier) AppendGroupMentionedBatch(ctx context.Context, targetUserIDs []string, actor types.UserInfo, groupID, groupName, messageID, contentPreview string) (map[string]string, error) {
	ids := make(map[string]string, len(targetUserIDs))
	for i, targetUserID := range targetUserIDs {
		s.appendedCount++
		s.lastTargetID = targetUserID
		ids[targetUserID] = "notify_123_" + strconv.Itoa(i)
	}
	return ids, s.returnErr
}

func TestServiceReportSnapshotAndMembership(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := gmAuthCtx("u1")
	svc := NewServiceWithRepo(repo)

	// 1. 发送消息
	sent, err := repo.Send(ctx, SendGroupMessageRequest{
		GroupID:      "g1",
		SenderUserID: "u1",
		Type:         "text",
		Content:      "举报测试消息",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// 2. GetMessageReportSnapshot
	snap, found, err := svc.GetMessageReportSnapshot(ctx, sent.MessageID)
	if err != nil || !found {
		t.Fatalf("snapshot found=%v err=%v", found, err)
	}
	if snap.GroupID != "g1" || snap.Content != "举报测试消息" {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}

	// 3. 不存在的消息快照
	_, found, err = svc.GetMessageReportSnapshot(ctx, "msg_nonexistent")
	if err != nil || found {
		t.Fatalf("expected missing snapshot, found=%v", found)
	}

	// 4. IsGroupMember
	isMember, err := svc.IsGroupMember(ctx, "g1", "u1")
	if err != nil || !isMember {
		t.Fatalf("expected member, got %v err=%v", isMember, err)
	}
	isMember, err = svc.IsGroupMember(ctx, "g1", "u_unknown")
	if err != nil || isMember {
		t.Fatalf("expected non-member, got %v err=%v", isMember, err)
	}
}

func TestServiceReadCountFlusherAndFlush(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := gmAuthCtx("u1")
	svc := NewServiceWithRepo(repo)

	sent, err := repo.Send(ctx, SendGroupMessageRequest{
		GroupID:      "g1",
		SenderUserID: "u1",
		Type:         "text",
		Content:      "已读节流测试",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// 1. 手动触发 flushReadCounts
	svc.flushReadCounts("g1", []string{sent.MessageID, "msg_missing"})

	// 2. scheduleReadCountUpdate
	svc.scheduleReadCountUpdate("g1", []string{sent.MessageID})

	// 3. MarkAllRead
	err = svc.MarkAllRead(ctx)
	if err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}
}

func TestServiceNotifyMentions(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := gmAuthCtx("u1")

	// u1, u2 为成员
	repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "init1"})
	repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u2", Type: "text", Content: "init2"})

	notifier := &stubMentionNotifier{}
	svc := NewServiceWithDeps(repo, nil, notifier, nil)

	// 发送一条 @ u2 和 @ u1 和 @ u_outsider 的消息
	req := SendGroupMessageRequest{
		GroupID:      "g1",
		SenderUserID: "u1",
		Type:         "text",
		Content:      "hello @u2 @outsider",
		Mentions:     []string{"u2", "u1", "outsider"},
	}
	msg := MessageSentResult{
		MessageID: "msg_m1",
		SentAt:    "2026-01-01T00:00:00Z",
	}

	svc.notifyMentions(ctx, "u1", req, msg)
	if notifier.appendedCount != 1 || notifier.lastTargetID != "u2" {
		t.Fatalf("expected 1 mention notification to u2, got %d, lastTarget=%s", notifier.appendedCount, notifier.lastTargetID)
	}
}

func TestHandlerSendValidation(t *testing.T) {
	// 1. Type 空
	if isValidSendRequest(SendGroupMessageRequest{}) {
		t.Fatal("empty type should be invalid")
	}
	// 2. Mentions 超过 20
	mentions21 := make([]string, 21)
	if isValidSendRequest(SendGroupMessageRequest{Type: "text", Content: "hi", Mentions: mentions21}) {
		t.Fatal(">20 mentions should be invalid")
	}
	// 3. text 内容为空或超长
	if isValidSendRequest(SendGroupMessageRequest{Type: "text", Content: ""}) {
		t.Fatal("empty text content should be invalid")
	}
	longContent := strings.Repeat("a", messageContentMaxRunes+1)
	if isValidSendRequest(SendGroupMessageRequest{Type: "text", Content: longContent}) {
		t.Fatal("overlength text content should be invalid")
	}
	// 4. image/video/audio/file 缺少 file_id 或有效
	if isValidSendRequest(SendGroupMessageRequest{Type: "image"}) {
		t.Fatal("empty file_id image should be invalid")
	}
	if !isValidSendRequest(SendGroupMessageRequest{Type: "image", FileID: "f1"}) {
		t.Fatal("valid image should be valid")
	}
	if !isValidSendRequest(SendGroupMessageRequest{Type: "file", FileID: "f1"}) {
		t.Fatal("valid file should be valid")
	}
	// 5. share 缺少 share_code 或有效
	if isValidSendRequest(SendGroupMessageRequest{Type: "share"}) {
		t.Fatal("empty share_code share should be invalid")
	}
	if !isValidSendRequest(SendGroupMessageRequest{Type: "share", ShareCode: "sc123"}) {
		t.Fatal("valid share should be valid")
	}
	// 6. 未知类型
	if isValidSendRequest(SendGroupMessageRequest{Type: "unknown"}) {
		t.Fatal("unknown type should be invalid")
	}
	// 7. 媒体/share 类型携带超长 content 同样无效（审查 P2-4，对齐 Update
	// 路径防绕过：此前仅 text 校验长度，type=image + 合法 file_id + 超长
	// content 可绕过 5000 字契约入库并广播给全群）。
	if isValidSendRequest(SendGroupMessageRequest{Type: "image", FileID: "f1", Content: longContent}) {
		t.Fatal("image with overlength content should be invalid")
	}
	if isValidSendRequest(SendGroupMessageRequest{Type: "share", ShareCode: "sc123", Content: longContent}) {
		t.Fatal("share with overlength content should be invalid")
	}
	if !isValidSendRequest(SendGroupMessageRequest{Type: "image", FileID: "f1", Content: "图片说明"}) {
		t.Fatal("image with short optional content should be valid")
	}
}
