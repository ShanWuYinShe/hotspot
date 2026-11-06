package groupmessages

import (
	"context"
	"testing"
)

// TestRecallMessageByIDAsAdmin：举报裁决按 message_id 撤回群消息（无作者/时间窗
// 限制）；不存在或已撤回返回 (false, nil) 供裁决侧 fallback 私聊表。
func TestRecallMessageByIDAsAdmin(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	ctx := context.Background()

	sent, err := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "user_a", Type: "text", Content: "bad"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	hit, err := service.RecallMessageByIDAsAdmin(ctx, "admin_1", sent.MessageID)
	if err != nil || !hit {
		t.Fatalf("recall as admin = %v err=%v, want hit", hit, err)
	}
	hit, err = service.RecallMessageByIDAsAdmin(ctx, "admin_1", sent.MessageID)
	if err != nil || hit {
		t.Fatalf("repeat recall = %v err=%v, want false/nil (idempotent)", hit, err)
	}
	hit, err = service.RecallMessageByIDAsAdmin(ctx, "admin_1", "message_missing")
	if err != nil || hit {
		t.Fatalf("missing message = %v err=%v, want false/nil", hit, err)
	}
}
