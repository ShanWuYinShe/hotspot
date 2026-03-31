package private_messages

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) History(ctx context.Context, userID, targetUserID string) (PaginatedMessages, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedMessages{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedMessages{}, persistence.ErrNotImplemented
	}
	chatKey := conversationKey(userID, targetUserID)
	rows, err := queryable.Query(ctx, queryPrivateMessagesHistory, chatKey, userID, targetUserID)
	if err != nil {
		return PaginatedMessages{}, err
	}
	defer rows.Close()
	items := []PrivateMessageItem{}
	for rows.Next() {
		var item PrivateMessageItem
		var senderUserID string
		if err := rows.Scan(&item.MessageID, &item.Type, &item.Content, &item.ShareCode, &senderUserID, &item.SentAt, &item.IsRead); err != nil {
			return PaginatedMessages{}, err
		}
		item.Sender = UserInfo{UserID: senderUserID, Nickname: senderUserID}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMessages{}, err
	}
	return PaginatedMessages{Items: items}, nil
}

func (r *PostgresRepository) Recent(ctx context.Context, userID, targetUserID string, limit int) (PaginatedMessages, error) {
	history, err := r.History(ctx, userID, targetUserID)
	if err != nil {
		return PaginatedMessages{}, err
	}
	if len(history.Items) > limit {
		history.Items = history.Items[:limit]
	}
	return history, nil
}

func (r *PostgresRepository) Last(ctx context.Context, userID, targetUserID string) (*PrivateMessageItem, error) {
	history, err := r.History(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}
	if len(history.Items) == 0 {
		return nil, nil
	}
	return &history.Items[0], nil
}
