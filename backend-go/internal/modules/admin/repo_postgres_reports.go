package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	reportsmodule "hotspot/backend-go/internal/modules/reports"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

// errReportAlreadyHandled 表示举报已被并发裁决处理（repo 层 sentinel，
// service 层据此返回业务冲突而非重复执行副作用）。
var errReportAlreadyHandled = errors.New("admin: report already handled")

func (r *PostgresRepository) SearchReports(ctx context.Context, req AdminReportListRequest) (adminReportPage, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return adminReportPage{}, persistence.ErrNotImplemented
	}
	cursorRaw := types.PaginationCursor(req.Pagination)
	cursor, apiErr := decodeAdminReportCursor(cursorRaw)
	if apiErr != nil {
		return adminReportPage{}, apiErr
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	// keyset 谓词与 LIMIT 下推 SQL（审查 P1-1），limit+1 探测下一页。
	var cursorAt, cursorID any
	if cursorRaw != "" {
		cursorAt, cursorID = cursor.CreatedAt.UTC(), cursor.ReportID
	}
	rows, err := queryable.Query(ctx, queryAdminSearchReports, string(req.Status), string(req.TargetType), cursorAt, cursorID, limit+1)
	if err != nil {
		return adminReportPage{}, err
	}
	defer rows.Close()
	items := make([]AdminReportItem, 0, limit+1)
	rawCreatedAt := make([]time.Time, 0, limit+1)
	for rows.Next() {
		record, err := scanAdminReportRow(rows, false)
		if err != nil {
			return adminReportPage{}, err
		}
		items = append(items, record.Item)
		rawCreatedAt = append(rawCreatedAt, record.CreatedAtRaw)
	}
	if err := rows.Err(); err != nil {
		return adminReportPage{}, err
	}
	page := adminReportPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.Next = &adminReportCursor{CreatedAt: rawCreatedAt[limit-1], ReportID: last.ReportID}
	}
	// Total = 过滤后、分页前的命中总数（「待处理」角标，spec AdminReportsPage.
	// total）。仅 service 会透出的场景（带 status 过滤）才跑独立 COUNT，避免
	// 每次翻页都全表计数。
	if req.Status != "" {
		var total int
		if err := queryable.QueryRow(ctx, queryAdminCountReports, string(req.Status), string(req.TargetType)).Scan(&total); err != nil {
			return adminReportPage{}, err
		}
		page.Total = total
	}
	return page, nil
}

func (r *PostgresRepository) GetReportHandleRecord(ctx context.Context, reportID string) (reportHandleRecord, bool, error) {
	queryable, ok := r.queryable(ctx)
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
	queryable, ok := r.queryable(ctx)
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
	// pending 守卫（见 queryAdminHandleReport）：并发裁决只有一方命中；0 行
	// 返回「已处理」，防止双重执行解散/封禁等副作用。
	var updated string
	if err := queryable.QueryRow(ctx, queryAdminHandleReport, reportID, string(status), now, handledByID, nullableString(strings.TrimSpace(note))).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminReportItem{}, false, errReportAlreadyHandled
		}
		return AdminReportItem{}, false, err
	}
	record.Item.Status = status
	handledAt := now.Format(time.RFC3339)
	record.Item.HandledAt = &handledAt
	record.Item.HandleNote = types.StrPtr(strings.TrimSpace(note))
	// 裁决后同步刷新 resolution（spec api_spec_types.md ReportItem.resolution）。
	if status == ReportStatusResolved || status == ReportStatusDismissed {
		action := reportsmodule.ReportActionDismiss
		if status == ReportStatusResolved {
			action = reportsmodule.ReportActionResolve
		}
		note := strings.TrimSpace(note)
		record.Item.Resolution = &reportsmodule.ReportResolution{Action: action, ResolvedAt: handledAt, Note: types.StrPtr(note)}
	}
	if handledBy != nil {
		hb := *handledBy
		record.Item.HandledBy = &hb
	}
	return record.Item, true, nil
}
