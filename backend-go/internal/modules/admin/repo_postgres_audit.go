package admin

import (
	"context"
	"encoding/json"
	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/types"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) ListAuditLogs(ctx context.Context, filter AuditLogFilter, includeIPAddress bool) (auditLogPage, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return auditLogPage{}, persistence.ErrNotImplemented
	}
	cursor, apiErr := decodeAuditCursor(filter.Pagination.Cursor)
	if apiErr != nil {
		return auditLogPage{}, apiErr
	}
	limit := types.NormalizeLimit(filter.Pagination.Limit)
	// 过滤 + keyset + LIMIT 全部下推 SQL（审查 P1-1：此前无 WHERE 无 LIMIT，
	// 全表物化后在 Go 里过滤/翻页——审计表随管理操作只增不减，列表页
	// O(全表)）。时间过滤与游标谓词口径与原 Go 实现一致（start/end 闭区间）。
	var cursorAt, cursorLogID any
	if filter.Pagination.Cursor != "" {
		cursorAt, cursorLogID = cursor.CreatedAt.UTC(), cursor.LogID
	}
	var startTime, endTime any
	if filter.StartTime != nil {
		startTime = filter.StartTime.UTC()
	}
	if filter.EndTime != nil {
		endTime = filter.EndTime.UTC()
	}
	rows, err := queryable.Query(ctx, queryAuditList,
		string(filter.Action), filter.OperatorID, string(filter.TargetType), filter.TargetID,
		startTime, endTime, cursorAt, cursorLogID, limit+1)
	if err != nil {
		return auditLogPage{}, err
	}
	defer rows.Close()
	items := make([]AuditLogItem, 0, limit+1)
	for rows.Next() {
		entry, err := scanAuditEntry(rows)
		if err != nil {
			return auditLogPage{}, err
		}
		items = append(items, auditLogItemFromEntry(entry, includeIPAddress))
	}
	if err := rows.Err(); err != nil {
		return auditLogPage{}, err
	}
	page := auditLogPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.Next = &auditCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), LogID: last.LogID}
	}
	return page, nil
}

func (r *PostgresRepository) AppendAuditLog(ctx context.Context, entry AuditLogEntry) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if entry.LogID == "" {
		entry.LogID = idgen.MustWithPrefix("audit_")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	detailsJSON := "{}"
	if len(entry.Details) > 0 {
		blob, err := json.Marshal(entry.Details)
		if err != nil {
			return err
		}
		detailsJSON = string(blob)
	}
	return queryable.Exec(ctx, queryAuditInsert,
		entry.LogID,
		string(entry.Action),
		entry.Operator.UserID,
		entry.Operator.Nickname,
		nullableString(entry.Operator.AvatarURL),
		string(entry.TargetType),
		entry.TargetID,
		nullableString(entry.TargetName),
		detailsJSON,
		nullableString(entry.IPLocation),
		nullableString(entry.IPAddress),
		nullableString(entry.DeviceSummary),
		entry.CreatedAt.UTC(),
	)
}
