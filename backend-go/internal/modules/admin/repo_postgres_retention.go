package admin

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

// DeleteAuditLogsBefore 物理删除 created_at 早于 cutoff 的审计日志（保留期清理），
// 单次最多 limit 行；调用方循环分批直到删空或达到单轮上限。
func (r *PostgresRepository) DeleteAuditLogsBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var deleted int64
	if err := queryable.QueryRow(ctx, queryDeleteAuditLogsBefore, cutoff.UTC(), limit).Scan(&deleted); err != nil {
		return 0, err
	}
	return deleted, nil
}
