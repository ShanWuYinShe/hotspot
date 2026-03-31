package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string, req NotificationListRequest) (PaginatedNotifications, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedNotifications{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedNotifications{}, persistence.ErrNotImplemented
	}
	query, err := normalizeNotificationListQuery(req)
	if err != nil {
		return PaginatedNotifications{}, err
	}
	cursorCreatedAt := ""
	cursorNotificationID := ""
	if query.Cursor != nil {
		cursorCreatedAt = query.Cursor.CreatedAt
		cursorNotificationID = query.Cursor.NotificationID
	}
	rows, err := queryable.Query(ctx, queryListNotifications, userID, query.Type, cursorCreatedAt, cursorNotificationID, query.Limit+1)
	if err != nil {
		return PaginatedNotifications{}, err
	}
	defer rows.Close()
	items := make([]NotificationItem, 0, query.Limit+1)
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
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedNotifications{}, err
	}
	page := PaginatedNotifications{}
	if len(items) > query.Limit {
		last := items[query.Limit-1]
		page.NextCursor = encodeNotificationCursor(notificationCursor{CreatedAt: last.CreatedAt, NotificationID: last.NotificationID})
		items = items[:query.Limit]
	}
	page.Items = items
	return page, nil
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, userID string) (UnreadCountResult, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return UnreadCountResult{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return UnreadCountResult{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryUnreadNotificationCounts, userID)
	if err != nil {
		return UnreadCountResult{}, err
	}
	defer rows.Close()
	result := UnreadCountResult{ByType: map[string]int{"social": 0, "interaction": 0, "group": 0, "system": 0}}
	for rows.Next() {
		var notificationType, countText string
		if err := rows.Scan(&notificationType, &countText); err != nil {
			return UnreadCountResult{}, err
		}
		count := 0
		if _, err := fmt.Sscanf(countText, "%d", &count); err != nil {
			return UnreadCountResult{}, err
		}
		result.Total += count
		result.ByType[bucket(notificationType)] += count
	}
	if err := rows.Err(); err != nil {
		return UnreadCountResult{}, err
	}
	return result, nil
}
