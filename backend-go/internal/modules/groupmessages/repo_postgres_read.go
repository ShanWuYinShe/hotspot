package groupmessages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

func (r *PostgresRepository) History(ctx context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedGroupMessages{}, persistence.ErrNotImplemented
	}
	// limit 归一化走全后端统一入口（默认 20、上限 100，spec api_spec_base.md）
	limit := types.NormalizeLimit(types.PaginationLimit(pagination))
	cursorValue := ""
	if pagination != nil {
		cursorValue = pagination.Cursor
	}
	cursorSentAt, cursorMessageID, err := decodeGroupMessageCursor(cursorValue)
	if err != nil {
		return PaginatedGroupMessages{}, err
	}
	var cursorSentAtValue any
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
		var content string
		var fileID, shareCode string
		var replyMessageID, replyType, replyContent, replyShareCode string
		var replySenderID, replyNickname, replyAvatar, replySentAt string
		var readCount int
		if err := rows.Scan(&item.MessageID, &item.Type, &content, &item.SentAt, &editedAt, &recalledAt, &fileID, &shareCode, &replyMessageID, &replyType, &replyContent, &replyShareCode, &replySenderID, &replyNickname, &replyAvatar, &replySentAt, &senderUserID, &senderNickname, &senderAvatar, &readCount); err != nil {
			return PaginatedGroupMessages{}, err
		}
		item.Content = types.StrPtr(content)
		if shareCode != "" {
			item.ShareCode = types.StrPtr(shareCode)
		}
		if replyMessageID != "" {
			item.ReplyTo = &MessageBrief{
				MessageID: replyMessageID,
				Type:      replyType,
				Content:   types.StrPtr(replyContent),
				ShareCode: types.StrPtr(replyShareCode),
				Sender:    types.UserInfo{UserID: replySenderID, Nickname: replyNickname, AvatarURL: types.StrPtr(replyAvatar)},
				SentAt:    replySentAt,
			}
		}
		item.EditedAt = types.StrPtr(editedAt)
		item.RecalledAt = types.StrPtr(recalledAt)
		item.ReadCount = &readCount
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
	c := encodeGroupMessageCursor(next.SentAt, next.MessageID)
	return PaginatedGroupMessages{Items: items[:limit], NextCursor: &c}, nil
}
