package groups

import (
	"context"
	"errors"

	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ListMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	rows, err := queryable.Query(ctx, queryListGroupMembers, groupID, roleValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupMemberItem, 0)
	for rows.Next() {
		var item GroupMemberItem
		var roleText string
		var joinedAt time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &roleText, &joinedAt); err != nil {
			return nil, "", err
		}
		item.Role = GroupRole(roleText)
		item.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

// ListAdminMembers returns the admin-perspective member list (includes the
// truthy muted_until; null = not muted). groups/admin/members/list only.
func (r *PostgresRepository) ListAdminMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]AdminGroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	rows, err := queryable.Query(ctx, queryListAdminGroupMembers, groupID, roleValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]AdminGroupMemberItem, 0)
	for rows.Next() {
		var item AdminGroupMemberItem
		var roleText string
		var joinedAt time.Time
		var mutedUntil *time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &roleText, &joinedAt, &mutedUntil); err != nil {
			return nil, "", err
		}
		item.Role = GroupRole(roleText)
		item.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		item.MutedUntil = formatMutedUntil(mutedUntil)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

// IsMember is a direct membership probe (single indexed lookup) used on hot
// paths such as realtime authorization. It replaces the previous pattern of
// paging through the whole member list to answer a yes/no question.
func (r *PostgresRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	rows, err := queryable.Query(ctx, querySearchGroupMembers, groupID, roleValue, keyword)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupMemberItem, 0)
	for rows.Next() {
		var item GroupMemberItem
		var roleText string
		var joinedAt time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &roleText, &joinedAt); err != nil {
			return nil, "", err
		}
		item.Role = GroupRole(roleText)
		item.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) SetMemberRole(ctx context.Context, userID string, req SetMemberRoleRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupMemberRole, req.GroupID, req.UserID, string(req.Role))
}

func (r *PostgresRepository) MuteMember(ctx context.Context, userID string, req MuteMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	mutedUntil := time.Now().UTC().Add(time.Duration(req.DurationMinutes) * time.Minute)
	return queryable.Exec(ctx, queryMuteGroupMember, req.GroupID, req.UserID, mutedUntil)
}

func (r *PostgresRepository) UnmuteMember(ctx context.Context, userID string, req UnmuteMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUnmuteGroupMember, req.GroupID, req.UserID)
}

func (r *PostgresRepository) BanMember(ctx context.Context, userID string, req BanMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	if _, ok := r.factory.DB.(persistence.QueryableDB); !ok {
		return persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	// 插封禁 + 移除成员两步包进同一事务：避免「已封禁但仍是成员」的
	// 中间状态，且删除成员失败不再被吞。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryInsertGroupBan, req.GroupID, req.UserID, userID, nullableStringPtr(req.Reason), now); err != nil {
			return err
		}
		return txQueryable.Exec(txCtx, queryDeleteGroupMember, req.GroupID, req.UserID)
	})
}

func (r *PostgresRepository) UnbanMember(ctx context.Context, userID string, req UnbanMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupBan, req.GroupID, req.UserID)
}

func (r *PostgresRepository) ListBannedMembers(ctx context.Context, groupID string, limit int, cursor string) ([]BannedMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListGroupBans, groupID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]BannedMemberItem, 0)
	for rows.Next() {
		var item BannedMemberItem
		var bannedAt time.Time
		var reason *string
		var bannedByUserID *string
		if err := rows.Scan(&item.User.UserID, &bannedAt, &reason, &bannedByUserID); err != nil {
			return nil, "", err
		}
		item.User.Nickname = item.User.UserID
		item.BannedAt = bannedAt.UTC().Format(time.RFC3339)
		item.Reason = reason
		if bannedByUserID != nil && *bannedByUserID != "" {
			item.BannedBy = &UserInfo{UserID: *bannedByUserID, Nickname: *bannedByUserID}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

// IsBanned reports whether the user is banned from the group (join-time check, C.3).
func (r *PostgresRepository) IsBanned(ctx context.Context, groupID, userID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var one int
	err := queryable.QueryRow(ctx, queryIsGroupBanned, groupID, userID).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, userID string, req RemoveMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupMember, req.GroupID, req.UserID)
}

func (r *PostgresRepository) TransferOwnership(ctx context.Context, userID string, req TransferOwnershipRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	if _, ok := r.factory.DB.(persistence.QueryableDB); !ok {
		return persistence.ErrNotImplemented
	}
	// 降级旧 owner + 提升新 owner + 更新 owner_id 三步包进同一事务：
	// 第三步失败不会留下 owner_id 与角色不一致的状态。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryUpdateGroupMemberRole, req.GroupID, userID, string(GroupRoleAdmin)); err != nil {
			return err
		}
		if err := txQueryable.Exec(txCtx, queryUpdateGroupMemberRole, req.GroupID, req.NewOwnerID, string(GroupRoleOwner)); err != nil {
			return err
		}
		return txQueryable.Exec(txCtx, queryUpdateGroupOwner, req.GroupID, req.NewOwnerID)
	})
}
