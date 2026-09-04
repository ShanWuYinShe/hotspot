package groups

import (
	"time"

	"hotspot/backend-go/internal/platform/cursor"
)

// groupPageCursor 是成员/申请/邀请等列表的 keyset 游标载荷：(排序时间, 稳定 id)。
// cursor 不透明（AES-GCM 加密），客户端只透传；编解码仅存在于 Repository 层
// （AGENTS.md 分页规范）。
type groupPageCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

var groupPageCursorCodec = cursor.NewCodec("groups-page")

func encodeGroupPageCursor(at time.Time, id string) string {
	token, err := groupPageCursorCodec.Encode(groupPageCursor{At: at.UTC().Format(time.RFC3339Nano), ID: id})
	if err != nil {
		return ""
	}
	return token
}

func decodeGroupPageCursor(raw string) (time.Time, string, error) {
	if raw == "" {
		return time.Time{}, "", nil
	}
	var decoded groupPageCursor
	if err := groupPageCursorCodec.Decode(raw, &decoded); err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	if decoded.At == "" || decoded.ID == "" {
		return time.Time{}, "", ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return parsed.UTC(), decoded.ID, nil
}
