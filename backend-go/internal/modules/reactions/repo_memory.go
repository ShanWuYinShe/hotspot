package reactions

import (
	"context"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

// reactEntry is one stored reaction row in memory.
type reactEntry struct {
	userID    string
	nickname  string
	emoji     string
	reactedAt string
}

// targetKey is the composite (target_type, target_id) used as the top-level
// map key so we can host all three target kinds in one store.
type targetKey struct {
	kind ReactTarget
	id   string
}

// InMemoryRepository is the non-persistent Repository implementation, used by
// tests and single-instance mode without postgres.
type InMemoryRepository struct {
	mu        sync.RWMutex
	reactions map[targetKey]map[string]reactEntry // target -> userID -> entry
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{reactions: make(map[targetKey]map[string]reactEntry)}
}

func (r *InMemoryRepository) React(ctx context.Context, targetType ReactTarget, targetID, userID, emoji string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := targetKey{targetType, targetID}
	if r.reactions[key] == nil {
		r.reactions[key] = make(map[string]reactEntry)
	}
	existing, ok := r.reactions[key][userID]
	if ok && existing.emoji == emoji {
		return false, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.reactions[key][userID] = reactEntry{userID: userID, nickname: userID, emoji: emoji, reactedAt: now}
	// Enroll undo so a transaction rollback reverts the change.
	if tx, ok := persistence.MemoryTxFromContext(ctx); ok {
		var restore func()
		if ok {
			prev := existing
			restore = func() {
				r.mu.Lock()
				defer r.mu.Unlock()
				// If prev was zero-value (no prior reaction), delete; else revert.
				if prev.userID == "" {
					delete(r.reactions[key], userID)
				} else {
					r.reactions[key][userID] = prev
				}
			}
		}
		tx.RecordUndo(func(_ context.Context) {
			if restore != nil {
				restore()
			}
		})
	}
	return true, nil
}

func (r *InMemoryRepository) Unreact(ctx context.Context, targetType ReactTarget, targetID, userID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := targetKey{targetType, targetID}
	entries := r.reactions[key]
	if entries == nil {
		return false, nil
	}
	prev, ok := entries[userID]
	if !ok {
		return false, nil
	}
	delete(entries, userID)
	if tx, ok := persistence.MemoryTxFromContext(ctx); ok {
		// capture by value for the undo closure
		prev := prev
		tx.RecordUndo(func(_ context.Context) {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.reactions[key] == nil {
				r.reactions[key] = make(map[string]reactEntry)
			}
			r.reactions[key][userID] = prev
		})
	}
	return true, nil
}

func (r *InMemoryRepository) ListReactions(_ context.Context, targetType ReactTarget, targetID, emoji string, limit int, cursor string) ([]ReactionItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit = normalizeReactionLimit(limit)
	entries := r.entriesSorted(targetType, targetID)
	if emoji != "" {
		filtered := entries[:0]
		for _, e := range entries {
			if e.emoji == emoji {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
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
		return []ReactionItem{}, "", nil
	}
	end := len(entries)
	if start+limit < end {
		end = start + limit
	}
	page := entries[start:end]
	items := make([]ReactionItem, 0, len(page))
	for _, e := range page {
		items = append(items, ReactionItem{
			User:      UserInfo{UserID: e.userID, Nickname: e.nickname},
			Emoji:     e.emoji,
			ReactedAt: e.reactedAt,
		})
	}
	var nextCursor string
	if end < len(entries) {
		nextCursor = page[len(page)-1].userID
	}
	return items, nextCursor, nil
}

func (r *InMemoryRepository) Aggregate(_ context.Context, targetType ReactTarget, targetID, viewerID string) ([]Reaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := targetKey{targetType, targetID}
	entries := r.reactions[key]
	byEmoji := make(map[string]int)
	reacted := false
	for userID, e := range entries {
		byEmoji[e.emoji]++
		if userID == viewerID && viewerID != "" {
			reacted = true
		}
	}
	out := make([]Reaction, 0, len(byEmoji))
	for emoji, count := range byEmoji {
		out = append(out, Reaction{Emoji: emoji, Count: count, ReactedByMe: false})
	}
	// Sort by count desc, emoji asc for stable output.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Emoji < out[j].Emoji
	})
	if reacted {
		// Mark the viewer's emoji. Since one user has one emoji per target,
		// there is exactly one matching entry.
		viewerEmoji := entries[viewerID].emoji
		for i := range out {
			if out[i].Emoji == viewerEmoji {
				out[i].ReactedByMe = true
			}
		}
	}
	return out, nil
}

func (r *InMemoryRepository) AggregateBatch(_ context.Context, targetType ReactTarget, targetIDs []string, viewerID string) (map[string][]Reaction, error) {
	result := make(map[string][]Reaction, len(targetIDs))
	for _, id := range targetIDs {
		recs, err := r.Aggregate(context.Background(), targetType, id, viewerID)
		if err != nil {
			return nil, err
		}
		result[id] = recs
	}
	return result, nil
}

func (r *InMemoryRepository) ListReactedTargets(_ context.Context, targetType ReactTarget, userID string, limit int, cursor string) ([]string, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit = normalizeReactionLimit(limit)
	type rec struct {
		targetID  string
		reactedAt string
	}
	var all []rec
	for key, entries := range r.reactions {
		if key.kind != targetType {
			continue
		}
		if e, ok := entries[userID]; ok {
			all = append(all, rec{targetID: key.id, reactedAt: e.reactedAt})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].reactedAt == all[j].reactedAt {
			return all[i].targetID > all[j].targetID
		}
		return all[i].reactedAt > all[j].reactedAt
	})
	start := 0
	if cursor != "" {
		for i, e := range all {
			if e.targetID == cursor {
				start = i + 1
				break
			}
		}
	}
	if start >= len(all) {
		return []string{}, "", nil
	}
	end := len(all)
	if start+limit < end {
		end = start + limit
	}
	page := all[start:end]
	ids := make([]string, 0, len(page))
	for _, e := range page {
		ids = append(ids, e.targetID)
	}
	var nextCursor string
	if end < len(all) {
		nextCursor = page[len(page)-1].targetID
	}
	return ids, nextCursor, nil
}

// entriesSorted returns all reaction entries for a target, sorted newest-first
// with userID ASC tie-break (matching the postgres ordering).
func (r *InMemoryRepository) entriesSorted(targetType ReactTarget, targetID string) []reactEntry {
	src := r.reactions[targetKey{targetType, targetID}]
	out := make([]reactEntry, 0, len(src))
	for _, e := range src {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].reactedAt == out[j].reactedAt {
			return out[i].userID < out[j].userID
		}
		return out[i].reactedAt > out[j].reactedAt
	})
	return out
}

// compile-time assertion that InMemoryRepository satisfies Repository.
var _ Repository = (*InMemoryRepository)(nil)
