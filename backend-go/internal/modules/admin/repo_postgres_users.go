package admin

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/types"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func nullableString(value any) any {
	switch v := value.(type) {
	case *string:
		if v == nil || strings.TrimSpace(*v) == "" {
			return nil
		}
		return *v
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return v
	default:
		return nil
	}
}

// strPtr 返回 string 的指针（用于 AdminUserItem 中常量 nullable 的 PII IP 字段：
// 空串仍序列化为空指针，由 service 层按 read_ip 权限决定置 null）。
func strPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (r *PostgresRepository) SearchUsers(ctx context.Context, req AdminUserSearchRequest) (adminUserPage, error) {
	queryable, ok := r.queryable()
	if !ok {
		return adminUserPage{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryAdminSearchUsers, req.UserID, req.Nickname, req.LoginID, string(req.Status), string(req.UserType))
	if err != nil {
		return adminUserPage{}, err
	}
	defer rows.Close()
	items := make([]AdminUserItem, 0)
	for rows.Next() {
		item, err := scanAdminUserRow(rows)
		if err != nil {
			return adminUserPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return adminUserPage{}, err
	}
	cursorRaw := types.PaginationCursor(req.Pagination)
	cursor, apiErr := decodeAdminUserCursor(cursorRaw)
	if apiErr != nil {
		return adminUserPage{}, apiErr
	}
	filtered := make([]AdminUserItem, 0, len(items))
	for _, item := range items {
		if cursorRaw != "" {
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
	page := adminUserPage{Items: make([]AdminUserItem, 0, limit)}
	for idx, item := range filtered {
		if idx >= limit {
			page.Next = &adminUserCursor{CreatedAt: timeFromRFC3339(item.CreatedAt), UserID: item.User.UserID}
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (AdminUserItem, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AdminUserItem{}, false, persistence.ErrNotImplemented
	}
	item, err := scanAdminUserRow(queryable.QueryRow(ctx, queryAdminGetUser, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminUserItem{}, false, nil
		}
		return AdminUserItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) GetUserBanState(ctx context.Context, userID string) (UserBanState, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return UserBanState{}, false, persistence.ErrNotImplemented
	}
	var state UserBanState
	var bannedAt *time.Time
	if err := queryable.QueryRow(ctx, queryAdminGetUserBanState, userID).Scan(&state.Status, &state.Reason, &state.DurationDays, &bannedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserBanState{}, false, nil
		}
		return UserBanState{}, false, err
	}
	if bannedAt != nil {
		state.BannedAt = *bannedAt
	}
	return state, true, nil
}

func (r *PostgresRepository) SetUserBanState(ctx context.Context, userID string, state UserBanState) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	status := state.Status
	if status == "" {
		status = UserAccountStatusNormal
	}
	var reason any
	var duration any
	var bannedAt any
	if status == UserAccountStatusBanned {
		reason = nullableString(strings.TrimSpace(state.Reason))
		duration = state.DurationDays
		if state.BannedAt.IsZero() {
			bannedAt = time.Now().UTC()
		} else {
			bannedAt = state.BannedAt
		}
	} else {
		reason = nil
		duration = nil
		bannedAt = nil
	}
	return queryable.Exec(ctx, queryAdminSetUserBanState, userID, string(status), reason, duration, bannedAt)
}

func (r *PostgresRepository) ListUserSessions(ctx context.Context, userID string) ([]string, error) {
	queryable, ok := r.queryable()
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryAdminListUserSessions, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, err
		}
		items = append(items, sessionID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, sessionID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryAdminRevokeSession, sessionID)
}

func (r *PostgresRepository) IsBanned(ctx context.Context, userID string) (bool, error) {
	state, found, err := r.GetUserBanState(ctx, userID)
	if err != nil || !found {
		return false, err
	}
	return state.Status == UserAccountStatusBanned, nil
}
