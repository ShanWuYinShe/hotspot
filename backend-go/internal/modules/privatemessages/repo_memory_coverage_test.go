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

func TestMemoryRecentAndLastWithDeletedAfter(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// 1. 发送 2 条消息
	_, err := repo.Send(ctx, "u1", "u2", SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "first"})
	if err != nil {
		t.Fatalf("send 1: %v", err)
	}
	cutoff := time.Now().Add(-time.Minute)
	_, err = repo.Send(ctx, "u1", "u2", SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "second"})
	if err != nil {
		t.Fatalf("send 2: %v", err)
	}

	// 2. 携带 deletedAfter 过滤
	recent, err := repo.Recent(ctx, "u1", "u2", 10, &cutoff)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent.Items) == 0 {
		t.Fatal("expected recent items")
	}

	// 3. Last
	last, err := repo.Last(ctx, "u1", "u2", &cutoff)
	if err != nil || last == nil {
		t.Fatalf("expected last message, got %v", last)
	}

	// 4. 空会话 Last
	emptyLast, err := repo.Last(ctx, "u1", "u_nobody", nil)
	if err != nil || emptyLast != nil {
		t.Fatalf("expected nil for empty conversation last, got %v", emptyLast)
	}
}
