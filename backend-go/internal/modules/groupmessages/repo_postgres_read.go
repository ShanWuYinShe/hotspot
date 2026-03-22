package groupmessages

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) History(ctx context.Context, groupID string) (PaginatedGroupMessages, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.History(ctx, groupID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedGroupMessages{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryGroupMessagesHistory, groupID)
	if err != nil {
		return PaginatedGroupMessages{}, err
	}
	defer rows.Close()
	items := []GroupMessageItem{}
	for rows.Next() {
		var item GroupMessageItem
		if err := rows.Scan(&item.MessageID, &item.Type, &item.Content, &item.SentAt); err != nil {
			return PaginatedGroupMessages{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedGroupMessages{}, err
	}
	return PaginatedGroupMessages{Items: items}, nil
}
