package reports

import (
	"context"
	"encoding/json"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// 举报列表游标命名空间（platform/cursor AES-GCM 加密，跨列表游标不可互用）。
const reportListCursorNamespace = "reports.mine"

// reportListCursor 是 keyset 游标载荷：(created_at, report_id) 稳定元组
// （spec api_spec_report.md：created_at DESC + report_id DESC tie-breaker）。
type reportListCursor struct {
	At string `json:"at"` // RFC3339 UTC
	ID string `json:"id"`
}

func encodeReportListCursor(createdAt time.Time, reportID string) string {
	value, err := cursor.Encode(reportListCursorNamespace, reportListCursor{At: createdAt.UTC().Format(time.RFC3339Nano), ID: reportID})
	if err != nil {
		return ""
	}
	return value
}

func decodeReportListCursor(raw string) (time.Time, string, *httpx.APIError) {
	if raw == "" {
		return time.Time{}, "", nil
	}
	var decoded reportListCursor
	if err := cursor.Decode(reportListCursorNamespace, raw, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return time.Time{}, "", httpx.InvalidRequest("pagination.cursor 无效")
	}
	parsedAt, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return time.Time{}, "", httpx.InvalidRequest("pagination.cursor 无效")
	}
	return parsedAt.UTC(), decoded.ID, nil
}

func (r *PostgresRepository) List(ctx context.Context, userID string, req ListReportsRequest) (PaginatedReports, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedReports{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedReports{}, persistence.ErrNotImplemented
	}
	// limit 归一化（api_spec_base.md 分页：default=20 max=100），多取 1 行探测下一页。
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	cursorAt, cursorID, apiErr := decodeReportListCursor(types.PaginationCursor(req.Pagination))
	if apiErr != nil {
		return PaginatedReports{}, apiErr
	}
	var queryCursorAt any
	var queryCursorID any
	if !cursorAt.IsZero() {
		queryCursorAt = cursorAt
		queryCursorID = cursorID
	}
	rows, err := queryable.Query(ctx, queryListReports, userID, string(req.Status), string(req.TargetType), queryCursorAt, queryCursorID, limit+1)
	if err != nil {
		return PaginatedReports{}, err
	}
	defer rows.Close()
	items := make([]ReportItem, 0, limit+1)
	for rows.Next() {
		item, err := scanReportRow(rows)
		if err != nil {
			return PaginatedReports{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedReports{}, err
	}
	page := PaginatedReports{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		next := encodeReportListCursor(timeFromRFC3339(last.CreatedAt), last.ReportID)
		if next != "" {
			page.NextCursor = &next
		}
	}
	return page, nil
}

func scanReportRow(rows persistence.Rows) (ReportItem, error) {
	var item ReportItem
	var snapshotJSON *string
	var createdAt time.Time
	var handledAt *time.Time
	var handleNote string
	if err := rows.Scan(&item.ReportID, &item.TargetType, &item.TargetID, &item.Reason, &item.Description, &item.Status, &createdAt, &snapshotJSON, &handledAt, &handleNote); err != nil {
		return ReportItem{}, err
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if snapshotJSON != nil && *snapshotJSON != "" {
		var snapshot ReportSnapshot
		if err := json.Unmarshal([]byte(*snapshotJSON), &snapshot); err != nil {
			return ReportItem{}, err
		}
		item.MessageSnapshot = &snapshot
	}
	applyReportResolution(&item, handledAt, handleNote)
	return item, nil
}

// applyReportResolution 组装 resolution（@owner-only，spec api_spec_types.md）：
// action 从终态 status 推断，不暴露处理管理员身份；pending/未处理为 null。
func applyReportResolution(item *ReportItem, handledAt *time.Time, handleNote string) {
	if handledAt == nil || (item.Status != ReportStatusResolved && item.Status != ReportStatusDismissed) {
		return
	}
	action := ReportActionDismiss
	if item.Status == ReportStatusResolved {
		action = ReportActionResolve
	}
	item.Resolution = &ReportResolution{
		Action:     action,
		ResolvedAt: handledAt.UTC().Format(time.RFC3339),
		Note:       types.StrPtr(handleNote),
	}
}

// timeFromRFC3339 容错解析 RFC3339 时间串（解析失败返回零值；仅用于游标键，
// DB 写入与读取侧均为 UTC RFC3339）。
func timeFromRFC3339(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
