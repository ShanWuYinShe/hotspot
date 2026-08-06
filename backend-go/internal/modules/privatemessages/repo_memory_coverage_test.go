package privatemessages

import (
	"context"
	"testing"
	"time"
)

func TestMemoryMessageExistsAndCount(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	sent, err := repo.Send(ctx, "u1", "u2", SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if exists, owner, err := repo.MessageExists(ctx, sent.MessageID); err != nil || !exists || owner != "u1" {
		t.Fatalf("message exists: %v %q err=%v", exists, owner, err)
	}
	if exists, _, _ := repo.MessageExists(ctx, "missing"); exists {
		t.Fatal("missing message must not exist")
	}

	counts, err := repo.CountMessagesByTarget(ctx, "u1", time.Now().Add(-time.Hour))
	if err != nil || counts["u2"] != 1 {
		t.Fatalf("count by target: %#v err=%v", counts, err)
	}
	// since 在未来 → 0
	counts, err = repo.CountMessagesByTarget(ctx, "u1", time.Now().Add(time.Hour))
	if err != nil || len(counts) != 0 {
		t.Fatalf("future since: %#v err=%v", counts, err)
	}
	// 自己的消息不计入（只统计 sender）
	counts, err = repo.CountMessagesByTarget(ctx, "u2", time.Now().Add(-time.Hour))
	if err != nil || counts["u1"] != 0 {
		t.Fatalf("other direction: %#v err=%v", counts, err)
	}
}
