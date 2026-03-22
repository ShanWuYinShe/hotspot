package privatechats

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string) (map[string]time.Time, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.List(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListPrivateChats, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]time.Time{}
	for rows.Next() {
		var chatKey string
		var createdAt time.Time
		if err := rows.Scan(&chatKey, &createdAt); err != nil {
			return nil, err
		}
		items[chatKey] = createdAt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
