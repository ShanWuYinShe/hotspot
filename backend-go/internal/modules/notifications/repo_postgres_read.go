package notifications

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string, req NotificationListRequest) PaginatedNotifications {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.List(ctx, userID, req)
	}
	rows, err := queryable.Query(ctx, queryListNotifications, userID)
	if err != nil {
		return r.fallback.List(ctx, userID, req)
	}
	defer rows.Close()
	items := []NotificationItem{}
	for rows.Next() {
		var item NotificationItem
		if err := rows.Scan(&item.NotificationID, &item.Type, &item.IsRead, &item.CreatedAt); err != nil {
			return r.fallback.List(ctx, userID, req)
		}
		if req.Type == "" || item.Type == req.Type {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return r.fallback.List(ctx, userID, req)
	}
	return PaginatedNotifications{Items: items}
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, userID string) UnreadCountResult {
	list := r.List(ctx, userID, NotificationListRequest{})
	if len(list.Items) == 0 {
		return r.fallback.UnreadCount(ctx, userID)
	}
	result := UnreadCountResult{ByType: map[string]int{"social": 0, "interaction": 0, "group": 0, "system": 0}}
	for _, item := range list.Items {
		if item.IsRead {
			continue
		}
		result.Total++
		result.ByType[bucket(item.Type)]++
	}
	return result
}
