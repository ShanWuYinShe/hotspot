package groups

import (
	"context"
	"errors"

	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetApplication(ctx context.Context, applicationID string) (GroupApplicationItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
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
	rows, err := queryable.Query(ctx, queryListGroupApplications, groupID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupApplicationItem, 0)
	for rows.Next() {
		var item GroupApplicationItem
		var message, statusText, applicantUserID string
		var createdAt time.Time
		var handledAt *time.Time
		var handledByUserID *string
		if err := rows.Scan(&item.ApplicationID, &applicantUserID, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
			return nil, "", err
		}
		item.Applicant = UserInfo{UserID: applicantUserID, Nickname: applicantUserID}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if handledAt != nil {
			value := handledAt.UTC().Format(time.RFC3339)
			item.HandledAt = &value
		}
		if handledByUserID != nil && *handledByUserID != "" {
			item.HandledBy = &UserInfo{UserID: *handledByUserID, Nickname: *handledByUserID}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) ApproveApplication(ctx context.Context, approverID string, applicationID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
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
		if err := queryable.Exec(txCtx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusAccepted), now, approverID); err != nil {
			return err
		}
		return queryable.Exec(txCtx, queryInsertGroupMember, record.GroupID, record.Applicant.UserID, string(GroupRoleMember), now)
	})
}

func (r *PostgresRepository) RejectApplication(ctx context.Context, applicationID string, reason *string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRejected), time.Now().UTC(), nil)
}

func (r *PostgresRepository) ListMyApplications(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
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
	rows, err := queryable.Query(ctx, queryListMyGroupApplications, userID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupApplicationItem, 0)
	for rows.Next() {
		var item GroupApplicationItem
		var message, statusText, groupID string
		var createdAt time.Time
		var handledAt *time.Time
		var handledByUserID *string
		if err := rows.Scan(&item.ApplicationID, &groupID, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
			return nil, "", err
		}
		item.Applicant = UserInfo{UserID: userID, Nickname: userID}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if handledAt != nil {
			value := handledAt.UTC().Format(time.RFC3339)
			item.HandledAt = &value
		}
		if handledByUserID != nil && *handledByUserID != "" {
			item.HandledBy = &UserInfo{UserID: *handledByUserID, Nickname: *handledByUserID}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) CancelApplication(ctx context.Context, applicationID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRevoked), time.Now().UTC(), nil)
}

func (r *PostgresRepository) loadApplication(ctx context.Context, applicationID string) (GroupApplicationItem, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupApplicationItem{}, persistence.ErrNotImplemented
	}
	var item GroupApplicationItem
	var groupID, applicantUserID, message, statusText string
	var createdAt time.Time
	var handledAt *time.Time
	var handledByUserID *string
	if err := queryable.QueryRow(ctx, queryGetGroupApplication, applicationID).Scan(&item.ApplicationID, &groupID, &applicantUserID, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
		return GroupApplicationItem{}, err
	}
	item.GroupID = groupID
	item.Applicant = UserInfo{UserID: applicantUserID, Nickname: applicantUserID}
	if message != "" {
		item.Message = &message
	}
	item.Status = ApplicationStatus(statusText)
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if handledAt != nil {
		value := handledAt.UTC().Format(time.RFC3339)
		item.HandledAt = &value
	}
	if handledByUserID != nil && *handledByUserID != "" {
		item.HandledBy = &UserInfo{UserID: *handledByUserID, Nickname: *handledByUserID}
	}
	return item, nil
}
