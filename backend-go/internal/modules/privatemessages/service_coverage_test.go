package privatemessages

import (
	"context"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func TestMemoryReplyMediaAndParticipants(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// GetMessageForReply / GetMediaForMessage 内存实现 stub
	if _, found, err := repo.GetMessageForReply(ctx, "m1", "u1"); err != nil || found {
		t.Fatalf("reply stub: found=%v err=%v", found, err)
	}
	if _, found, err := repo.GetMediaForMessage(ctx, "f1", "u1"); err != nil || found {
		t.Fatalf("media stub: found=%v err=%v", found, err)
	}

	// MessageParticipants：存在与缺失
	sent, err := repo.Send(ctx, "u1", "u2", SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	sender, receiver, found, err := repo.MessageParticipants(ctx, sent.MessageID)
	if err != nil || !found || sender != "u1" || receiver != "u2" {
		t.Fatalf("participants: %q %q found=%v err=%v", sender, receiver, found, err)
	}
	// 反向视角（u2 发送）也应解析正确
	sent2, err := repo.Send(ctx, "u2", "u1", SendPrivateMessageRequest{TargetUserID: "u1", Type: "text", Content: "back"})
	if err != nil {
		t.Fatalf("send2: %v", err)
	}
	sender, receiver, found, err = repo.MessageParticipants(ctx, sent2.MessageID)
	if err != nil || !found || sender != "u2" || receiver != "u1" {
		t.Fatalf("reverse participants: %q %q found=%v err=%v", sender, receiver, found, err)
	}
	if _, _, found, err := repo.MessageParticipants(ctx, "missing"); err != nil || found {
		t.Fatalf("missing participants: found=%v err=%v", found, err)
	}
}

func TestServiceConstructorsAndLookupHelpers(t *testing.T) {
	svc := NewService()
	if svc == nil || svc.repo == nil {
		t.Fatal("NewService must wire repo")
	}
	repo := NewInMemoryRepository()
	svc2 := NewServiceWithRepo(repo)
	if svc2 == nil || svc2.repo == nil {
		t.Fatal("NewServiceWithRepo must wire repo")
	}
	// SetShareValidator / SetAppConfig nil 注入不 panic
	svc2.SetShareValidator(nil)
	svc2.SetAppConfig(nil)

	// NewRuntime / NewRuntimeWithRepo
	runtime := NewRuntimeWithRepo(repo)
	if runtime.Handler() == nil || runtime.ReadService() == nil {
		t.Fatal("NewRuntimeWithRepo must wire handler and read service")
	}
	runtime.SetShareValidator(nil)
	_ = runtime

	// LookupMessage 转发 MessageExists
	ctx := pmAuthCtx("u1")
	sent, _ := repo.Send(ctx, "u1", "u2", SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if found, senderID, err := svc2.LookupMessage(ctx, sent.MessageID); err != nil || !found || senderID != "u1" {
		t.Fatalf("lookup: found=%v sender=%q err=%v", found, senderID, err)
	}

	// IsReactionTargetVisible：会话参与者可见，第三方不可见
	if found, visible, err := svc2.IsReactionTargetVisible(ctx, sent.MessageID); err != nil || !found || !visible {
		t.Fatalf("participant visible: found=%v visible=%v err=%v", found, visible, err)
	}
	if found, visible, err := svc2.IsReactionTargetVisible(pmAuthCtx("u2"), sent.MessageID); err != nil || !found || !visible {
		t.Fatalf("receiver visible: found=%v visible=%v err=%v", found, visible, err)
	}
	if found, visible, err := svc2.IsReactionTargetVisible(pmAuthCtx("u3"), sent.MessageID); err != nil || !found || visible {
		t.Fatalf("outsider visible: found=%v visible=%v err=%v", found, visible, err)
	}
	if found, _, err := svc2.IsReactionTargetVisible(ctx, "missing"); err != nil || found {
		t.Fatalf("missing visible: found=%v err=%v", found, err)
	}
}

func pmAuthCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: "sess_" + userID})
}
