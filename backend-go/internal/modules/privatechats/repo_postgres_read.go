package privatechats

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string) map[string]time.Time {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.List(ctx, userID)
	}
	rows, err := queryable.Query(ctx, queryListPrivateChats, userID)
	if err != nil {
		return r.fallback.List(ctx, userID)
	}
	defer rows.Close()
	items := map[string]time.Time{}
	for rows.Next() {
		var chatKey string
		var createdAt time.Time
		if err := rows.Scan(&chatKey, &createdAt); err != nil {
			return r.fallback.List(ctx, userID)
		}
		items[chatKey] = createdAt
	}
	if len(items) == 0 {
		return r.fallback.List(ctx, userID)
	}
	return items
}
