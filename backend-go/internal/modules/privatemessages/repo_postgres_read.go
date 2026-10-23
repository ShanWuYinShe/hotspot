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
	// limit 下推 SQL（审计 perf H-2）：此前恒取默认页（21 行）再内存截断，
	// Last（limit=1）每会话白读 20 行完整消息并在 List 中被逐会话放大。
	var pagination *PaginationParams
	if limit > 0 {
		pagination = &PaginationParams{Limit: limit}
	}
	return r.historySince(ctx, userID, targetUserID, pagination, deletedAfter)
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

// LastByChatKeys 批量取每个会话最近一条（单条 DISTINCT ON 往返）。
// boundaries 缺失的 key 以 NULL 下推（不限可见性下界）。
func (r *PostgresRepository) LastByChatKeys(ctx context.Context, userID string, chatKeys []string, boundaries map[string]*time.Time) (map[string]PrivateMessageItem, error) {
	if len(chatKeys) == 0 {
		return map[string]PrivateMessageItem{}, nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	// sinceValues 用 []*time.Time 类型化下推（nil → NULL）：[]any 混合 nil 时
	// pgx 无法推断 timestamptz[] 元素类型。
	sinceValues := make([]*time.Time, len(chatKeys))
	for i, key := range chatKeys {
		if boundary, ok := boundaries[key]; ok && boundary != nil {
			utc := boundary.UTC()
			sinceValues[i] = &utc
		}
	}
	rows, err := queryable.Query(ctx, queryLastPrivateMessagesByChatKeys, chatKeys, sinceValues)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]PrivateMessageItem, len(chatKeys))
	// fileIDByChatKey 供循环后单次批量回填媒体元数据（审查 P1-7）。
	fileIDByChatKey := make(map[string]string, len(chatKeys))
	for rows.Next() {
		var chatKey string
		var item PrivateMessageItem
		var fileID string
		var senderNickname string
		var senderAvatar string
		if err := rows.Scan(&chatKey, &item.MessageID, &item.Type, &item.Content, &fileID, &item.ShareCode, &item.Sender.UserID, &senderNickname, &senderAvatar, &item.SentAt); err != nil {
			return nil, err
		}
		item.Sender.Nickname = senderNickname
		item.Sender.AvatarURL = types.StrPtr(senderAvatar)
		if fileID != "" {
			fileIDByChatKey[chatKey] = fileID
		}
		result[chatKey] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 批量回填（url=storage_key + 完整契约字段，对齐 WS/群聊口径；未命中
	// 的 file_id 为历史脏数据，保持 media 为 nil）。
	if len(fileIDByChatKey) > 0 {
		fileIDs := make([]string, 0, len(fileIDByChatKey))
		for _, fileID := range fileIDByChatKey {
			fileIDs = append(fileIDs, fileID)
		}
		mediaByFileID, err := r.mediaByFileIDs(ctx, queryable, fileIDs)
		if err != nil {
			return nil, err
		}
		for chatKey, fileID := range fileIDByChatKey {
			if media, ok := mediaByFileID[fileID]; ok {
				item := result[chatKey]
				item.Media = &media
				result[chatKey] = item
			}
		}
	}
	return result, nil
}

// mediaByFileIDs 批量取私聊消息媒体元数据（单次往返，queryMediaFilesByIDs）：
// 键集合与 spec MessageMedia 8 字段一致、url=storage_key——对齐 groupmessages
// 侧同款实现与私聊 WS 负载口径。此前 HTTP 历史/会话列表直填裸 file_id 且缺
// 7 个契约字段（审查 P1-7）。
func (r *PostgresRepository) mediaByFileIDs(ctx context.Context, queryable persistence.Queryable, fileIDs []string) (map[string]MessageMedia, error) {
	rows, err := queryable.Query(ctx, queryMediaFilesByIDs, fileIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mediaByFileID := make(map[string]MessageMedia, len(fileIDs))
	for rows.Next() {
		var fileID, storageKey, fileName, mimeType, thumbnailURL string
		var fileSize int
		var width, height, duration *int
		if err := rows.Scan(&fileID, &storageKey, &fileName, &fileSize, &mimeType, &thumbnailURL, &width, &height, &duration); err != nil {
			return nil, err
		}
		mediaByFileID[fileID] = MessageMedia{
			URL:          storageKey,
			ThumbnailURL: types.StrPtr(thumbnailURL),
			FileName:     types.StrPtr(fileName),
			FileSize:     &fileSize,
			Duration:     duration,
			Width:        width,
			Height:       height,
			MimeType:     types.StrPtr(mimeType),
		}
	}
	return mediaByFileID, rows.Err()
}

// UnreadCountsByChatKeys 批量取各会话未读数（单条 GROUP BY 往返）。
func (r *PostgresRepository) UnreadCountsByChatKeys(ctx context.Context, userID string, chatKeys []string, boundaries map[string]*time.Time) (map[string]int, error) {
	if len(chatKeys) == 0 {
		return map[string]int{}, nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	sinceValues := make([]*time.Time, len(chatKeys))
	for i, key := range chatKeys {
		if boundary, ok := boundaries[key]; ok && boundary != nil {
			utc := boundary.UTC()
			sinceValues[i] = &utc
		}
	}
	rows, err := queryable.Query(ctx, queryUnreadPrivateMessageCountsByChatKeys, chatKeys, userID, sinceValues)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int, len(chatKeys))
	for rows.Next() {
		var chatKey string
		var count int
		if err := rows.Scan(&chatKey, &count); err != nil {
			return nil, err
		}
		result[chatKey] = count
	}
	return result, rows.Err()
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
	// fileIDsByIndex 与 items 同步索引：媒体元数据循环后单次批量回填（审查
	// P1-7），避免逐条查询；语义与 WS media 负载一致（url=storage_key，8 键
	// 恒定 nullable）。
	fileIDsByIndex := make([]string, 0, limit+1)
	fileIDs := make([]string, 0, limit+1)
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
		fileIDsByIndex = append(fileIDsByIndex, fileID)
		if fileID != "" {
			fileIDs = append(fileIDs, fileID)
		}
		if replyMessageID != "" {
			item.ReplyTo = &MessageBrief{MessageID: replyMessageID, Type: replyType, Content: types.StrPtr(replyContent), ShareCode: types.StrPtr(replyShareCode), Sender: UserInfo{UserID: replySenderUserID, Nickname: replySenderNickname, AvatarURL: types.StrPtr(replySenderAvatar)}, SentAt: replySentAt}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMessages{}, err
	}
	// 页面级一次批量回填媒体元数据（url=storage_key + 完整契约字段；未命中
	// 的 file_id 为历史脏数据，保持 media 为 nil 由客户端占位兜底）。
	if len(fileIDs) > 0 {
		mediaByFileID, err := r.mediaByFileIDs(ctx, queryable, fileIDs)
		if err != nil {
			return PaginatedMessages{}, err
		}
		for i := range items {
			if media, ok := mediaByFileID[fileIDsByIndex[i]]; ok {
				items[i].Media = &media
			}
		}
	}
	if len(items) <= limit {
		return PaginatedMessages{Items: items}, nil
	}
	next := items[limit-1]
	c := encodeMessageCursor(next.SentAt, next.MessageID)
	return PaginatedMessages{Items: items[:limit], NextCursor: &c}, nil
}
