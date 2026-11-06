package moments

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

// HasTagAssignments 返回 tag_name 是否仍被任一瞬间引用（未被引用的标签在管理端
// 属「不存在」；含已软删瞬间的标签关联仍保留，与硬删除语义一致）。
func (r *PostgresRepository) HasTagAssignments(ctx context.Context, tagName string) (bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var exists bool
	if err := queryable.QueryRow(ctx, queryMomentTagAssignmentsExist, tagName).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// DeleteTagAssignments 物理删除该标签的全部瞬间关联（管理端 hard_delete）。
func (r *PostgresRepository) DeleteTagAssignments(ctx context.Context, tagName string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteMomentTagAssignments, tagName)
}
