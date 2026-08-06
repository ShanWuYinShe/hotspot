package moments

import (
	"context"
	"testing"
)

func TestMemoryMomentCountersAndFavorites(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// 计数器
	if err := repo.IncrementCommentCount(ctx, created.MomentID); err != nil {
		t.Fatalf("increment comments: %v", err)
	}
	if err := repo.DecrementCommentCount(ctx, created.MomentID); err != nil {
		t.Fatalf("decrement comments: %v", err)
	}
	// 减到负数钳制为 0
	if err := repo.DecrementCommentCount(ctx, created.MomentID); err != nil {
		t.Fatalf("decrement below zero: %v", err)
	}
	if err := repo.IncrementSharesCount(ctx, created.MomentID); err != nil {
		t.Fatalf("increment shares: %v", err)
	}
	// 不存在的 moment 计数器 no-op
	if err := repo.DecrementCommentCount(ctx, "missing"); err != nil {
		t.Fatalf("decrement missing: %v", err)
	}
	if err := repo.IncrementSharesCount(ctx, "missing"); err != ErrMomentNotFound {
		t.Fatalf("increment shares missing: expected not found, got %v", err)
	}

	// NotificationContext
	item, found, err := repo.NotificationContext(ctx, created.MomentID)
	if err != nil || !found || item.MomentID != created.MomentID {
		t.Fatalf("notification context: %#v found=%v err=%v", item, found, err)
	}

	// 收藏/取消收藏
	if err := repo.Favorite(ctx, "u2", created.MomentID); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	if err := repo.Favorite(ctx, "u2", "missing"); err != ErrMomentNotFound {
		t.Fatalf("favorite missing: expected not found, got %v", err)
	}
	if err := repo.Unfavorite(ctx, "u2", created.MomentID); err != nil {
		t.Fatalf("unfavorite: %v", err)
	}

	// MyState
	if state := repo.LoadMyState(ctx, created.MomentID, "u2"); state == nil {
		t.Fatal("my state should exist for logged-in viewer")
	}
	if state := repo.LoadMyState(ctx, created.MomentID, ""); state != nil {
		t.Fatal("anonymous viewer must have no my_state")
	}
	if state := repo.LoadMyState(ctx, "missing", "u2"); state == nil || state.IsFavorited {
		t.Fatalf("missing moment my_state: %#v", state)
	}
}

func TestMemoryMomentDeleteAndExists(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ok, err := repo.Exists(ctx, created.MomentID); err != nil || !ok {
		t.Fatalf("exists: %v err=%v", ok, err)
	}
	if err := repo.Delete(ctx, created.MomentID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(ctx, created.MomentID); err != ErrMomentNotFound {
		t.Fatalf("double delete: expected not found, got %v", err)
	}
	if ok, _ := repo.Exists(ctx, created.MomentID); ok {
		t.Fatal("deleted moment must not exist")
	}
}
