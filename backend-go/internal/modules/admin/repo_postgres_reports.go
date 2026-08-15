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

func (r *PostgresRepository) SearchReports(ctx context.Context, req AdminReportListRequest) (adminReportPage, error) {
	queryable, ok := r.queryable()
	if !ok {
		return adminReportPage{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryAdminSearchReports, string(req.Status), string(req.TargetType))
	if err != nil {
		return adminReportPage{}, err
	}
	defer rows.Close()
	items := make([]AdminReportItem, 0)
	for rows.Next() {
		record, err := scanAdminReportRow(rows, false)
		if err != nil {
			return adminReportPage{}, err
		}
		items = append(items, record.Item)
	}
	if err := rows.Err(); err != nil {
		return adminReportPage{}, err
	}
	cursorRaw := types.PaginationCursor(req.Pagination)
	cursor, apiErr := decodeAdminReportCursor(cursorRaw)
	if apiErr != nil {
		return adminReportPage{}, apiErr
	}
	filtered := make([]AdminReportItem, 0, len(items))
	for _, item := range items {
		if cursorRaw != "" {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursor.CreatedAt) {
				continue
			}
			if createdAt.Equal(cursor.CreatedAt) && item.ReportID >= cursor.ReportID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	page := adminReportPage{Items: make([]AdminReportItem, 0, limit)}
	for idx, item := range filtered {
		if idx >= limit {
			page.Next = &adminReportCursor{CreatedAt: timeFromRFC3339(item.CreatedAt), ReportID: item.ReportID}
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (r *PostgresRepository) GetReportHandleRecord(ctx context.Context, reportID string) (reportHandleRecord, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return reportHandleRecord{}, false, persistence.ErrNotImplemented
	}
	record, err := scanAdminReportRow(queryable.QueryRow(ctx, queryAdminGetReportHandleRecord, reportID), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reportHandleRecord{}, false, nil
		}
		return reportHandleRecord{}, false, err
	}
	return record, true, nil
}

func (r *PostgresRepository) HandleReport(ctx context.Context, reportID string, action ReportAction, note string, handledBy *UserInfo) (AdminReportItem, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AdminReportItem{}, false, persistence.ErrNotImplemented
	}
	record, found, err := r.GetReportHandleRecord(ctx, reportID)
	if err != nil || !found {
		return AdminReportItem{}, found, err
	}
	status := ReportStatusPending
	switch action {
	case ReportActionResolve:
		status = ReportStatusResolved
	case ReportActionDismiss:
		status = ReportStatusDismissed
	}
	now := time.Now().UTC()
	var handledByID any
	if handledBy != nil {
		handledByID = handledBy.UserID
	}
	if err := queryable.Exec(ctx, queryAdminHandleReport, reportID, string(status), now, handledByID, nullableString(strings.TrimSpace(note))); err != nil {
		return AdminReportItem{}, false, err
	}
	record.Item.Status = status
	handledAt := now.Format(time.RFC3339)
	record.Item.HandledAt = &handledAt
	record.Item.HandleNote = types.StrPtr(strings.TrimSpace(note))
	if handledBy != nil {
		hb := *handledBy
		record.Item.HandledBy = &hb
	}
	return record.Item, true, nil
}

func (r *PostgresRepository) RecallGroupMessage(ctx context.Context, messageID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, "UPDATE group_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE", messageID); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) RecallPrivateMessage(ctx context.Context, messageID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, "UPDATE private_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE", messageID); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteMoment(ctx context.Context, momentID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, "DELETE FROM moments WHERE moment_id = $1", momentID)
}

func (r *PostgresRepository) DeleteComment(ctx context.Context, commentID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, "DELETE FROM comments WHERE comment_id = $1", commentID)
}
