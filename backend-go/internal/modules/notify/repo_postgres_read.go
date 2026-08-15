package notify

import (
	"context"
	"encoding/json"
	"fmt"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string, req NotifyListRequest) (PaginatedNotify, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedNotify{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedNotify{}, persistence.ErrNotImplemented
	}
	query, err := normalizeNotifyListQuery(req)
	if err != nil {
		return PaginatedNotify{}, err
	}
	var cursorCreatedAt any
	cursorNotifyID := ""
	if query.Cursor != nil {
		cursorCreatedAt = query.Cursor.CreatedAt
		cursorNotifyID = query.Cursor.NotifyID
	}
	rows, err := queryable.Query(ctx, queryListNotifys, userID, query.Type, query.IsRead, cursorCreatedAt, cursorNotifyID, query.Limit+1)
	if err != nil {
		return PaginatedNotify{}, err
	}
	defer rows.Close()
	items := make([]NotifyItem, 0, query.Limit+1)
	for rows.Next() {
		var item NotifyItem
		var payloadJSON string
		if err := rows.Scan(&item.NotifyID, &item.Type, &item.IsRead, &item.CreatedAt, &payloadJSON); err != nil {
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
		c := encodeNotifyCursor(notifyCursor{CreatedAt: last.CreatedAt, NotifyID: last.NotifyID})
		page.NextCursor = &c
		items = items[:query.Limit]
	}
	page.Items = items
	return page, nil
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, userID string) (UnreadSummaryResult, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return UnreadSummaryResult{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return UnreadSummaryResult{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryUnreadNotifyCounts, userID)
	if err != nil {
		return UnreadSummaryResult{}, err
	}
	defer rows.Close()
	result := UnreadSummaryResult{ByType: UnreadCountByType{}}
	for rows.Next() {
		var notifyType, countText string
		if err := rows.Scan(&notifyType, &countText); err != nil {
			return UnreadSummaryResult{}, err
		}
		count := 0
		if _, err := fmt.Sscanf(countText, "%d", &count); err != nil {
			return UnreadSummaryResult{}, err
		}
		result.Total += count
		switch bucket(notifyType) {
		case "social":
			result.ByType.Social += count
		case "interaction":
			result.ByType.Interaction += count
		case "group":
			result.ByType.Group += count
		case "report":
			result.ByType.Report += count
		case "nearby":
			result.ByType.Nearby += count
		case "system":
			result.ByType.System += count
		}
	}
	if err := rows.Err(); err != nil {
		return UnreadSummaryResult{}, err
	}
	return result, nil
}
