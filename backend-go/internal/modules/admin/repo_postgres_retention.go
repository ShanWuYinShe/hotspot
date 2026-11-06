package admin

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

// DeleteAuditLogsBefore 物理删除 created_at 早于 cutoff 的审计日志（保留期清理）。
func (r *PostgresRepository) DeleteAuditLogsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var deleted int64
	if err := queryable.QueryRow(ctx, queryDeleteAuditLogsBefore, cutoff.UTC()).Scan(&deleted); err != nil {
		return 0, err
	}
	return deleted, nil
}
