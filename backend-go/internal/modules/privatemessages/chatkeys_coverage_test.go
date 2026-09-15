package privatemessages

import (
	"context"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func TestTargetOfChatKeyRules(t *testing.T) {
	// chat_key = 较小 id + ":" + 较大 id；viewer 在 key 中则取对方
	if target, ok := targetOfChatKey("usr_a:usr_b", "usr_a"); !ok || target != "usr_b" {
		t.Fatalf("expected usr_b, got %q ok=%v", target, ok)
	}
	if target, ok := targetOfChatKey("usr_a:usr_b", "usr_b"); !ok || target != "usr_a" {
		t.Fatalf("expected usr_a, got %q ok=%v", target, ok)
	}
	// key 不含 viewer 或无法解析出对方 → false
	if _, ok := targetOfChatKey("usr_a:usr_a", "usr_a"); ok {
		t.Fatal("key only containing viewer should not resolve")
	}
	if _, ok := targetOfChatKey(":usr_b", "usr_b"); ok {
		t.Fatal("malformed key whose only non-empty segment is the viewer should not resolve")
	}
	if target, ok := targetOfChatKey(":usr_b", "usr_a"); !ok || target != "usr_b" {
		t.Fatalf("empty leading segment should resolve peer, got %q ok=%v", target, ok)
	}
	if _, ok := targetOfChatKey("usr_a:", "usr_a"); ok {
		t.Fatal("key with only viewer should not resolve")
	}
}

func TestMemoryLastByChatKeysReturnsLatestPerKey(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	if _, err := repo.Send(ctx, "usr_a", "usr_b", SendPrivateMessageRequest{TargetUserID: "usr_b", Type: "text", Content: "first"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if _, err := repo.Send(ctx, "usr_b", "usr_a", SendPrivateMessageRequest{TargetUserID: "usr_a", Type: "text", Content: "latest"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	key := conversationKey("usr_a", "usr_b")
	// viewer=usr_a：两条消息里最新一条是 usr_b 的回复
	got, err := repo.LastByChatKeys(ctx, "usr_a", []string{key}, nil)
	if err != nil {
		t.Fatalf("last by chat keys: %v", err)
	}
	item, ok := got[key]
	if !ok {
		t.Fatalf("expected item for key %s, got %#v", key, got)
	}
	if item.Content == nil || *item.Content != "latest" {
		t.Fatalf("expected latest message content, got %v", item.Content)
	}

	// 边界晚于全部消息 → 该 key 无可见消息，结果中不出现
	future := time.Now().Add(time.Hour)
	empty, err := repo.LastByChatKeys(ctx, "usr_a", []string{key}, map[string]*time.Time{key: &future})
	if err != nil {
		t.Fatalf("last with boundary: %v", err)
	}
	if _, ok := empty[key]; ok {
		t.Fatalf("expected no item beyond boundary, got %#v", empty)
	}

	// 无法解析对方的 key 被跳过
	skip, err := repo.LastByChatKeys(ctx, "usr_a", []string{"usr_a:usr_a"}, nil)
	if err != nil || len(skip) != 0 {
		t.Fatalf("unresolvable key should be skipped, got %#v err=%v", skip, err)
	}
}

func TestMemoryUnreadCountsByChatKeys(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	if _, err := repo.Send(ctx, "usr_a", "usr_b", SendPrivateMessageRequest{TargetUserID: "usr_b", Type: "text", Content: "one"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	sent, err := repo.Send(ctx, "usr_a", "usr_b", SendPrivateMessageRequest{TargetUserID: "usr_b", Type: "text", Content: "two"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	key := conversationKey("usr_a", "usr_b")
	counts, err := repo.UnreadCountsByChatKeys(ctx, "usr_b", []string{key, "usr_a:usr_a"}, nil)
	if err != nil {
		t.Fatalf("unread counts: %v", err)
	}
	if counts[key] != 2 {
		t.Fatalf("expected 2 unread for viewer usr_b, got %#v", counts)
	}

	// 标记已读后计数清零；无法解析的 key 不出现在结果中
	if _, err := repo.MarkAsRead(ctx, "usr_b", MarkPrivateMessagesReadRequest{TargetUserID: "usr_a", UpToMessageID: sent.MessageID}); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	counts, err = repo.UnreadCountsByChatKeys(ctx, "usr_b", []string{key, "usr_a:usr_a"}, nil)
	if err != nil {
		t.Fatalf("unread counts after read: %v", err)
	}
	if v, ok := counts[key]; ok && v != 0 {
		t.Fatalf("expected zero unread after mark read, got %#v", counts)
	}
	if _, ok := counts["usr_a:usr_a"]; ok {
		t.Fatalf("unresolvable key should be absent, got %#v", counts)
	}
}

func TestServiceLastAndUnreadByChatKeys(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	ctx := context.Background()
	if _, err := repo.Send(ctx, "usr_a", "usr_b", SendPrivateMessageRequest{TargetUserID: "usr_b", Type: "text", Content: "hi"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	viewerCtx := platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: "usr_b", SessionID: "sess_1"})
	key := conversationKey("usr_a", "usr_b")
	last, err := service.LastByChatKeys(viewerCtx, []string{key}, nil)
	if err != nil {
		t.Fatalf("service last: %v", err)
	}
	if item, ok := last[key]; !ok || item.Content == nil || *item.Content != "hi" {
		t.Fatalf("unexpected last result: %#v", last)
	}

	unread, err := service.UnreadCountsByChatKeys(viewerCtx, []string{key}, nil)
	if err != nil {
		t.Fatalf("service unread: %v", err)
	}
	if unread[key] != 1 {
		t.Fatalf("expected 1 unread, got %#v", unread)
	}
}
