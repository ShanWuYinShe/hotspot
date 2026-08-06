package groups

import (
	"context"

	"time"

	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) ListMyInvites(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows, err := queryable.Query(ctx, queryListMyInvites, userID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupInviteItem, 0)
	for rows.Next() {
		var item GroupInviteItem
		var inviterUserID, groupID, message, statusText, groupName, groupAvatar string
		var membersCount int
		var createdAt time.Time
		var expiresAt, respondedAt *time.Time
		if err := rows.Scan(&item.InviteID, &groupID, &groupName, &groupAvatar, &membersCount, &inviterUserID, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
			return nil, "", err
		}
		item.Group = GroupInfo{GroupID: groupID, Name: groupName, AvatarURL: groupAvatar, MembersCount: membersCount}
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
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) AcceptInvite(ctx context.Context, userID string, inviteID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	invite, err := r.loadInvite(ctx, inviteID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	// Wrap the invite-status update and the member insert in a single tx so a
	// failure of the second statement rolls back the invite acceptance.
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusAccepted), now); err != nil {
			return err
		}
		return queryable.Exec(txCtx, queryInsertGroupMember, invite.Group.GroupID, userID, string(GroupRoleMember), now)
	})
}

func (r *PostgresRepository) RejectInvite(ctx context.Context, inviteID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusRejected), time.Now().UTC())
}

func (r *PostgresRepository) InviteMember(ctx context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return Invitation{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return Invitation{}, persistence.ErrNotImplemented
	}
	var existingID string
	var existingExpiresAt *time.Time
	if err := queryable.QueryRow(ctx, queryFindPendingGroupInvite, req.GroupID, req.UserID).Scan(&existingID, &existingExpiresAt); err == nil && existingID != "" {
		return Invitation{InviteID: existingID, ExpiresAt: derefTime(existingExpiresAt)}, nil
	}
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	inviteID, err := idgen.WithPrefix("invite_")
	if err != nil {
		return Invitation{}, err
	}
	if err := queryable.Exec(ctx, queryInsertGroupInvite, inviteID, req.GroupID, userID, req.UserID, nullableStringPtr(req.Message), string(ApplicationStatusPending), now, expiresAt); err != nil {
		return Invitation{}, err
	}
	return Invitation{InviteID: inviteID, ExpiresAt: expiresAt.Format(time.RFC3339)}, nil
}

func (r *PostgresRepository) ListSentInvites(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows, err := queryable.Query(ctx, queryListSentInvites, groupID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupSentInviteItem, 0)
	for rows.Next() {
		var item GroupSentInviteItem
		var inviteeUserID, message, statusText string
		var createdAt time.Time
		var expiresAt, respondedAt *time.Time
		if err := rows.Scan(&item.InviteID, &inviteeUserID, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
			return nil, "", err
		}
		item.Invitee = UserInfo{UserID: inviteeUserID, Nickname: inviteeUserID}
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
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) RevokeInvite(ctx context.Context, groupID string, inviteID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusRevoked), time.Now().UTC())
}

func (r *PostgresRepository) loadInvite(ctx context.Context, inviteID string) (GroupInviteItem, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupInviteItem{}, persistence.ErrNotImplemented
	}
	var item GroupInviteItem
	var groupID, inviterUserID, inviteeUserID, message, statusText string
	var createdAt time.Time
	var expiresAt, respondedAt *time.Time
	if err := queryable.QueryRow(ctx, queryGetGroupInvite, inviteID).Scan(&item.InviteID, &groupID, &inviterUserID, &inviteeUserID, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
		return GroupInviteItem{}, err
	}
	group, err := getGroupBase(ctx, queryable, groupID)
	if err != nil {
		return GroupInviteItem{}, err
	}
	count, err := countGroupMembers(ctx, queryable, groupID)
	if err != nil {
		return GroupInviteItem{}, err
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
	_ = respondedAt
	_ = inviteeUserID
	return item, nil
}
