package groupmessages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) History(ctx context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedGroupMessages{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedGroupMessages{}, persistence.ErrNotImplemented
	}
	limit := 20
	cursorValue := ""
	if pagination != nil {
		if pagination.Limit > 0 {
			limit = pagination.Limit
		}
		cursorValue = pagination.Cursor
	}
	cursorSentAt, cursorMessageID, err := decodeGroupMessageCursor(cursorValue)
	if err != nil {
		return PaginatedGroupMessages{}, err
	}
	cursorSentAtValue := ""
	if !cursorSentAt.IsZero() {
		cursorSentAtValue = cursorSentAt.Format(time.RFC3339Nano)
	}
	rows, err := queryable.Query(ctx, queryGroupMessagesHistory, groupID, cursorSentAtValue, cursorMessageID, limit+1)
	if err != nil {
		return PaginatedGroupMessages{}, err
	}
	defer rows.Close()
	items := []GroupMessageItem{}
	for rows.Next() {
		var item GroupMessageItem
		var editedAt string
		var recalledAt string
		var senderUserID string
		var senderNickname string
		var senderAvatar string
		if err := rows.Scan(&item.MessageID, &item.Type, &item.Content, &item.SentAt, &editedAt, &recalledAt, &senderUserID, &senderNickname, &senderAvatar, &item.ReadCount); err != nil {
			return PaginatedGroupMessages{}, err
		}
		item.EditedAt = editedAt
		item.RecalledAt = recalledAt
		item.Sender = map[string]any{"user_id": senderUserID, "nickname": senderNickname, "avatar_url": senderAvatar}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedGroupMessages{}, err
	}
	if len(items) <= limit {
		return PaginatedGroupMessages{Items: items}, nil
	}
	next := items[limit-1]
	return PaginatedGroupMessages{Items: items[:limit], NextCursor: encodeGroupMessageCursor(next.SentAt, next.MessageID)}, nil
}
