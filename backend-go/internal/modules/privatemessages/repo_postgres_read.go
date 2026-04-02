package private_messages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) History(ctx context.Context, userID string, req GetHistoryRequest) (PaginatedMessages, error) {
	return r.historySince(ctx, userID, req.TargetUserID, req.Pagination, nil)
}

func (r *PostgresRepository) Recent(ctx context.Context, userID, targetUserID string, limit int, deletedAfter *time.Time) (PaginatedMessages, error) {
	history, err := r.historySince(ctx, userID, targetUserID, nil, deletedAfter)
	if err != nil {
		return PaginatedMessages{}, err
	}
	if limit > 0 && len(history.Items) > limit {
		history.Items = history.Items[:limit]
	}
	return history, nil
}

func (r *PostgresRepository) Last(ctx context.Context, userID, targetUserID string, deletedAfter *time.Time) (*PrivateMessageItem, error) {
	history, err := r.Recent(ctx, userID, targetUserID, 1, deletedAfter)
	if err != nil {
		return nil, err
	}
	if len(history.Items) == 0 {
		return nil, nil
	}
	return &history.Items[0], nil
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, userID, targetUserID string, deletedAfter *time.Time) (int, error) {
	history, err := r.historySince(ctx, targetUserID, userID, nil, deletedAfter)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range history.Items {
		if item.Sender.UserID != userID && !item.IsRead {
			count++
		}
	}
	return count, nil
}

func (r *PostgresRepository) historySince(ctx context.Context, userID, targetUserID string, pagination *PaginationParams, deletedAfter *time.Time) (PaginatedMessages, error) {
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
		var fileID string
		var senderNickname string
		var senderAvatar string
		var replyMessageID string
		var replyType string
		var replyContent string
		var replyShareCode string
		var replySenderUserID string
		var replySenderNickname string
		var replySenderAvatar string
		var replySentAt string
		if err := rows.Scan(&item.MessageID, &item.Type, &item.Content, &fileID, &item.ShareCode, &item.Sender.UserID, &senderNickname, &senderAvatar, &replyMessageID, &replyType, &replyContent, &replyShareCode, &replySenderUserID, &replySenderNickname, &replySenderAvatar, &replySentAt, &item.SentAt, &item.EditedAt, &item.RecalledAt, &item.IsRead); err != nil {
			return PaginatedMessages{}, err
		}
		if deletedAfter != nil {
			sentAt, err := time.Parse(time.RFC3339Nano, item.SentAt)
			if err == nil && !sentAt.After(*deletedAfter) {
				continue
			}
		}
		item.Sender.Nickname = senderNickname
		item.Sender.AvatarURL = senderAvatar
		if fileID != "" {
			item.Media = &MessageMedia{URL: fileID}
		}
		if replyMessageID != "" {
			item.ReplyTo = &MessageReplyTo{MessageID: replyMessageID, Type: replyType, Content: replyContent, ShareCode: replyShareCode, Sender: UserInfo{UserID: replySenderUserID, Nickname: replySenderNickname, AvatarURL: replySenderAvatar}, SentAt: replySentAt}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMessages{}, err
	}
	limit := 20
	if pagination != nil && pagination.Limit > 0 {
		limit = pagination.Limit
	}
	cursor := ""
	if pagination != nil {
		cursor = pagination.Cursor
	}
	if cursor != "" {
		start := 0
		for start < len(items) && items[start].MessageID != cursor {
			start++
		}
		if start == len(items) {
			return PaginatedMessages{Items: []PrivateMessageItem{}}, nil
		}
		items = items[start+1:]
	}
	if len(items) <= limit {
		return PaginatedMessages{Items: items}, nil
	}
	return PaginatedMessages{Items: items[:limit], NextCursor: items[limit-1].MessageID}, nil
}
