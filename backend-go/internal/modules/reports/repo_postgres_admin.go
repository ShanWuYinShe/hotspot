package reports

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

// MarkHandledAsAdmin 管理端裁决举报：pending 守卫 + RETURNING（并发裁决只有
// 一方命中）；返回 false 表示举报不存在或已被处理。
func (r *PostgresRepository) MarkHandledAsAdmin(ctx context.Context, reportID, status string, handledAt time.Time, handledByID *string, note *string) (bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var updatedID string
	if err := queryable.QueryRow(ctx, queryMarkReportHandledAsAdmin, reportID, status, handledAt.UTC(), handledByID, note).Scan(&updatedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteHandledBefore 物理删除已处理终态且 handled_at 早于 cutoff 的举报
// （pending 永不自动清理），单次最多 limit 行；调用方循环分批直到删空或达到
// 单轮上限，避免无界 DELETE 长时间持锁。
func (r *PostgresRepository) DeleteHandledBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var deleted int64
	if err := queryable.QueryRow(ctx, queryDeleteHandledReportsBefore, cutoff.UTC(), limit).Scan(&deleted); err != nil {
		return 0, err
	}
	return deleted, nil
}
