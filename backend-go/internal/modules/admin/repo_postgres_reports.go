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
	page := adminReportPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.Next = &adminReportCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), ReportID: last.ReportID}
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

// 撤回按消息表拆两条 SQL 常量（hit=false 表示该表无此消息或已撤回）。
// Queryable.Exec 不返回受影响行数，用 UPDATE...RETURNING + EXISTS 计数，
// 模式同 SweepRetention。
const (
	queryRecallGroupMessage   = "WITH upd AS (UPDATE group_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE RETURNING 1) SELECT EXISTS (SELECT 1 FROM upd)"
	queryRecallPrivateMessage = "WITH upd AS (UPDATE private_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE RETURNING 1) SELECT EXISTS (SELECT 1 FROM upd)"
)

func (r *PostgresRepository) RecallGroupMessage(ctx context.Context, messageID string) (bool, error) {
	return r.recallMessage(ctx, queryRecallGroupMessage, messageID)
}

func (r *PostgresRepository) RecallPrivateMessage(ctx context.Context, messageID string) (bool, error) {
	return r.recallMessage(ctx, queryRecallPrivateMessage, messageID)
}

func (r *PostgresRepository) recallMessage(ctx context.Context, query, messageID string) (bool, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var hit bool
	if err := queryable.QueryRow(ctx, query, messageID).Scan(&hit); err != nil {
		return false, err
	}
	return hit, nil
}

// DeleteMoment 裁决执行的内容删除：与 moments 模块自身删除路径一致（spec
// api_spec_moment.md「删除语义」：软删除——标记 is_deleted=true、description
// 清空，评论/反应数据保留，不物理删除）。
const queryAdminSoftDeleteMoment = `
UPDATE moments
SET is_deleted = TRUE, description = ''
WHERE moment_id = $1`

// DeleteComment 与 comments 模块自身删除路径一致（spec api_spec_comment.md：
// 软删除——is_deleted=true、content 清空；回复删除同时递减父评论 replies_count）。
const queryAdminSoftDeleteComment = `
WITH deleted AS (
	UPDATE comments SET is_deleted = TRUE, content = '' WHERE comment_id = $1
	RETURNING parent_id
)
UPDATE comments SET replies_count = GREATEST(replies_count - 1, 0)
WHERE comment_id IN (SELECT parent_id FROM deleted WHERE parent_id IS NOT NULL AND parent_id != '')`

func (r *PostgresRepository) DeleteMoment(ctx context.Context, momentID string) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryAdminSoftDeleteMoment, momentID)
}

func (r *PostgresRepository) DeleteComment(ctx context.Context, commentID string) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryAdminSoftDeleteComment, commentID)
}
