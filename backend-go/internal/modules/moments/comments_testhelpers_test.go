package moments

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hotspot/backend-go/internal/modules/comments"
)

// momentsCommentsRepo 是 comments.Repository 的最小测试桩：comments 内存实现
// 已迁入其模块测试构建（P0-2），moments 用例只需要「创建后能列表/软删」的
// 行为，不再跨包依赖具体内存仓储。
type momentsCommentsRepo struct {
	comments.Repository
	mu     sync.Mutex
	byID   map[string]comments.CommentItem
	byTime map[string][]comments.CommentItem
	seq    int
}

func newMomentsCommentsRepo() *momentsCommentsRepo {
	return &momentsCommentsRepo{byID: make(map[string]comments.CommentItem), byTime: make(map[string][]comments.CommentItem)}
}

func (r *momentsCommentsRepo) Create(_ context.Context, userID string, req comments.CreateCommentRequest) (comments.CommentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	now := time.Now().UTC()
	item := comments.CommentItem{
		CommentID: fmt.Sprintf("test_comment_%d_%d", now.UnixNano(), r.seq),
		MomentID:  req.MomentID,
		User:      comments.UserInfo{UserID: userID, Nickname: userID},
		Content:   req.Content,
		ParentID:  nil,
		Mentions:  req.Mentions,
		CreatedAt: now.Format(time.RFC3339),
	}
	r.byID[item.CommentID] = item
	r.byTime[req.MomentID] = append(r.byTime[req.MomentID], item)
	return item, nil
}

func (r *momentsCommentsRepo) Get(_ context.Context, commentID string) (comments.CommentItem, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.byID[commentID]
	return item, ok, nil
}

func (r *momentsCommentsRepo) SoftDelete(_ context.Context, commentID string) (bool, error) {
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

func (r *momentsCommentsRepo) List(_ context.Context, momentID string, limit int, _ string) (comments.PaginatedComments, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]comments.CommentItem, 0, len(r.byTime[momentID]))
	for _, item := range r.byTime[momentID] {
		if item.IsDeleted {
			continue
		}
		items = append(items, item)
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return comments.PaginatedComments{Items: items}, nil
}

func (r *momentsCommentsRepo) ListReplies(context.Context, string, int, string) (comments.PaginatedComments, error) {
	return comments.PaginatedComments{}, nil
}

func (r *momentsCommentsRepo) ListTopRepliesByParentIDs(context.Context, []string, int) (map[string][]comments.CommentItem, error) {
	return map[string][]comments.CommentItem{}, nil
}
