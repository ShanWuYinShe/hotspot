package notifications

import (
	"context"
	"encoding/json"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string, req NotificationListRequest) (PaginatedNotifications, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.List(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedNotifications{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListNotifications, userID)
	if err != nil {
		return PaginatedNotifications{}, err
	}
	defer rows.Close()
	items := []NotificationItem{}
	for rows.Next() {
		var item NotificationItem
		var payloadJSON string
		if err := rows.Scan(&item.NotificationID, &item.Type, &item.IsRead, &item.CreatedAt, &payloadJSON); err != nil {
			return PaginatedNotifications{}, err
		}
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return PaginatedNotifications{}, err
			}
		}
		if req.Type == "" || item.Type == req.Type {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedNotifications{}, err
	}
	return PaginatedNotifications{Items: items}, nil
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, userID string) (UnreadCountResult, error) {
	list, err := r.List(ctx, userID, NotificationListRequest{})
	if err != nil {
		return UnreadCountResult{}, err
	}
	result := UnreadCountResult{ByType: map[string]int{"social": 0, "interaction": 0, "group": 0, "system": 0}}
	for _, item := range list.Items {
		if item.IsRead {
			continue
		}
		result.Total++
		result.ByType[bucket(item.Type)]++
	}
	return result, nil
}
