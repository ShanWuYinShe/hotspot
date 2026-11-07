package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string, req NotifyListRequest) (PaginatedNotify, error) {
	query, err := normalizeNotifyListQuery(req)
	if err != nil {
		return PaginatedNotify{}, err
	}
	if r.announcements == nil {
		q := query
		q.Limit = query.Limit + 1
		items, rawCreatedAt, err := r.queryPersistedNotifyItems(ctx, userID, q)
		if err != nil {
			return PaginatedNotify{}, err
		}
		return paginateNotifyItems(items, rawCreatedAt, query.Limit), nil
	}
	return r.listWithAnnouncements(ctx, userID, query)
}

// listWithAnnouncements 合并「持久化通知行」与「公告虚拟行」（system 模块注入）：
// 两个来源按同一 (created_at DESC, notify_id DESC) 全序归并，再统一分页。
// 持久化通知侧查询不受公告影响；类型过滤为 system_announcement 时跳过持久化
// 查询，其它具体类型时跳过公告来源。
func (r *PostgresRepository) listWithAnnouncements(ctx context.Context, userID string, query notifyListQuery) (PaginatedNotify, error) {
	var cursorAt *time.Time
	cursorNotifyID := ""
	if query.Cursor != nil {
		parsed, err := time.Parse(time.RFC3339Nano, query.Cursor.CreatedAt)
		if err != nil {
			return PaginatedNotify{}, ErrInvalidNotifyRequest
		}
		cursorAt = &parsed
		cursorNotifyID = query.Cursor.NotifyID
	}
	includePersisted := query.Type != "system_announcement"
	includeAnnouncements := query.Type == "" || query.Type == "system_announcement"
	items := make([]NotifyItem, 0, query.Limit+1)
	rawCreatedAt := make([]time.Time, 0, query.Limit+1)
	if includePersisted {
		q := query
		q.Limit = query.Limit + 1
		persisted, persistedRaw, err := r.queryPersistedNotifyItems(ctx, userID, q)
		if err != nil {
			return PaginatedNotify{}, err
		}
		items = append(items, persisted...)
		rawCreatedAt = append(rawCreatedAt, persistedRaw...)
	}
	if includeAnnouncements {
		cursor := systemmodule.AnnouncementNotifyCursor{NotifyID: cursorNotifyID}
		if cursorAt != nil {
			cursor.CreatedAt = *cursorAt
		}
		announcements, err := r.announcements.ListAnnouncementNotifications(ctx, userID, cursor, query.IsRead, query.Limit+1)
		if err != nil {
			return PaginatedNotify{}, err
		}
		for _, announcement := range announcements {
			createdAt := announcement.CreatedAt.UTC()
			items = append(items, NotifyItem{
				NotifyID:  systemmodule.AnnouncementNotifyID(announcement.AnnouncementID),
				Type:      "system_announcement",
				IsRead:    announcement.IsRead,
				CreatedAt: createdAt.Format(time.RFC3339),
				Payload: map[string]any{
					"type":            "system_announcement",
					"announcement_id": announcement.AnnouncementID,
					"title":           announcement.Title,
					"severity":        string(announcement.Severity),
				},
			})
			rawCreatedAt = append(rawCreatedAt, createdAt)
		}
	}
	// 同一 (created_at, notify_id) 全序归并；跨来源 id 前缀不同但字符串序一致。
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		ia, ib := order[a], order[b]
		if !rawCreatedAt[ia].Equal(rawCreatedAt[ib]) {
			return rawCreatedAt[ia].After(rawCreatedAt[ib])
		}
		return items[ia].NotifyID > items[ib].NotifyID
	})
	mergedItems := make([]NotifyItem, len(items))
	mergedRaw := make([]time.Time, len(items))
	for i, idx := range order {
		mergedItems[i] = items[idx]
		mergedRaw[i] = rawCreatedAt[idx]
	}
	return paginateNotifyItems(mergedItems, mergedRaw, query.Limit), nil
}

func paginateNotifyItems(items []NotifyItem, rawCreatedAt []time.Time, limit int) PaginatedNotify {
	page := PaginatedNotify{}
	if len(items) > limit {
		last := items[limit-1]
		// 解码侧 normalizeNotifyListQuery 直接把字符串交给 SQL 的 timestamptz 参数，
		// RFC3339Nano 与既有秒级格式均被 PG 接受（老游标仍有效）。
		c := encodeNotifyCursor(notifyCursor{CreatedAt: rawCreatedAt[limit-1].Format(time.RFC3339Nano), NotifyID: last.NotifyID})
		page.NextCursor = &c
		items = items[:limit]
	}
	page.Items = items
	return page
}

// queryPersistedNotifyItems 读取 notifications 表中分页窗口内（最多 query.Limit 条）
// 的原始行及真实时间精度；游标/类型/已读过滤与旧实现一致。
func (r *PostgresRepository) queryPersistedNotifyItems(ctx context.Context, userID string, query notifyListQuery) ([]NotifyItem, []time.Time, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, nil, persistence.ErrNotImplemented
	}
	var cursorCreatedAt any
	cursorNotifyID := ""
	if query.Cursor != nil {
		cursorCreatedAt = query.Cursor.CreatedAt
		cursorNotifyID = query.Cursor.NotifyID
	}
	rows, err := queryable.Query(ctx, queryListNotifys, userID, query.Type, query.IsRead, cursorCreatedAt, cursorNotifyID, query.Limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := make([]NotifyItem, 0, query.Limit)
	// rawCreatedAt 与 items 平行保存列的真实精度（timestamptz 微秒）：响应按秒展示，
	// 但 keyset 游标必须用真实值编码，否则同秒通知翻页会重复/丢行。
	rawCreatedAt := make([]time.Time, 0, query.Limit)
	for rows.Next() {
		var item NotifyItem
		var payloadJSON string
		// created_at 列为 TIMESTAMPTZ：pgx 简单协议文本格式下直接扫进 string 会得到
		// PostgreSQL 风格 "2006-01-02 15:04:05+00"，不符合契约「UTC ISO 8601」。
		var createdAt time.Time
		if err := rows.Scan(&item.NotifyID, &item.Type, &item.IsRead, &createdAt, &payloadJSON); err != nil {
			return nil, nil, err
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return nil, nil, err
			}
		}
		items = append(items, item)
		rawCreatedAt = append(rawCreatedAt, createdAt.UTC())
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, rawCreatedAt, nil
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, userID string) (UnreadSummaryResult, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
	if r.announcements != nil {
		count, err := r.announcements.UnreadAnnouncementCount(ctx, userID)
		if err != nil {
			return UnreadSummaryResult{}, err
		}
		result.ByType.System += count
		result.Total += count
	}
	return result, nil
}
