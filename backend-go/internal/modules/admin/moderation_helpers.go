package admin

import (
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

type tagCursor struct {
	UsageCount int       `json:"usage_count"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

type announcementCursor struct {
	CreatedAt      time.Time `json:"created_at"`
	AnnouncementID string    `json:"announcement_id"`
}

// moderationTagCursorNamespace / moderationAnnouncementCursorNamespace 是
// 标签搜索与公告管理游标的加密命名空间。
const (
	moderationTagCursorNamespace          = "admin.tags"
	moderationAnnouncementCursorNamespace = "admin.announcements"
)

func decodeTagCursor(raw string) (tagCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return tagCursor{}, nil
	}
	c, apiErr := decodeCursor[tagCursor](moderationTagCursorNamespace, raw)
	if apiErr != nil {
		return c, apiErr
	}
	if c.Name == "" || c.CreatedAt.IsZero() {
		return tagCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return c, nil
}

func encodeTagCursor(c tagCursor) string {
	return encodeCursor(moderationTagCursorNamespace, c)
}

func decodeAnnouncementCursor(raw string) (announcementCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return announcementCursor{}, nil
	}
	c, apiErr := decodeCursor[announcementCursor](moderationAnnouncementCursorNamespace, raw)
	if apiErr != nil {
		return c, apiErr
	}
	if c.AnnouncementID == "" || c.CreatedAt.IsZero() {
		return announcementCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return c, nil
}

func encodeAnnouncementCursor(c announcementCursor) string {
	return encodeCursor(moderationAnnouncementCursorNamespace, c)
}

func normalizeCursorLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > types.MaxPageSize:
		return types.MaxPageSize
	default:
		return limit
	}
}
