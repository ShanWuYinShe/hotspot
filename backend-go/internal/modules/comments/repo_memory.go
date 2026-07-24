package comments

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type likeEntry struct {
	userID   string
	nickname string
	likedAt  string
}

type InMemoryRepository struct {
	mu       sync.RWMutex
	comments map[string][]CommentItem        // momentID -> comments
	byID     map[string]CommentItem          // commentID -> item
	byParent map[string][]string             // parentID -> child commentIDs
	likes    map[string]map[string]likeEntry // commentID -> userID -> entry
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		comments: make(map[string][]CommentItem),
		byID:     make(map[string]CommentItem),
		byParent: make(map[string][]string),
		likes:    make(map[string]map[string]likeEntry),
	}
}

func (r *InMemoryRepository) Create(ctx context.Context, userID string, req CreateCommentRequest) (CommentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	item := CommentItem{
		CommentID:    fmt.Sprintf("comment_%d", now.UnixNano()),
		MomentID:     req.MomentID,
		User:         UserInfo{UserID: userID, Nickname: userID},
		Content:      req.Content,
		ParentID:     req.ParentID,
		Mentions:     req.Mentions,
		CreatedAt:    now.Format(time.RFC3339),
		LikesCount:   0,
		RepliesCount: 0,
		MyState:      &MyCommentState{IsLiked: false},
	}
	r.comments[req.MomentID] = append(r.comments[req.MomentID], item)
	r.byID[item.CommentID] = item
	hadParentEntry := false
	var parentRepliesCountBefore int
	if req.ParentID != "" {
		r.byParent[req.ParentID] = append(r.byParent[req.ParentID], item.CommentID)
		if parent, ok := r.byID[req.ParentID]; ok {
			hadParentEntry = true
			parentRepliesCountBefore = parent.RepliesCount
			parent.RepliesCount++
			r.byID[req.ParentID] = parent
		}
	}
	// Enroll an undo so a transaction rollback fully removes the comment
	// (rather than leaving a soft-deleted stub that List would still surface).
	if tx, ok := persistence.MemoryTxFromContext(ctx); ok {
		commentID := item.CommentID
		momentID := req.MomentID
		parentID := req.ParentID
		tx.RecordUndo(func(_ context.Context) {
			r.mu.Lock()
			defer r.mu.Unlock()
			delete(r.byID, commentID)
			if items, ok := r.comments[momentID]; ok {
				for i, it := range items {
					if it.CommentID == commentID {
						r.comments[momentID] = append(items[:i], items[i+1:]...)
						break
					}
				}
			}
			if parentID != "" {
				if ids, ok := r.byParent[parentID]; ok {
					for i, id := range ids {
						if id == commentID {
							r.byParent[parentID] = append(ids[:i], ids[i+1:]...)
							break
						}
					}
				}
				if hadParentEntry {
					if parent, ok := r.byID[parentID]; ok {
						parent.RepliesCount = parentRepliesCountBefore
						r.byID[parentID] = parent
					}
				}
			}
		})
	}
	return item, nil
}

func (r *InMemoryRepository) SoftDelete(_ context.Context, commentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.byID[commentID]
	if !ok {
		return nil
	}
	// Already soft-deleted — no-op
	if item.IsDeleted {
		return nil
	}
	// Soft delete: mark as deleted, clear content
	item.IsDeleted = true
	item.Content = ""
	r.byID[commentID] = item
	// Update the moment-indexed copy too
	for momentID, items := range r.comments {
		for idx, it := range items {
			if it.CommentID == commentID {
				r.comments[momentID][idx] = item
				break
			}
		}
	}
	// Decrement parent comment's RepliesCount if this is a reply
	if item.ParentID != "" {
		if parent, ok := r.byID[item.ParentID]; ok {
			parent.RepliesCount--
			if parent.RepliesCount < 0 {
				parent.RepliesCount = 0
			}
			r.byID[item.ParentID] = parent
		}
	}
	return nil
}

func (r *InMemoryRepository) List(_ context.Context, momentID string, limit int, cursor string) (PaginatedComments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []CommentItem
	for _, item := range r.comments[momentID] {
		if item.ParentID == "" {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].CommentID > items[j].CommentID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	if cursor != "" {
		for i, item := range items {
			if item.CommentID == cursor {
				items = items[i+1:]
				break
			}
		}
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if len(items) > limit {
		return PaginatedComments{Items: items[:limit], NextCursor: items[limit-1].CommentID}, nil
	}
	return PaginatedComments{Items: items}, nil
}

func (r *InMemoryRepository) Get(_ context.Context, commentID string) (CommentItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.byID[commentID]
	return item, ok, nil
}

func (r *InMemoryRepository) ListReplies(_ context.Context, commentID string, limit int, cursor string) (PaginatedComments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	childIDs := r.byParent[commentID]
	items := make([]CommentItem, 0, len(childIDs))
	for _, id := range childIDs {
		if item, ok := r.byID[id]; ok {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].CommentID < items[j].CommentID
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	start := 0
	if cursor != "" {
		for i, item := range items {
			if item.CommentID == cursor {
				start = i + 1
				break
			}
		}
	}
	if start >= len(items) {
		return PaginatedComments{Items: []CommentItem{}}, nil
	}
	end := len(items)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	page := items[start:end]
	var nextCursor string
	if end < len(items) {
		nextCursor = page[len(page)-1].CommentID
	}
	return PaginatedComments{Items: page, NextCursor: nextCursor}, nil
}

// ListTopRepliesByParentIDs returns up to `limit` newest replies per parent in
// one pass (single lock acquisition), mirroring the Postgres batch query so the
// service can avoid an N+1 of ListReplies calls.
func (r *InMemoryRepository) ListTopRepliesByParentIDs(_ context.Context, parentIDs []string, limit int) (map[string][]CommentItem, error) {
	result := make(map[string][]CommentItem, len(parentIDs))
	if limit <= 0 {
		return result, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, parentID := range parentIDs {
		childIDs := r.byParent[parentID]
		items := make([]CommentItem, 0, len(childIDs))
		for _, id := range childIDs {
			if item, ok := r.byID[id]; ok {
				items = append(items, item)
			}
		}
		// newest first (matches ListReplies ordering inverted to ListReplies'
		// ASC — here we want DESC for "top replies"), then take `limit`.
		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].CommentID > items[j].CommentID
			}
			return items[i].CreatedAt > items[j].CreatedAt
		})
		if len(items) > limit {
			items = items[:limit]
		}
		result[parentID] = items
	}
	return result, nil
}

func (r *InMemoryRepository) Like(_ context.Context, commentID string, userID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[commentID]; !ok {
		return false, ErrCommentNotFound
	}
	if r.likes[commentID] == nil {
		r.likes[commentID] = make(map[string]likeEntry)
	}
	if _, exists := r.likes[commentID][userID]; exists {
		return false, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.likes[commentID][userID] = likeEntry{userID: userID, nickname: userID, likedAt: now}
	if item, ok := r.byID[commentID]; ok {
		item.LikesCount++
		r.byID[commentID] = item
	}
	return true, nil
}

func (r *InMemoryRepository) Unlike(_ context.Context, commentID string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[commentID]; !ok {
		return ErrCommentNotFound
	}
	if r.likes[commentID] == nil {
		return nil
	}
	if _, exists := r.likes[commentID][userID]; !exists {
		return nil
	}
	delete(r.likes[commentID], userID)
	if item, ok := r.byID[commentID]; ok {
		item.LikesCount--
		if item.LikesCount < 0 {
			item.LikesCount = 0
		}
		r.byID[commentID] = item
	}
	return nil
}

func (r *InMemoryRepository) ListLikes(_ context.Context, commentID string, limit int, cursor string) ([]CommentLikeItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	likeMap := r.likes[commentID]
	entries := make([]likeEntry, 0, len(likeMap))
	for _, entry := range likeMap {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].likedAt == entries[j].likedAt {
			return entries[i].userID < entries[j].userID
		}
		return entries[i].likedAt > entries[j].likedAt
	})
	start := 0
	if cursor != "" {
		for i, e := range entries {
			if e.userID == cursor {
				start = i + 1
				break
			}
		}
	}
	if start >= len(entries) {
		return []CommentLikeItem{}, "", nil
	}
	end := len(entries)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	page := entries[start:end]
	items := make([]CommentLikeItem, 0, len(page))
	for _, e := range page {
		items = append(items, CommentLikeItem{
			User:    UserInfo{UserID: e.userID, Nickname: e.nickname},
			LikedAt: e.likedAt,
		})
	}
	var nextCursor string
	if end < len(entries) {
		nextCursor = page[len(page)-1].userID
	}
	return items, nextCursor, nil
}

// ensure InMemoryRepository compiles as Repository
var _ Repository = (*InMemoryRepository)(nil)
