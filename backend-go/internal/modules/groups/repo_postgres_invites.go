package groups

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ListMyInvites(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	cursorAt, cursorInviteID, err := decodeGroupPageCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var cursorAtValue any
	if !cursorAt.IsZero() {
		cursorAtValue = cursorAt
	}
	rows, err := queryable.Query(ctx, queryListMyInvites, userID, statusValue, cursorAtValue, cursorInviteID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupInviteItem, 0, limit+1)
	createdAts := make([]time.Time, 0, limit+1)
	for rows.Next() {
		var item GroupInviteItem
		var inviterUserID, inviterNickname, groupID, message, statusText, groupName, groupAvatar string
		var membersCount int
		var createdAt time.Time
		var expiresAt, respondedAt *time.Time
		if err := rows.Scan(&item.InviteID, &groupID, &groupName, &groupAvatar, &membersCount, &inviterUserID, &inviterNickname, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
			return nil, "", err
		}
		item.Group = GroupInfo{GroupID: groupID, Name: groupName, AvatarURL: types.StrPtr(groupAvatar), MembersCount: membersCount}
		item.Inviter = UserInfo{UserID: inviterUserID, Nickname: inviterNickname}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if expiresAt != nil {
			value := expiresAt.UTC().Format(time.RFC3339)
			item.ExpiresAt = &value
		}
		if respondedAt != nil {
			value := respondedAt.UTC().Format(time.RFC3339)
			item.RespondedAt = &value
		}
		items = append(items, item)
		createdAts = append(createdAts, createdAt)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) <= limit {
		return items, "", nil
	}
	next := encodeGroupPageCursor(createdAts[limit-1], items[limit-1].InviteID)
	return items[:limit], next, nil
}

func (r *PostgresRepository) AcceptInvite(ctx context.Context, userID string, inviteID string) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	invite, err := r.loadInvite(ctx, inviteID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	// Wrap the invite-status update and the member insert in a single tx so a
	// failure of the second statement rolls back the invite acceptance.
	// pending 守卫 + RETURNING：已被撤销/拒绝/过期或并发接受的邀请更新 0 行，
	// 返回冲突而非重复入群。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := enforceGroupCapacity(txCtx, queryable, invite.Group.GroupID); err != nil {
			return err
		}
		var updated string
		if err := queryable.QueryRow(txCtx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusAccepted), now).Scan(&updated); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInviteExpired
			}
			return err
		}
		return queryable.Exec(txCtx, queryInsertGroupMember, invite.Group.GroupID, userID, string(GroupRoleMember), now)
	})
}

func (r *PostgresRepository) RejectInvite(ctx context.Context, inviteID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var updated string
	if err := queryable.QueryRow(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusRejected), time.Now().UTC()).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInviteExpired
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) InviteMember(ctx context.Context, userID string, req InviteMemberRequest, expiresAt *time.Time) (Invitation, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return Invitation{}, persistence.ErrNotImplemented
	}
	var existingID string
	var existingExpiresAt *time.Time
	if err := queryable.QueryRow(ctx, queryFindPendingGroupInvite, req.GroupID, req.UserID).Scan(&existingID, &existingExpiresAt); err == nil && existingID != "" {
		expiresAtStr := derefTime(existingExpiresAt)
		var expiresAtPtr *string
		if expiresAtStr != "" {
			expiresAtPtr = &expiresAtStr
		}
		return Invitation{InviteID: &existingID, ExpiresAt: expiresAtPtr}, nil
	}
	inviteID, err := idgen.WithPrefix("invite_")
	if err != nil {
		return Invitation{}, err
	}
	// expiresAt 为 nil 表示永不过期（TTL=0，spec types.md group_invite_ttl_hours）
	var expiresAtDB *time.Time
	if expiresAt != nil {
		expiresAtDB = expiresAt
	}
	if err := queryable.Exec(ctx, queryInsertGroupInvite, inviteID, req.GroupID, userID, req.UserID, nullableStringPtr(req.Message), string(ApplicationStatusPending), time.Now().UTC(), expiresAtDB); err != nil {
		return Invitation{}, err
	}
	if expiresAtDB == nil {
		return Invitation{InviteID: &inviteID}, nil
	}
	expiresAtValue := expiresAtDB.UTC().Format(time.RFC3339)
	return Invitation{InviteID: &inviteID, ExpiresAt: &expiresAtValue}, nil
}

func (r *PostgresRepository) ListSentInvites(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	cursorAt, cursorInviteID, err := decodeGroupPageCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var cursorAtValue any
	if !cursorAt.IsZero() {
		cursorAtValue = cursorAt
	}
	rows, err := queryable.Query(ctx, queryListSentInvites, groupID, statusValue, cursorAtValue, cursorInviteID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupSentInviteItem, 0, limit+1)
	createdAts := make([]time.Time, 0, limit+1)
	for rows.Next() {
		var item GroupSentInviteItem
		var inviterUserID, inviterNickname, inviteeUserID, inviteeNickname, message, statusText string
		var createdAt time.Time
		var expiresAt, respondedAt *time.Time
		if err := rows.Scan(&item.InviteID, &inviterUserID, &inviterNickname, &inviteeUserID, &inviteeNickname, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
			return nil, "", err
		}
		item.Inviter = UserInfo{UserID: inviterUserID, Nickname: inviterNickname}
		item.Invitee = UserInfo{UserID: inviteeUserID, Nickname: inviteeNickname}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if expiresAt != nil {
			value := expiresAt.UTC().Format(time.RFC3339)
			item.ExpiresAt = &value
		}
		if respondedAt != nil {
			value := respondedAt.UTC().Format(time.RFC3339)
			item.RespondedAt = &value
		}
		items = append(items, item)
		createdAts = append(createdAts, createdAt)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) <= limit {
		return items, "", nil
	}
	next := encodeGroupPageCursor(createdAts[limit-1], items[limit-1].InviteID)
	return items[:limit], next, nil
}

func (r *PostgresRepository) RevokeInvite(ctx context.Context, groupID string, inviteID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	// 归属双键 + pending 守卫：仅本群的 pending 邀请可撤销。
	var updated string
	if err := queryable.QueryRow(ctx, queryRevokeGroupInvite, inviteID, string(ApplicationStatusRevoked), time.Now().UTC(), groupID).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInviteExpired
		}
		return err
	}
	return nil
}

// GetInvite 按 invite_id 直查邀请（repo.go 接口注释）。found=false 表示
// 邀请不存在。
func (r *PostgresRepository) GetInvite(ctx context.Context, inviteID string) (GroupInviteItem, string, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return GroupInviteItem{}, "", false, persistence.ErrNotImplemented
	}
	item, invitee, err := loadInviteWithInvitee(ctx, queryable, inviteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupInviteItem{}, "", false, nil
		}
		return GroupInviteItem{}, "", false, err
	}
	return item, invitee, true, nil
}

func (r *PostgresRepository) loadInvite(ctx context.Context, inviteID string) (GroupInviteItem, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return GroupInviteItem{}, persistence.ErrNotImplemented
	}
	item, _, err := loadInviteWithInvitee(ctx, queryable, inviteID)
	return item, err
}

// loadInviteWithInvitee 按主键取邀请并补全 group base 与成员数，同时返回
// invitee_user_id（GroupInviteItem 是 API 输出形状，不携带该字段）。
func loadInviteWithInvitee(ctx context.Context, queryable persistence.Queryable, inviteID string) (GroupInviteItem, string, error) {
	var item GroupInviteItem
	var groupID, inviterUserID, inviteeUserID, message, statusText string
	var createdAt time.Time
	var expiresAt, respondedAt *time.Time
	if err := queryable.QueryRow(ctx, queryGetGroupInvite, inviteID).Scan(&item.InviteID, &groupID, &inviterUserID, &inviteeUserID, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
		return GroupInviteItem{}, "", err
	}
	group, err := getGroupBase(ctx, queryable, groupID)
	if err != nil {
		return GroupInviteItem{}, "", err
	}
	count, err := countGroupMembers(ctx, queryable, groupID)
	if err != nil {
		return GroupInviteItem{}, "", err
	}
	group.Group.MembersCount = count
	item.Group = group.Group
	item.Inviter = UserInfo{UserID: inviterUserID, Nickname: inviterUserID}
	if message != "" {
		item.Message = &message
	}
	item.Status = ApplicationStatus(statusText)
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if expiresAt != nil {
		value := expiresAt.UTC().Format(time.RFC3339)
		item.ExpiresAt = &value
	}
	if respondedAt != nil {
		value := respondedAt.UTC().Format(time.RFC3339)
		item.RespondedAt = &value
	}
	return item, inviteeUserID, nil
}
