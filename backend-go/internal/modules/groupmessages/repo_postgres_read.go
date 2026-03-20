package groupmessages

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) History(ctx context.Context, groupID string) PaginatedGroupMessages {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.History(ctx, groupID)
	}
	rows, err := queryable.Query(ctx, queryGroupMessagesHistory, groupID)
	if err != nil {
		return r.fallback.History(ctx, groupID)
	}
	defer rows.Close()
	items := []GroupMessageItem{}
	for rows.Next() {
		var item GroupMessageItem
		if err := rows.Scan(&item.MessageID, &item.Type, &item.Content, &item.SentAt); err != nil {
			return r.fallback.History(ctx, groupID)
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return r.fallback.History(ctx, groupID)
	}
	return PaginatedGroupMessages{Items: items}
}
