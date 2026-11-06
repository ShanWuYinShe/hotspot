package bootstrap

import (
	"context"
	"sync"
	"testing"

	"hotspot/backend-go/internal/modules/comments"
)

// commentRepoTestStub 是 comments.Repository 的最小桩（comments 内存实现已迁入
// 其模块测试构建，P0-2）：reaction provider 用例只需要创建/读取/软删语义。
type commentRepoTestStub struct {
	comments.Repository
	mu   sync.Mutex
	byID map[string]comments.CommentItem
}

func newCommentRepoTestStub() *commentRepoTestStub {
	return &commentRepoTestStub{byID: make(map[string]comments.CommentItem)}
}

func (r *commentRepoTestStub) Create(_ context.Context, userID string, req comments.CreateCommentRequest) (comments.CommentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := comments.CommentItem{CommentID: "comment_stub_1", MomentID: req.MomentID, User: comments.UserInfo{UserID: userID, Nickname: userID}, Content: req.Content}
	r.byID[item.CommentID] = item
	return item, nil
}

func (r *commentRepoTestStub) Get(_ context.Context, commentID string) (comments.CommentItem, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.byID[commentID]
	return item, ok, nil
}

func (r *commentRepoTestStub) SoftDelete(_ context.Context, commentID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.byID[commentID]
	if !ok || item.IsDeleted {
		return false, nil
	}
	item.IsDeleted = true
	item.Content = ""
	r.byID[commentID] = item
	return true, nil
}

// 审计 F-11：软删评论必须按不存在处理——此前 commentReactionProvider.Resolve
// 只查评论存在性，软删评论（content 已清空）仍可被点赞并给作者发
// reaction_added 通知。这里把 moments 置 nil 以隔离该分支，只验证 IsDeleted 判定。
func TestCommentReactionProviderRejectsSoftDeletedComment(t *testing.T) {
	repo := newCommentRepoTestStub()
	svc := comments.NewServiceWithDeps(repo, nil, nil, nil, nil, nil)
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
