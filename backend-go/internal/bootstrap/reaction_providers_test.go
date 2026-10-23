package bootstrap

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/modules/comments"
)

// 审计 F-11：软删评论必须按不存在处理——此前 commentReactionProvider.Resolve
// 只查评论存在性，软删评论（content 已清空）仍可被点赞并给作者发
// reaction_added 通知。这里把 moments 置 nil 以隔离该分支，只验证 IsDeleted 判定。
func TestCommentReactionProviderRejectsSoftDeletedComment(t *testing.T) {
	repo := comments.NewInMemoryRepository()
	svc := comments.NewService(repo, nil)
	provider := commentReactionProvider{comments: svc, moments: nil}

	ctx := context.Background()
	created, err := repo.Create(ctx, "user_alice", comments.CreateCommentRequest{
		MomentID: "moment_1",
		Content:  "会被软删的评论",
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}

	// 软删前：可反应，owner 为评论作者
	got, ok, err := provider.Resolve(ctx, created.CommentID)
	if err != nil || !ok {
		t.Fatalf("live comment must resolve: ok=%v err=%v", ok, err)
	}
	if got.OwnerUserID != "user_alice" {
		t.Fatalf("owner = %q, want user_alice", got.OwnerUserID)
	}

	// 软删后：按不存在处理（SoftDelete 现返回是否命中，防并发双递减）
	if _, err := repo.SoftDelete(ctx, created.CommentID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if _, ok, err := provider.Resolve(ctx, created.CommentID); err != nil || ok {
		t.Fatalf("soft-deleted comment must not resolve: ok=%v err=%v", ok, err)
	}
}
