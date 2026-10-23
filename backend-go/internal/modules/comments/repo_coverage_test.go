package comments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestMemoryListRepliesAndTopReplies(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	parent, err := repo.Create(ctx, "u1", CreateCommentRequest{MomentID: "m1", Content: "parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := repo.Create(ctx, "u2", CreateCommentRequest{MomentID: "m1", Content: "reply", ParentID: parent.CommentID}); err != nil {
			t.Fatalf("create reply: %v", err)
		}
	}

	replyPage, err := repo.ListReplies(ctx, parent.CommentID, 2, "")
	if err != nil || len(replyPage.Items) != 2 {
		t.Fatalf("list replies: %d items err=%v", len(replyPage.Items), err)
	}
	replyPage2, err := repo.ListReplies(ctx, parent.CommentID, 2, *replyPage.NextCursor)
	if err != nil || len(replyPage2.Items) != 1 {
		t.Fatalf("replies page2: %d items err=%v", len(replyPage2.Items), err)
	}

	top, err := repo.ListTopRepliesByParentIDs(ctx, []string{parent.CommentID}, 2)
	if err != nil || len(top[parent.CommentID]) != 2 {
		t.Fatalf("top replies: %#v err=%v", top, err)
	}
	empty, err := repo.ListTopRepliesByParentIDs(ctx, nil, 2)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty parents: %#v err=%v", empty, err)
	}
}

func TestMemoryGetCommentSoftDelete(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	item, err := repo.Create(ctx, "u1", CreateCommentRequest{MomentID: "m1", Content: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, found, err := repo.Get(ctx, item.CommentID)
	if err != nil || !found || got.Content != "hi" {
		t.Fatalf("get: %#v found=%v err=%v", got, found, err)
	}
	if deleted, err := repo.SoftDelete(ctx, item.CommentID); err != nil || !deleted {
		t.Fatalf("soft delete: deleted=%v err=%v", deleted, err)
	}
	got, found, err = repo.Get(ctx, item.CommentID)
	if err != nil || !found || got.Content != "" {
		t.Fatalf("deleted comment should be placeholder: %#v found=%v err=%v", got, found, err)
	}
}

// 防回归（P1-5）：并发/重复 Delete 的双重递减防线——第二次软删返回
// (false, nil)，service 据此跳过 moment.comments_count 递减。
func TestMemorySoftDeleteSecondCallReturnsFalse(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	item, err := repo.Create(ctx, "u1", CreateCommentRequest{MomentID: "m1", Content: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	deleted, err := repo.SoftDelete(ctx, item.CommentID)
	if err != nil || !deleted {
		t.Fatalf("first soft delete: deleted=%v err=%v", deleted, err)
	}
	deleted, err = repo.SoftDelete(ctx, item.CommentID)
	if err != nil || deleted {
		t.Fatalf("second soft delete must be a no-op: deleted=%v err=%v", deleted, err)
	}
}

func TestPostgresCommentRepoNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()
	if _, err := repo.SoftDelete(ctx, "c1"); err != persistence.ErrNotImplemented {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err := repo.ListTopRepliesByParentIDs(ctx, []string{"c1"}, 2); err != persistence.ErrNotImplemented {
		t.Fatalf("top replies: %v", err)
	}
	if _, err := repo.ListReplies(ctx, "c1", 2, ""); err != persistence.ErrNotImplemented {
		t.Fatalf("list replies: %v", err)
	}
}

func TestCommentsConstructorsWire(t *testing.T) {
	svc := NewService(NewInMemoryRepository(), nil)
	if svc == nil || svc.repo == nil {
		t.Fatal("NewService must wire repo")
	}
	svc2 := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil, nil, nil)
	if svc2 == nil || svc2.repo == nil {
		t.Fatal("NewServiceWithDeps must wire repo")
	}
}

func TestCommentsPersistenceRepoDelegates(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()
	if _, err := repo.SoftDelete(ctx, "c1"); err != persistence.ErrNotImplemented {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err := repo.ListTopRepliesByParentIDs(ctx, []string{"c1"}, 2); err != persistence.ErrNotImplemented {
		t.Fatalf("top replies: %v", err)
	}
}
