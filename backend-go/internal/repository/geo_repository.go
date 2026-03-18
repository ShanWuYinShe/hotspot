package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// S2StatRow 表示 s2_cell_stats_daily 中的一行聚合结果。
// 这里只包含地图模块需要的基础字段，后续可按需扩展。
type S2StatRow struct {
	Date        string
	Level       int16
	CellID      int64
	UserCount   int64
	GroupCount  int64
	MomentCount int64
}

// GeoRepository 封装与地理统计相关的持久化操作。
type GeoRepository interface {
	AggregateS2Stats(ctx context.Context, date string, level int16, cellIDs []int64) ([]S2StatRow, error)
}

type geoRepository struct {
	pool *pgxpool.Pool
}

func NewGeoRepository(pool *pgxpool.Pool) GeoRepository {
	return &geoRepository{pool: pool}
}

// AggregateS2Stats 从 s2_cell_stats_daily 中读取指定日期、level 和 cellID 列表的聚合结果。
// 约定：
//   - date 为 UTC 日期字符串（例如 "2026-03-18"）；
//   - level 为目标 S2 层级；
//   - cellIDs 为 S2CellID 的 int64 表示列表。
func (r *geoRepository) AggregateS2Stats(ctx context.Context, date string, level int16, cellIDs []int64) ([]S2StatRow, error) {
	if len(cellIDs) == 0 {
		return nil, nil
	}

	const query = `
SELECT date, level, cell_id, user_count, group_count, moment_count
FROM s2_cell_stats_daily
WHERE date = $1 AND level = $2 AND cell_id = ANY($3)
`

	rows, err := r.pool.Query(ctx, query, date, level, cellIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []S2StatRow
	for rows.Next() {
		var row S2StatRow
		if err := rows.Scan(&row.Date, &row.Level, &row.CellID, &row.UserCount, &row.GroupCount, &row.MomentCount); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
