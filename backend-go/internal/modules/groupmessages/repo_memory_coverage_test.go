package groupmessages

import (
	"context"
	"testing"
)

func TestMemoryGroupMessageLifecycle(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	repo.SetGroupName("g1", "热群")

	name, ok, err := repo.GetGroupName(ctx, "g1")
	if err != nil || !ok || name != "热群" {
		t.Fatalf("get group name: %q ok=%v err=%v", name, ok, err)
	}
	if _, ok, _ := repo.GetGroupName(ctx, "missing"); ok {
		t.Fatal("missing group name should not be found")
	}

	sent, err := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "你好"})
	if err != nil || sent.MessageID == "" {
		t.Fatalf("send: %#v err=%v", sent, err)
	}
	sent2, err := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u2", Type: "text", Content: "回复"})
	if err != nil {
		t.Fatalf("send2: %v", err)
	}

	// History 分页（DESC 排序）
	page1, err := repo.History(ctx, "g1", &PaginationParams{Limit: 1})
	if err != nil || len(page1.Items) != 1 || page1.Items[0].MessageID != sent2.MessageID {
		t.Fatalf("history page1: %#v err=%v", page1, err)
	}
	page2, err := repo.History(ctx, "g1", &PaginationParams{Limit: 1, Cursor: page1.NextCursor})
	if err != nil || len(page2.Items) != 1 || page2.Items[0].MessageID != sent.MessageID {
		t.Fatalf("history page2: %#v err=%v", page2, err)
	}

	// 消息元数据
	if meta, found, err := repo.LookupMessageMeta(ctx, sent.MessageID); err != nil || !found || meta.SenderUserID != "u1" || meta.GroupID != "g1" {
		t.Fatalf("lookup meta: %#v found=%v err=%v", meta, found, err)
	}
	if _, found, _ := repo.LookupMessageMeta(ctx, "missing"); found {
		t.Fatal("missing meta must not be found")
	}

	// Update
	updated, err := repo.Update(ctx, UpdateGroupMessageRequest{MessageID: sent.MessageID, UserID: "u1", Content: "修改后"})
	if err != nil || updated.MessageID != sent.MessageID {
		t.Fatalf("update: %#v err=%v", updated, err)
	}
	// 非发送者更新 → 错误
	if _, err := repo.Update(ctx, UpdateGroupMessageRequest{MessageID: sent.MessageID, UserID: "u2", Content: "x"}); err == nil {
		t.Fatal("non-owner update must fail")
	}

	// Recall
	recalled, err := repo.Recall(ctx, RecallGroupMessageRequest{MessageID: sent.MessageID, UserID: "u1"})
	if err != nil || recalled.MessageID != sent.MessageID {
		t.Fatalf("recall: %#v err=%v", recalled, err)
	}
	// 再次 recall → 错误（已撤回）
	if _, err := repo.Recall(ctx, RecallGroupMessageRequest{MessageID: sent.MessageID, UserID: "u1"}); err == nil {
		t.Fatal("double recall must fail")
	}
	// 不存在的消息 recall → 错误
	if _, err := repo.Recall(ctx, RecallGroupMessageRequest{MessageID: "missing", UserID: "u1"}); err == nil {
		t.Fatal("recall missing must fail")
	}
}

func TestMemoryMarkReadAndUnreadCounts(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "m1"})
	sent2, _ := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "m2"})

	if err := repo.MarkAsRead(ctx, "u2", MarkGroupMessagesReadRequest{GroupID: "g1", UpToMessageID: sent2.MessageID}); err != nil {
		t.Fatalf("mark as read: %v", err)
	}
	if err := repo.MarkAllRead(ctx, "u2"); err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	// 未读计数为 0（已全部已读）
	if err := repo.MarkAsRead(ctx, "u2", MarkGroupMessagesReadRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("mark read empty: %v", err)
	}
}

func TestMemoryStubMethods(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	if ok, err := repo.IsMember(ctx, "g1", "u1"); err != nil || !ok {
		t.Fatalf("is member stub: %v %v", ok, err)
	}
	if _, _, _, err := repo.MemberMuteStatus(ctx, "g1", "u1"); err != nil {
		t.Fatalf("mute status: %v", err)
	}
	if _, found, _ := repo.GetMessageForReply(ctx, "m1"); found {
		t.Fatal("reply stub must not find")
	}
	if _, found, _ := repo.GetMediaForMessage(ctx, "m1"); found {
		t.Fatal("media stub must not find")
	}
}

func TestMemoryMessageExistsStub(t *testing.T) {
	repo := NewInMemoryRepository()
	if exists, groupID, err := repo.MessageExists(context.Background(), "m1"); err != nil || exists || groupID != "" {
		t.Fatalf("message exists stub: %v %q %v", exists, groupID, err)
	}
}
