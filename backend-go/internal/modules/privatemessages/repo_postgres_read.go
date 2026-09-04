package privatemessages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

func (r *PostgresRepository) History(ctx context.Context, userID string, req ListPrivateMessagesRequest) (PaginatedMessages, error) {
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	// 独立 COUNT：此前复用 historySince（limit=20 的窗口内数未读），几天没
	// 上线、会话超过 20 条时未读数被系统性截断为最多 20。
	chatKey := conversationKey(userID, targetUserID)
	var deletedAfterValue any
	if deletedAfter != nil {
		deletedAfterValue = deletedAfter.UTC()
	}
	var count int
	if err := queryable.QueryRow(ctx, queryUnreadPrivateMessageCount, chatKey, userID, deletedAfterValue).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresRepository) historySince(ctx context.Context, userID, targetUserID string, pagination *PaginationParams, deletedAfter *time.Time) (PaginatedMessages, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedMessages{}, persistence.ErrNotImplemented
	}
	// limit 归一化走全后端统一入口（默认 20、上限 100，spec api_spec_base.md）
	limit := types.NormalizeLimit(types.PaginationLimit(pagination))
	cursorValue := ""
	if pagination != nil {
		cursorValue = pagination.Cursor
	}
	cursorSentAt, cursorMessageID, err := decodeMessageCursor(cursorValue)
	if err != nil {
		return PaginatedMessages{}, err
	}
	chatKey := conversationKey(userID, targetUserID)
	var cursorSentAtValue any
	if !cursorSentAt.IsZero() {
		cursorSentAtValue = cursorSentAt.Format(time.RFC3339Nano)
	}
	// deletedAfter 谓词下推 SQL（$7）：此前先 LIMIT 截断再在 Go 层跳过
	// sent_at <= deleted_after 的行，被跳过行占用页配额会导致翻页提前终止漏数据。
	var deletedAfterArg any
	if deletedAfter != nil {
		deletedAfterArg = deletedAfter.UTC()
	}
	rows, err := queryable.Query(ctx, queryPrivateMessagesHistory, chatKey, userID, targetUserID, cursorSentAtValue, cursorMessageID, limit+1, deletedAfterArg)
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
		item.Sender.Nickname = senderNickname
		item.Sender.AvatarURL = types.StrPtr(senderAvatar)
		if fileID != "" {
			item.Media = &MessageMedia{URL: fileID}
		}
		if replyMessageID != "" {
			item.ReplyTo = &MessageBrief{MessageID: replyMessageID, Type: replyType, Content: types.StrPtr(replyContent), ShareCode: types.StrPtr(replyShareCode), Sender: UserInfo{UserID: replySenderUserID, Nickname: replySenderNickname, AvatarURL: types.StrPtr(replySenderAvatar)}, SentAt: replySentAt}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMessages{}, err
	}
	if len(items) <= limit {
		return PaginatedMessages{Items: items}, nil
	}
	next := items[limit-1]
	c := encodeMessageCursor(next.SentAt, next.MessageID)
	return PaginatedMessages{Items: items[:limit], NextCursor: &c}, nil
}
