package groups

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetApplication(ctx context.Context, applicationID string) (GroupApplicationItem, bool, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return GroupApplicationItem{}, false, persistence.ErrNotImplemented
	}
	item, err := r.loadApplication(ctx, applicationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupApplicationItem{}, false, nil
		}
		return GroupApplicationItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) ListApplications(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	cursorCreatedAt, cursorApplicationID, err := decodeGroupPageCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var cursorCreatedAtValue any
	if !cursorCreatedAt.IsZero() {
		cursorCreatedAtValue = cursorCreatedAt
	}
	rows, err := queryable.Query(ctx, queryListGroupApplications, groupID, statusValue, cursorCreatedAtValue, cursorApplicationID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupApplicationItem, 0, limit+1)
	createdAts := make([]time.Time, 0, limit+1)
	for rows.Next() {
		var item GroupApplicationItem
		var groupID, groupName, groupAvatar, message, statusText, applicantUserID, applicantNickname string
		var membersCount int
		var createdAt time.Time
		var handledAt *time.Time
		var handledByUserID *string
		if err := rows.Scan(&item.ApplicationID, &groupID, &groupName, &groupAvatar, &membersCount, &applicantUserID, &applicantNickname, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
			return nil, "", err
		}
		item.GroupID = groupID
		item.Group = GroupInfo{GroupID: groupID, Name: groupName, AvatarURL: types.StrPtr(groupAvatar), MembersCount: membersCount}
		item.Applicant = UserInfo{UserID: applicantUserID, Nickname: applicantNickname}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if handledAt != nil {
			value := handledAt.UTC().Format(time.RFC3339)
			item.HandledAt = &value
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
	next := encodeGroupPageCursor(createdAts[limit-1], items[limit-1].ApplicationID)
	return items[:limit], next, nil
}

func (r *PostgresRepository) ApproveApplication(ctx context.Context, approverID string, applicationID string) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	record, err := r.loadApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	// Wrap the application-status update and the member insert in a single tx.
	// Previously these were two separate Execs: if the second failed the
	// application was already marked accepted with no matching member row.
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		// 封禁复检下沉到事务内（审计 F-04 收口）：服务层的 IsBanned 检查与
		// 本事务提交之间存在竞态窗口（检查后、提交前被 ban），此处同事务
		// 复检封死旁路；BanMember 的 pending 申请级联撤销仍是兜底。
		var one int
		err := queryable.QueryRow(txCtx, queryIsGroupBanned, record.GroupID, record.Applicant.UserID).Scan(&one)
		switch {
		case err == nil:
			return ErrBannedInGroup
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		if err := enforceGroupCapacity(txCtx, queryable, record.GroupID); err != nil {
			return err
		}
		if err := queryable.QueryRow(txCtx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusAccepted), now, approverID).Scan(&applicationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAlreadyProcessed
			}
			return err
		}
		return queryable.Exec(txCtx, queryInsertGroupMember, record.GroupID, record.Applicant.UserID, string(GroupRoleMember), now)
	})
}

func (r *PostgresRepository) RejectApplication(ctx context.Context, applicationID string, reason *string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var updated string
	if err := queryable.QueryRow(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRejected), time.Now().UTC(), nil).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAlreadyProcessed
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) ListMyApplications(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	// keyset 游标消费 + limit+1 探测下一页（对齐同文件 ListApplications；
	// 此前忽略 limit/cursor 全量返回，next_cursor 恒 null——审查 P1-2）。
	cursorCreatedAt, cursorApplicationID, err := decodeGroupPageCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var cursorCreatedAtValue any
	if !cursorCreatedAt.IsZero() {
		cursorCreatedAtValue = cursorCreatedAt
	}
	rows, err := queryable.Query(ctx, queryListMyGroupApplications, userID, statusValue, cursorCreatedAtValue, cursorApplicationID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupApplicationItem, 0, limit+1)
	createdAts := make([]time.Time, 0, limit+1)
	for rows.Next() {
		var item GroupApplicationItem
		var groupID, groupName, groupAvatar, applicantUserID, applicantNickname, message, statusText string
		var membersCount int
		var createdAt time.Time
		var handledAt *time.Time
		var handledByUserID *string
		if err := rows.Scan(&item.ApplicationID, &groupID, &groupName, &groupAvatar, &membersCount, &applicantUserID, &applicantNickname, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
			return nil, "", err
		}
		item.GroupID = groupID
		item.Group = GroupInfo{GroupID: groupID, Name: groupName, AvatarURL: types.StrPtr(groupAvatar), MembersCount: membersCount}
		item.Applicant = UserInfo{UserID: applicantUserID, Nickname: applicantNickname}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if handledAt != nil {
			value := handledAt.UTC().Format(time.RFC3339)
			item.HandledAt = &value
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
	next := encodeGroupPageCursor(createdAts[limit-1], items[limit-1].ApplicationID)
	return items[:limit], next, nil
}

func (r *PostgresRepository) CancelApplication(ctx context.Context, applicationID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var updated string
	if err := queryable.QueryRow(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRevoked), time.Now().UTC(), nil).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAlreadyProcessed
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) loadApplication(ctx context.Context, applicationID string) (GroupApplicationItem, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return GroupApplicationItem{}, persistence.ErrNotImplemented
	}
	var item GroupApplicationItem
	var groupID, groupName, groupAvatar, applicantUserID, applicantNickname, message, statusText string
	var membersCount int
	var createdAt time.Time
	var handledAt *time.Time
	var handledByUserID *string
	if err := queryable.QueryRow(ctx, queryGetGroupApplication, applicationID).Scan(&item.ApplicationID, &groupID, &groupName, &groupAvatar, &membersCount, &applicantUserID, &applicantNickname, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
		return GroupApplicationItem{}, err
	}
	item.GroupID = groupID
	item.Group = GroupInfo{GroupID: groupID, Name: groupName, AvatarURL: types.StrPtr(groupAvatar), MembersCount: membersCount}
	item.Applicant = UserInfo{UserID: applicantUserID, Nickname: applicantNickname}
	if message != "" {
		item.Message = &message
	}
	item.Status = ApplicationStatus(statusText)
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if handledAt != nil {
		value := handledAt.UTC().Format(time.RFC3339)
		item.HandledAt = &value
	}
	return item, nil
}
