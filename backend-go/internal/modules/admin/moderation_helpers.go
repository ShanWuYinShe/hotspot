package admin

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
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

func decodeTagCursor(raw string) (tagCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return tagCursor{}, nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return tagCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	var cursor tagCursor
	if err := json.Unmarshal(blob, &cursor); err != nil {
		return tagCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	if cursor.Name == "" || cursor.CreatedAt.IsZero() {
		return tagCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return cursor, nil
}

func encodeTagCursor(cursor tagCursor) string {
	blob, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func decodeAnnouncementCursor(raw string) (announcementCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return announcementCursor{}, nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return announcementCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	var cursor announcementCursor
	if err := json.Unmarshal(blob, &cursor); err != nil {
		return announcementCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	if cursor.AnnouncementID == "" || cursor.CreatedAt.IsZero() {
		return announcementCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return cursor, nil
}

func encodeAnnouncementCursor(cursor announcementCursor) string {
	blob, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func normalizeCursorLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 100:
		return 100
	default:
		return limit
	}
}
