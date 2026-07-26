package comments

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu       sync.RWMutex
	comments map[string][]CommentItem
}

func NewStore() *Store {
	return &Store{comments: make(map[string][]CommentItem)}
}

func (s *Store) Create(userID string, req CreateCommentRequest) CommentItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item := CommentItem{
		CommentID:    fmt.Sprintf("comment_%d", now.UnixNano()),
		User:         UserInfo{UserID: userID, Nickname: userID},
		Content:      req.Content,
		ParentID:     req.ParentID,
		Mentions:     req.Mentions,
		CreatedAt:    now.Format(time.RFC3339),
		LikesCount:   0,
		RepliesCount: 0,
		MyState:      &MyCommentState{},
	}
	s.comments[req.MomentID] = append(s.comments[req.MomentID], item)
	return item
}

func (s *Store) List(momentID string) PaginatedComments {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]CommentItem(nil), s.comments[momentID]...)
	sort.Slice(items, func(i, j int) bool { return items[i].CommentID > items[j].CommentID })
	return PaginatedComments{Items: items}
}
