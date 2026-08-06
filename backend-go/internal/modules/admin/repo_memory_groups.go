package admin

import (
	"context"

	"hotspot/backend-go/internal/platform/types"
	"sort"
	"strings"
)

func cloneAdminGroupItem(item AdminGroupItem) AdminGroupItem {
	cloned := item
	cloned.Tags = cloneStrings(item.Tags)
	if item.MyState != nil {
		state := *item.MyState
		cloned.MyState = &state
	}
	return cloned
}

func (r *InMemoryRepository) SearchGroups(_ context.Context, req AdminGroupSearchRequest) (adminGroupPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAdminGroupCursor(types.PaginationCursor(req.Pagination))
	if apiErr != nil {
		return adminGroupPage{}, apiErr
	}
	items := make([]AdminGroupItem, 0, len(r.groups))
	for _, item := range r.groups {
		if req.GroupID != "" && item.Group.GroupID != req.GroupID {
			continue
		}
		if req.Name != "" && !strings.Contains(strings.ToLower(item.Group.Name), strings.ToLower(req.Name)) {
			continue
		}
		if req.OwnerID != "" && item.Owner.UserID != req.OwnerID {
			continue
		}
		items = append(items, cloneAdminGroupItem(item))
	}
	sort.Slice(items, func(i, j int) bool {
		it, jt := timeFromRFC3339(items[i].CreatedAt), timeFromRFC3339(items[j].CreatedAt)
		if it.Equal(jt) {
			return items[i].Group.GroupID < items[j].Group.GroupID
		}
		return it.After(jt)
	})
	filtered := make([]AdminGroupItem, 0, len(items))
	for _, item := range items {
		if types.PaginationCursor(req.Pagination) != "" {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursor.CreatedAt) {
				continue
			}
			if createdAt.Equal(cursor.CreatedAt) && item.Group.GroupID <= cursor.GroupID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	paged, hasMore := limitPageItems(filtered, limit)
	page := adminGroupPage{Items: append(make([]AdminGroupItem, 0, len(paged)), paged...)}
	if hasMore {
		last := paged[len(paged)-1]
		page.Next = &adminGroupCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), GroupID: last.Group.GroupID}
	}
	return page, nil
}

func (r *InMemoryRepository) GetGroup(_ context.Context, groupID string) (AdminGroupItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.groups[groupID]
	if !ok {
		return AdminGroupItem{}, false, nil
	}
	return cloneAdminGroupItem(item), true, nil
}

func (r *InMemoryRepository) DismissGroup(_ context.Context, groupID string) (AdminGroupItem, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.groups[groupID]
	if !ok {
		return AdminGroupItem{}, false, nil
	}
	delete(r.groups, groupID)
	delete(r.groupMembers, groupID)
	return cloneAdminGroupItem(item), true, nil
}

func (r *InMemoryRepository) ListGroupMemberIDs(_ context.Context, groupID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneStrings(r.groupMembers[groupID]), nil
}
