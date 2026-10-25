package moments

import (
	"context"
	"fmt"

	"hotspot/backend-go/internal/platform/persistence"
)

// loadMomentFavorites 批量查询当前用户在指定瞬间集合中已收藏的集合（page-only 补全）。
// 匿名视角（viewerID 为空）直接返回空集，不查库。
func loadMomentFavorites(ctx context.Context, queryable persistence.Queryable, viewerID string, momentIDs []string) (map[string]bool, error) {
	favorited := make(map[string]bool, len(momentIDs))
	if viewerID == "" || len(momentIDs) == 0 {
		return favorited, nil
	}
	rows, err := queryable.Query(ctx, queryMomentFavoritesByIDs, viewerID, momentIDs)
	if err != nil {
		return nil, fmt.Errorf("moments.loadMomentFavorites query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var momentID string
		if err := rows.Scan(&momentID); err != nil {
			return nil, fmt.Errorf("moments.loadMomentFavorites scan: %w", err)
		}
		favorited[momentID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moments.loadMomentFavorites rows: %w", err)
	}
	return favorited, nil
}
