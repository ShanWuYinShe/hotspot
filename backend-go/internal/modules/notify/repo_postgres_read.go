package notify

import (
	"context"
	"encoding/json"
	"fmt"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string, req NotificationListRequest) (PaginatedNotify, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedNotify{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedNotify{}, persistence.ErrNotImplemented
	}
	query, err := normalizeNotificationListQuery(req)
	if err != nil {
		return PaginatedNotify{}, err
	}
	var cursorCreatedAt any
	cursorNotificationID := ""
	if query.Cursor != nil {
		cursorCreatedAt = query.Cursor.CreatedAt
		cursorNotificationID = query.Cursor.NotificationID
	}
	rows, err := queryable.Query(ctx, queryListNotifications, userID, query.Type, cursorCreatedAt, cursorNotificationID, query.Limit+1)
	if err != nil {
		return PaginatedNotify{}, err
	}
	defer rows.Close()
	items := make([]NotifyItem, 0, query.Limit+1)
	for rows.Next() {
		var item NotifyItem
		var payloadJSON string
		if err := rows.Scan(&item.NotificationID, &item.Type, &item.IsRead, &item.CreatedAt, &payloadJSON); err != nil {
			return PaginatedNotify{}, err
		}
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return PaginatedNotify{}, err
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedNotify{}, err
	}
	page := PaginatedNotify{}
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
	result := UnreadCountResult{ByType: UnreadCountByType{}}
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
		switch bucket(notificationType) {
		case "social":
			result.ByType.Social += count
		case "interaction":
			result.ByType.Interaction += count
		case "group":
			result.ByType.Group += count
		case "system":
			result.ByType.System += count
		}
	}
	if err := rows.Err(); err != nil {
		return UnreadCountResult{}, err
	}
	return result, nil
}
