package admin

import (
	"context"
	"time"

	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/types"
	"sort"
	"strings"

	"hotspot/backend-go/internal/platform/httpx"
)

func cloneAdminUserItem(item AdminUserItem) AdminUserItem {
	cloned := item
	cloned.Tags = cloneStrings(item.Tags)
	if item.Location != nil {
		loc := *item.Location
		cloned.Location = &loc
	}
	if item.Stats != nil {
		stats := *item.Stats
		cloned.Stats = &stats
	}
	return cloned
}

func (r *InMemoryRepository) SearchUsers(_ context.Context, req AdminUserSearchRequest) (adminUserPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAdminUserCursor(types.PaginationCursor(req.Pagination))
	if apiErr != nil {
		return adminUserPage{}, apiErr
	}
	items := make([]AdminUserItem, 0, len(r.adminUsers))
	for _, item := range r.adminUsers {
		if req.UserID != "" && item.User.UserID != req.UserID {
			continue
		}
		if req.Nickname != "" && !strings.Contains(strings.ToLower(item.User.Nickname), strings.ToLower(req.Nickname)) {
			continue
		}
		if req.LoginID != "" && !strings.Contains(strings.ToLower(item.LoginID), strings.ToLower(req.LoginID)) {
			continue
		}
		if req.Status != "" && item.Status != req.Status {
			continue
		}
		if req.UserType != "" && item.User.UserID != "admin" && req.UserType != usersmodule.UserTypeNormal {
			continue
		}
		items = append(items, cloneAdminUserItem(item))
	}
	sort.Slice(items, func(i, j int) bool {
		it, jt := timeFromRFC3339(items[i].CreatedAt), timeFromRFC3339(items[j].CreatedAt)
		if it.Equal(jt) {
			return items[i].User.UserID < items[j].User.UserID
		}
		return it.After(jt)
	})
	filtered := make([]AdminUserItem, 0, len(items))
	for _, item := range items {
		if types.PaginationCursor(req.Pagination) != "" {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursor.CreatedAt) {
				continue
			}
			if createdAt.Equal(cursor.CreatedAt) && item.User.UserID <= cursor.UserID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	paged, hasMore := limitPageItems(filtered, limit)
	page := adminUserPage{Items: append(make([]AdminUserItem, 0, len(paged)), paged...)}
	if hasMore {
		last := paged[len(paged)-1]
		page.Next = &adminUserCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), UserID: last.User.UserID}
	}
	return page, nil
}

func (r *InMemoryRepository) GetUser(_ context.Context, userID string) (AdminUserItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.adminUsers[userID]
	if !ok {
		return AdminUserItem{}, false, nil
	}
	return cloneAdminUserItem(item), true, nil
}

func (r *InMemoryRepository) GetUserBanState(_ context.Context, userID string) (UserBanState, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.adminUsers[userID]; !ok {
		return UserBanState{}, false, nil
	}
	state, ok := r.banStates[userID]
	if !ok {
		return UserBanState{Status: UserAccountStatusNormal}, true, nil
	}
	return state, true, nil
}

func (r *InMemoryRepository) SetUserBanState(_ context.Context, userID string, state UserBanState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.adminUsers[userID]
	if !ok {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}
	}
	if state.Status == "" {
		state.Status = UserAccountStatusNormal
	}
	if state.Status != UserAccountStatusBanned {
		state = UserBanState{Status: UserAccountStatusNormal}
	}
	// banned_until（spec api_spec_admin.md AdminUserItem）：封禁到期时间，
	// null = 未封禁/永久封禁（duration 0）。
	if state.Status == UserAccountStatusBanned && state.DurationDays > 0 {
		bannedAt := state.BannedAt
		if bannedAt.IsZero() {
			bannedAt = time.Now().UTC()
		}
		until := bannedAt.Add(time.Duration(state.DurationDays) * 24 * time.Hour).UTC().Format(time.RFC3339)
		item.BannedUntil = &until
	} else {
		item.BannedUntil = nil
	}
	r.banStates[userID] = state
	item.Status = state.Status
	r.adminUsers[userID] = item
	return nil
}

func (r *InMemoryRepository) ListUserSessions(_ context.Context, userID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneStrings(r.userSessions[userID]), nil
}

func (r *InMemoryRepository) RevokeSession(_ context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for userID, sessions := range r.userSessions {
		filtered := make([]string, 0, len(sessions))
		for _, candidate := range sessions {
			if candidate != sessionID {
				filtered = append(filtered, candidate)
			}
		}
		if len(filtered) == 0 {
			delete(r.userSessions, userID)
			continue
		}
		r.userSessions[userID] = filtered
	}
	return nil
}

func (r *InMemoryRepository) IsBanned(_ context.Context, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.banStates[userID]
	if !ok {
		return false, nil
	}
	if state.Status != UserAccountStatusBanned {
		return false, nil
	}
	return !state.Expired(time.Now().UTC()), nil
}
