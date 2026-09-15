package groups

import (
	"context"
	"errors"

	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ListMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	cursorJoinedAt, cursorUserID, err := decodeGroupPageCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var cursorJoinedAtValue any
	if !cursorJoinedAt.IsZero() {
		cursorJoinedAtValue = cursorJoinedAt
	}
	rows, err := queryable.Query(ctx, queryListGroupMembers, groupID, roleValue, cursorJoinedAtValue, cursorUserID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupMemberItem, 0, limit+1)
	joinedAts := make([]time.Time, 0, limit+1)
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
		joinedAts = append(joinedAts, joinedAt)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) <= limit {
		return items, "", nil
	}
	next := encodeGroupPageCursor(joinedAts[limit-1], items[limit-1].User.UserID)
	return items[:limit], next, nil
}

// ListAdminMembers returns the admin-perspective member list (includes the
// truthy muted_until; null = not muted). groups/admin/members/list only.
func (r *PostgresRepository) ListAdminMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]AdminGroupMemberItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	cursorJoinedAt, cursorUserID, err := decodeGroupPageCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var cursorJoinedAtValue any
	if !cursorJoinedAt.IsZero() {
		cursorJoinedAtValue = cursorJoinedAt
	}
	rows, err := queryable.Query(ctx, queryListAdminGroupMembers, groupID, roleValue, cursorJoinedAtValue, cursorUserID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]AdminGroupMemberItem, 0, limit+1)
	joinedAts := make([]time.Time, 0, limit+1)
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
		joinedAts = append(joinedAts, joinedAt)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) <= limit {
		return items, "", nil
	}
	next := encodeGroupPageCursor(joinedAts[limit-1], items[limit-1].User.UserID)
	return items[:limit], next, nil
}

// IsMember is a direct membership probe (single indexed lookup) used on hot
// paths such as realtime authorization. It replaces the previous pattern of
// paging through the whole member list to answer a yes/no question.
func (r *PostgresRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	cursorJoinedAt, cursorUserID, err := decodeGroupPageCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var cursorJoinedAtValue any
	if !cursorJoinedAt.IsZero() {
		cursorJoinedAtValue = cursorJoinedAt
	}
	rows, err := queryable.Query(ctx, querySearchGroupMembers, groupID, roleValue, keyword, cursorJoinedAtValue, cursorUserID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupMemberItem, 0, limit+1)
	joinedAts := make([]time.Time, 0, limit+1)
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
		joinedAts = append(joinedAts, joinedAt)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) <= limit {
		return items, "", nil
	}
	next := encodeGroupPageCursor(joinedAts[limit-1], items[limit-1].User.UserID)
	return items[:limit], next, nil
}

func (r *PostgresRepository) SetMemberRole(ctx context.Context, userID string, req SetMemberRoleRequest) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupMemberRole, req.GroupID, req.UserID, string(req.Role))
}

func (r *PostgresRepository) MuteMember(ctx context.Context, userID string, req MuteMemberRequest) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	mutedUntil := time.Now().UTC().Add(time.Duration(req.DurationMinutes) * time.Minute)
	return queryable.Exec(ctx, queryMuteGroupMember, req.GroupID, req.UserID, mutedUntil)
}

func (r *PostgresRepository) UnmuteMember(ctx context.Context, userID string, req UnmuteMemberRequest) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUnmuteGroupMember, req.GroupID, req.UserID)
}

func (r *PostgresRepository) BanMember(ctx context.Context, userID string, req BanMemberRequest) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	// 插封禁 + 移除成员 + 撤销 pending 申请/邀请包进同一事务：避免「已封禁
	// 但仍是成员」的中间状态，删除成员失败不再被吞；级联撤销是对契约
	// spec api_spec_group.md → set_ban「封禁时被 ban 用户发出的该群 pending
	// 申请置 revoked；发给被 ban 用户的该群 pending 邀请置 revoked」的落实，
	// 同时堵住「封禁后经邀请直通 / 审批通过回归」两条旁路。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryInsertGroupBan, req.GroupID, req.UserID, userID, nullableStringPtr(req.Reason), now); err != nil {
			return err
		}
		if err := txQueryable.Exec(txCtx, queryDeleteGroupMember, req.GroupID, req.UserID); err != nil {
			return err
		}
		if err := txQueryable.Exec(txCtx, queryRevokeApplicationsByApplicant, req.GroupID, req.UserID, now); err != nil {
			return err
		}
		return txQueryable.Exec(txCtx, queryRevokeInvitesByInvitee, req.GroupID, req.UserID, now)
	})
}

func (r *PostgresRepository) UnbanMember(ctx context.Context, userID string, req UnbanMemberRequest) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupBan, req.GroupID, req.UserID)
}

func (r *PostgresRepository) ListBannedMembers(ctx context.Context, groupID string, limit int, cursor string) ([]AdminBannedMemberItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListGroupBans, groupID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]AdminBannedMemberItem, 0)
	for rows.Next() {
		var item AdminBannedMemberItem
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupMember, req.GroupID, req.UserID)
}

func (r *PostgresRepository) TransferOwnership(ctx context.Context, userID string, req TransferOwnershipRequest) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	// 降级旧 owner + 提升新 owner + 更新 owner_id 三步包进同一事务：
	// 第三步失败不会留下 owner_id 与角色不一致的状态。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		// 降旧 owner 走专用守卫（仅 role='owner' 行可降）：queryUpdateGroupMemberRole
		// 的 role <> 'owner' 守卫会挡住这条合法变更。
		if err := txQueryable.Exec(txCtx, queryDemoteGroupOwnerRole, req.GroupID, userID, string(GroupRoleAdmin)); err != nil {
			return err
		}
		if err := txQueryable.Exec(txCtx, queryUpdateGroupMemberRole, req.GroupID, req.NewOwnerID, string(GroupRoleOwner)); err != nil {
			return err
		}
		return txQueryable.Exec(txCtx, queryUpdateGroupOwner, req.GroupID, req.NewOwnerID)
	})
}
