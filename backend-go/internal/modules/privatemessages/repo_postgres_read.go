package private_messages

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) History(ctx context.Context, userID, targetUserID string) PaginatedMessages {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.History(ctx, userID, targetUserID)
	}
	chatKey := conversationKey(userID, targetUserID)
	rows, err := queryable.Query(ctx, queryPrivateMessagesHistory, chatKey)
	if err != nil {
		return r.fallback.History(ctx, userID, targetUserID)
	}
	defer rows.Close()
	items := []PrivateMessageItem{}
	for rows.Next() {
		var item PrivateMessageItem
		var senderUserID string
		if err := rows.Scan(&item.MessageID, &item.Type, &item.Content, &item.ShareCode, &senderUserID, &item.SentAt); err != nil {
			return r.fallback.History(ctx, userID, targetUserID)
		}
		item.Sender = UserInfo{UserID: senderUserID, Nickname: senderUserID}
		items = append(items, item)
	}
	if len(items) == 0 {
		return r.fallback.History(ctx, userID, targetUserID)
	}
	return PaginatedMessages{Items: items}
}

func (r *PostgresRepository) Recent(ctx context.Context, userID, targetUserID string, limit int) PaginatedMessages {
	history := r.History(ctx, userID, targetUserID)
	if len(history.Items) > limit {
		history.Items = history.Items[:limit]
	}
	return history
}

func (r *PostgresRepository) Last(ctx context.Context, userID, targetUserID string) *PrivateMessageItem {
	history := r.History(ctx, userID, targetUserID)
	if len(history.Items) == 0 {
		return nil
	}
	return &history.Items[0]
}
