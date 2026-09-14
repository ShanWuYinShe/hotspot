package users

import (
	"context"
	"testing"
)

func TestMemoryRelationshipsAndBlocking(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	if _, err := repo.Follow(ctx, "u1", "u2", "search", "m1"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if _, err := repo.Follow(ctx, "u2", "u1", "", ""); err != nil {
		t.Fatalf("mutual follow: %v", err)
	}
	if err := repo.Block(ctx, "u1", "u3", "spam"); err != nil {
		t.Fatalf("block: %v", err)
	}
	if err := repo.Block(ctx, "u1", "u4", "harassment"); err != nil {
		t.Fatalf("block u4: %v", err)
	}
	if err := repo.SetRemark(ctx, "u1", "u2", "老朋友"); err != nil {
		t.Fatalf("set remark: %v", err)
	}

	// Relationship / RelationshipBatch
	rel, err := repo.Relationship(ctx, "u1", "u2")
	if err != nil || !rel.IsFollowing || !rel.IsFollowedBy {
		t.Fatalf("relationship: %#v err=%v", rel, err)
	}
	batch, err := repo.RelationshipBatch(ctx, "u1", []string{"u2", "u3", "missing"})
	if err != nil || len(batch) != 3 {
		t.Fatalf("batch: %#v err=%v", batch, err)
	}
	if batch["u2"].IsBlocked || !batch["u2"].IsFollowing || !batch["u3"].IsBlocked {
		t.Fatalf("unexpected relation flags: %#v", batch)
	}
	if batch["missing"].IsFollowing || batch["missing"].IsBlocked {
		t.Fatalf("missing user should be empty relation: %#v", batch["missing"])
	}

	// ListBlocked 游标
	page1, err := repo.ListBlocked(ctx, "u1", 1, "")
	if err != nil || len(page1.Items) != 1 {
		t.Fatalf("blocked page1: %d items err=%v", len(page1.Items), err)
	}
	page2, err := repo.ListBlocked(ctx, "u1", 1, *page1.NextCursor)
	if err != nil || len(page2.Items) != 1 || page2.NextCursor != nil {
		t.Fatalf("blocked page2: %d items next=%v err=%v", len(page2.Items), page2.NextCursor, err)
	}

	// Unblock/Unfollow
	if err := repo.Unblock(ctx, "u1", "u3"); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	if err := repo.Unfollow(ctx, "u1", "u2"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	rel, _ = repo.Relationship(ctx, "u1", "u2")
	if rel.IsFollowing {
		t.Fatalf("unfollow should clear own direction: %#v", rel)
	}
	if !rel.IsFollowedBy {
		t.Fatalf("unfollow must not clear the other direction: %#v", rel)
	}
}

func TestMemoryBlockedEitherWay(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	if err := repo.Block(ctx, "u1", "u2", "spam"); err != nil {
		t.Fatalf("block: %v", err)
	}
	if err := repo.Block(ctx, "u3", "u1", "harassment"); err != nil {
		t.Fatalf("block reverse: %v", err)
	}

	// u1 拉黑 u2：caller 侧为真，target 侧为假
	callerBlocked, targetBlocked, err := repo.BlockedEitherWay(ctx, "u1", "u2")
	if err != nil || !callerBlocked || targetBlocked {
		t.Fatalf("u1->u2: caller=%v target=%v err=%v", callerBlocked, targetBlocked, err)
	}
	// u3 拉黑 u1：反向为真
	callerBlocked, targetBlocked, err = repo.BlockedEitherWay(ctx, "u1", "u3")
	if err != nil || callerBlocked || !targetBlocked {
		t.Fatalf("u1->u3: caller=%v target=%v err=%v", callerBlocked, targetBlocked, err)
	}
	// 双向拉黑
	if err := repo.Block(ctx, "u1", "u3", "again"); err != nil {
		t.Fatalf("block both ways: %v", err)
	}
	callerBlocked, targetBlocked, err = repo.BlockedEitherWay(ctx, "u1", "u3")
	if err != nil || !callerBlocked || !targetBlocked {
		t.Fatalf("u1<->u3: caller=%v target=%v err=%v", callerBlocked, targetBlocked, err)
	}
	// 无任何关系的用户对返回全假
	callerBlocked, targetBlocked, err = repo.BlockedEitherWay(ctx, "u9", "u8")
	if err != nil || callerBlocked || targetBlocked {
		t.Fatalf("no relation: caller=%v target=%v err=%v", callerBlocked, targetBlocked, err)
	}

	// Service 层透传
	svc := NewServiceWithRepo(repo)
	if _, _, err := svc.BlockStateBetween(ctx, "u1", "u2"); err != nil {
		t.Fatalf("service BlockStateBetween: %v", err)
	}
}

func TestMemoryDeleteProfileClearsRelationships(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	_, _ = repo.Follow(ctx, "u1", "u2", "", "")
	_, _ = repo.Follow(ctx, "u2", "u1", "", "")

	if err := repo.DeleteProfile(ctx, "u1"); err != nil {
		t.Fatalf("delete profile: %v", err)
	}
	rel, err := repo.Relationship(ctx, "u2", "u1")
	if err != nil || rel.IsFollowing || rel.IsBlocked {
		t.Fatalf("deleted user relation should be cleared: %#v err=%v", rel, err)
	}
}
