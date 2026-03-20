package comments

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu       sync.RWMutex
	comments map[string][]CommentItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{comments: make(map[string][]CommentItem)}
}
func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateCommentRequest) CommentItem {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	item := CommentItem{CommentID: fmt.Sprintf("comment_%d", now.UnixNano()), User: UserInfo{UserID: userID, Nickname: userID}, Content: req.Content, ParentID: req.ParentID, Mentions: req.Mentions, CreatedAt: now.Format(time.RFC3339), LikesCount: 0, RepliesCount: 0, MyState: &MyCommentState{IsLiked: false}}
	r.comments[req.MomentID] = append(r.comments[req.MomentID], item)
	return item
}
func (r *InMemoryRepository) List(_ context.Context, momentID string) PaginatedComments {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]CommentItem(nil), r.comments[momentID]...)
	sort.Slice(items, func(i, j int) bool { return items[i].CommentID > items[j].CommentID })
	return PaginatedComments{Items: items}
}
