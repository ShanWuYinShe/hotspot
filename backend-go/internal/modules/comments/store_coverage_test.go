package comments

import (
	"context"
	"errors"
	"testing"
)

func TestStoreCreateAndList(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore must return a store")
	}
	item := s.Create("u1", CreateCommentRequest{MomentID: "m1", Content: "hi"})
	if item.CommentID == "" || item.User.UserID != "u1" || item.Content != "hi" {
		t.Fatalf("create: %#v", item)
	}
	s.Create("u1", CreateCommentRequest{MomentID: "m1", Content: "second", ParentID: item.CommentID})
	s.Create("u2", CreateCommentRequest{MomentID: "m2", Content: "other"})

	list := s.List("m1")
	if len(list.Items) != 2 {
		t.Fatalf("list m1: %d items", len(list.Items))
	}
	empty := s.List("missing")
	if len(empty.Items) != 0 {
		t.Fatalf("missing moment: %d items", len(empty.Items))
	}
}

func TestServiceCoverageSettersAndGetters(t *testing.T) {
	svc := NewService(NewInMemoryRepository(), nil)
	ctx := context.Background()

	// SetBlockChecker 注入后生效（nil 注入不 panic）
	svc.SetBlockChecker(nil)
	svc.SetBlockChecker(func(callerID, authorID string) bool { return callerID == "blocked" })
	if svc.blockChecks == nil {
		t.Fatal("blockChecks must be wired")
	}

	// SetReactionAggregator 注入后生效
	svc.SetReactionAggregator(nil)
	if svc.reactions != nil {
		t.Fatal("reactions must be nil after nil inject")
	}

	// Get 委托 repo：存在的评论可查，缺失返回 found=false
	created, err := svc.repo.Create(ctx, "u1", CreateCommentRequest{MomentID: "m1", Content: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	item, found, err := svc.Get(ctx, created.CommentID)
	if err != nil || !found || item.Content != "hi" {
		t.Fatalf("get: %#v found=%v err=%v", item, found, err)
	}
	if _, found, err := svc.Get(ctx, "missing"); err != nil || found {
		t.Fatalf("missing get: found=%v err=%v", found, err)
	}
}

func TestServiceEnsureMomentExists(t *testing.T) {
	provider := &momentProviderStub{exists: true}
	svc := NewService(NewInMemoryRepository(), provider)
	ctx := context.Background()

	if err := svc.ensureMomentExists(ctx, "m1"); err != nil {
		t.Fatalf("existing moment: %v", err)
	}
	provider.exists = false
	if err := svc.ensureMomentExists(ctx, "m1"); !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("missing moment: %v", err)
	}
	provider.commentableErr = errors.New("boom")
	if err := svc.ensureMomentExists(ctx, "m1"); err == nil {
		t.Fatal("provider error must propagate")
	}
}

func TestServiceGetForDetailPrivacy(t *testing.T) {
	ctx := testCommentContext("viewer")
	provider := &momentProviderStub{visibleFound: true, visible: true}
	svc := NewService(NewInMemoryRepository(), provider)
	created, err := svc.repo.Create(ctx, "u1", CreateCommentRequest{MomentID: "m1", Content: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	item, found, err := svc.GetForDetail(ctx, created.CommentID)
	if err != nil || !found || item.Content != "hi" {
		t.Fatalf("visible detail: %#v found=%v err=%v", item, found, err)
	}
	provider.visible = false
	if _, found, err := svc.GetForDetail(ctx, created.CommentID); err != nil || found {
		t.Fatalf("invisible detail must be treated as missing: found=%v err=%v", found, err)
	}
}
