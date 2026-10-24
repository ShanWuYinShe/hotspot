package comments

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type InMemoryRepository struct {
	mu       sync.RWMutex
	comments map[string][]CommentItem // momentID -> comments
	byID     map[string]CommentItem   // commentID -> item
	byParent map[string][]string      // parentID -> child commentIDs
	// seq 保证同纳秒内连续创建的评论 ID 不碰撞（UnixNano 时间戳在快速
	// 循环下会重复，导致 byID 覆盖与分页缺失）。
	seq uint64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		comments: make(map[string][]CommentItem),
		byID:     make(map[string]CommentItem),
		byParent: make(map[string][]string),
	}
}

func (r *InMemoryRepository) Create(ctx context.Context, userID string, req CreateCommentRequest) (CommentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	r.seq++
	item := CommentItem{
		CommentID:    fmt.Sprintf("comment_%d_%d", now.UnixNano(), r.seq),
		MomentID:     req.MomentID,
		User:         UserInfo{UserID: userID, Nickname: userID},
		Content:      req.Content,
		ParentID:     strPtr(req.ParentID),
		Mentions:     req.Mentions,
		CreatedAt:    now.Format(time.RFC3339),
		LikesCount:   0,
		RepliesCount: 0,
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

// SoftDelete 返回是否实际命中（与 Postgres 实现同契约：已删/不存在 →
// (false, nil)，调用方据此跳过 moment 计数递减）。
func (r *InMemoryRepository) SoftDelete(_ context.Context, commentID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.byID[commentID]
	if !ok {
		return false, nil
	}
	// Already soft-deleted — no-op
	if item.IsDeleted {
		return false, nil
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
	if item.ParentID != nil {
		if parent, ok := r.byID[*item.ParentID]; ok {
			parent.RepliesCount--
			if parent.RepliesCount < 0 {
				parent.RepliesCount = 0
			}
			r.byID[*item.ParentID] = parent
		}
	}
	return true, nil
}

func (r *InMemoryRepository) List(_ context.Context, momentID string, limit int, cursor string) (PaginatedComments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursorAt, cursorID, err := decodeCommentCursor(cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	// 以 byID 为真值来源（评论被回复/软删后，moment 索引里的副本会过期）。
	items := make([]CommentItem, 0)
	for _, stored := range r.comments[momentID] {
		if stored.ParentID != nil {
			continue
		}
		if item, ok := r.byID[stored.CommentID]; ok {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].CommentID > items[j].CommentID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	if cursorID != "" {
		needle, ok := cursorAt.(time.Time)
		if !ok {
			return PaginatedComments{}, ErrInvalidCursor
		}
		filtered := make([]CommentItem, 0, len(items))
		for _, item := range items {
			at, parseErr := time.Parse(time.RFC3339Nano, item.CreatedAt)
			if parseErr != nil {
				continue
			}
			if at.Before(needle) || (at.Equal(needle) && item.CommentID < cursorID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	limit = normalizeCommentLimit(limit)
	if len(items) > limit {
		c := encodeCommentCursor(items[limit-1].CreatedAt, items[limit-1].CommentID)
		return PaginatedComments{Items: items[:limit], NextCursor: &c}, nil
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
	cursorAt, cursorID, err := decodeCommentCursor(cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	items := make([]CommentItem, 0, len(r.byParent[commentID]))
	for _, id := range r.byParent[commentID] {
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
	if cursorID != "" {
		needle, ok := cursorAt.(time.Time)
		if !ok {
			return PaginatedComments{}, ErrInvalidCursor
		}
		filtered := make([]CommentItem, 0, len(items))
		for _, item := range items {
			at, parseErr := time.Parse(time.RFC3339Nano, item.CreatedAt)
			if parseErr != nil {
				continue
			}
			if at.After(needle) || (at.Equal(needle) && item.CommentID > cursorID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	limit = normalizeCommentLimit(limit)
	if len(items) > limit {
		c := encodeCommentCursor(items[limit-1].CreatedAt, items[limit-1].CommentID)
		return PaginatedComments{Items: items[:limit], NextCursor: &c}, nil
	}
	return PaginatedComments{Items: items}, nil
}

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

// ensure InMemoryRepository compiles as Repository
var _ Repository = (*InMemoryRepository)(nil)
