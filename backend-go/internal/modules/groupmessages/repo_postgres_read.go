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
	// fileIDsByIndex 与 items 同步索引：媒体元数据循环后单次批量回填，
	// 避免逐条查询（N+1）。语义与 WS media 负载一致（url=storage_key）。
	fileIDsByIndex := make([]string, 0, limit+1)
	fileIDs := make([]string, 0, limit+1)
	messageIDs := make([]string, 0, limit+1)
	for rows.Next() {
		var item GroupMessageItem
		var editedAt string
		var recalledAt string
		var senderUserID string
		var senderNickname string
		var senderAvatar string
		var content string
		var recalledBy, recalledByNickname, recalledByAvatar string
		var mentions []string
		var fileID, shareCode string
		var replyMessageID, replyType, replyContent, replyShareCode string
		var replySenderID, replyNickname, replyAvatar, replySentAt string
		if err := rows.Scan(&item.MessageID, &item.Type, &content, &item.SentAt, &editedAt, &recalledAt, &fileID, &shareCode, &replyMessageID, &replyType, &replyContent, &replyShareCode, &replySenderID, &replyNickname, &replyAvatar, &replySentAt, &senderUserID, &senderNickname, &senderAvatar, &recalledBy, &recalledByNickname, &recalledByAvatar, &mentions); err != nil {
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
		item.Sender = map[string]any{"user_id": senderUserID, "nickname": senderNickname, "avatar_url": senderAvatar}
		// mentions 恒在（spec：空数组 = 无 @）；历史行 DEFAULT '{}' 兜底 nil。
		if mentions == nil {
			item.Mentions = []string{}
		} else {
			item.Mentions = mentions
		}
		if recalledBy != "" {
			item.RecalledBy = map[string]any{"user_id": recalledBy, "nickname": recalledByNickname, "avatar_url": recalledByAvatar}
		}
		items = append(items, item)
		fileIDsByIndex = append(fileIDsByIndex, fileID)
		if fileID != "" {
			fileIDs = append(fileIDs, fileID)
		}
		messageIDs = append(messageIDs, item.MessageID)
	}
	if err := rows.Err(); err != nil {
		return PaginatedGroupMessages{}, err
	}
	// 已读计数页面级一次聚合回填（取代逐行 LATERAL，见 queryGroupMessagesHistory 注释）。
	// 已撤回消息 read_count 恒 null（spec：null = 不适用，含已撤回消息）。
	if readCounts, err := r.readCountsByMessageIDs(ctx, queryable, groupID, messageIDs); err != nil {
		return PaginatedGroupMessages{}, err
	} else {
		for i := range items {
			if items[i].RecalledAt != nil {
				continue
			}
			if rc, ok := readCounts[items[i].MessageID]; ok {
				count := rc
				items[i].ReadCount = &count
			}
		}
	}
	if len(fileIDs) > 0 {
		mediaByFileID, err := r.mediaByFileIDs(ctx, queryable, fileIDs)
		if err != nil {
			return PaginatedGroupMessages{}, err
		}
		for i := range items {
			if media, ok := mediaByFileID[fileIDsByIndex[i]]; ok {
				items[i].Media = media
			}
		}
	}
	if len(items) <= limit {
		return PaginatedGroupMessages{Items: items}, nil
	}
	next := items[limit-1]
	c := encodeGroupMessageCursor(next.SentAt, next.MessageID)
	return PaginatedGroupMessages{Items: items[:limit], NextCursor: &c}, nil
}

// readCountsByMessageIDs 页面级批量已读计数（单次往返，口径与逐条版一致：
// last_read_sent_at >= 消息 sent_at 且成员 joined_at <= sent_at）。缺失的消息
// 不在返回 map 中（调用方按 0 处理）。
func (r *PostgresRepository) readCountsByMessageIDs(ctx context.Context, queryable persistence.Queryable, groupID string, messageIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(messageIDs))
	if len(messageIDs) == 0 {
		return result, nil
	}
	rows, err := queryable.Query(ctx, queryGroupMessageReadCounts, groupID, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var messageID string
		var count int
		if err := rows.Scan(&messageID, &count); err != nil {
			return nil, err
		}
		result[messageID] = count
	}
	return result, rows.Err()
}

// mediaByFileIDs 批量取消息媒体元数据（单次往返），键集合与 spec MessageMedia
// 一致（8 键恒定 nullable：url/thumbnail_url/width/height/mime_type/file_name/
// file_size/duration）。url=storage_key 与 WS media 负载同语义。仅回填
// status='completed' 的文件；未命中的 file_id（历史脏数据）保持 media 为空，
// 由客户端占位兜底。
func (r *PostgresRepository) mediaByFileIDs(ctx context.Context, queryable persistence.Queryable, fileIDs []string) (map[string]map[string]any, error) {
	rows, err := queryable.Query(ctx, queryGroupMediaFilesByIDs, fileIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mediaByFileID := make(map[string]map[string]any, len(fileIDs))
	for rows.Next() {
		var fileID, storageKey, fileName, mimeType, thumbnailURL string
		var fileSize int
		var width, height, duration *int
		if err := rows.Scan(&fileID, &storageKey, &fileName, &fileSize, &mimeType, &thumbnailURL, &width, &height, &duration); err != nil {
			return nil, err
		}
		mediaByFileID[fileID] = map[string]any{
			"url":           storageKey,
			"thumbnail_url": types.StrPtr(thumbnailURL),
			"width":         width,
			"height":        height,
			"mime_type":     types.StrPtr(mimeType),
			"file_name":     types.StrPtr(fileName),
			"file_size":     fileSize,
			"duration":      duration,
		}
	}
	return mediaByFileID, rows.Err()
}
