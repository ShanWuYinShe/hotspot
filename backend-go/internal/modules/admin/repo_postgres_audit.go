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
	queryable, ok := r.queryable()
	if !ok {
		return auditLogPage{}, persistence.ErrNotImplemented
	}
	cursor, apiErr := decodeAuditCursor(filter.Pagination.Cursor)
	if apiErr != nil {
		return auditLogPage{}, apiErr
	}
	rows, err := queryable.Query(ctx, queryAuditList)
	if err != nil {
		return auditLogPage{}, err
	}
	defer rows.Close()
	limit := types.NormalizeLimit(filter.Pagination.Limit)
	page := auditLogPage{Items: make([]AuditLogItem, 0, limit)}
	seenCursor := filter.Pagination.Cursor == ""
	for rows.Next() {
		entry, err := scanAuditEntry(rows)
		if err != nil {
			return auditLogPage{}, err
		}
		if !matchAuditFilter(entry, filter) {
			continue
		}
		if !seenCursor {
			if entry.CreatedAt.After(cursor.CreatedAt) {
				continue
			}
			if entry.CreatedAt.Equal(cursor.CreatedAt) && entry.LogID >= cursor.LogID {
				continue
			}
			seenCursor = true
		}
		if len(page.Items) >= limit {
			page.Next = &auditCursor{CreatedAt: entry.CreatedAt, LogID: entry.LogID}
			break
		}
		page.Items = append(page.Items, auditLogItemFromEntry(entry, includeIPAddress))
	}
	if err := rows.Err(); err != nil {
		return auditLogPage{}, err
	}
	return page, nil
}

func (r *PostgresRepository) GetAuditLog(ctx context.Context, logID string, includeIPAddress bool) (AuditLogItem, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AuditLogItem{}, false, persistence.ErrNotImplemented
	}
	entry, found, err := r.getAuditEntryByID(ctx, queryable, logID)
	if err != nil || !found {
		return AuditLogItem{}, found, err
	}
	return auditLogItemFromEntry(entry, includeIPAddress), true, nil
}

func (r *PostgresRepository) AppendAuditLog(ctx context.Context, entry AuditLogEntry) error {
	queryable, ok := r.queryable()
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
