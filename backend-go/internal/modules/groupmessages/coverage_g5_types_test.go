package groupmessages

import (
	"errors"
	"testing"

	platformcursor "hotspot/backend-go/internal/platform/cursor"
)

// TestDecodeGroupMessageCursorRejectsInvalidTimestamp 覆盖「游标可解密但
// sent_at 非法」的防御分支：按 ErrInvalidCursor 拒绝，而不是带着零值时间
// 继续查询（否则会把翻页静默截断）。
func TestDecodeGroupMessageCursorRejectsInvalidTimestamp(t *testing.T) {
	token, err := platformcursor.Encode("group_messages.history", groupMessageCursor{SentAt: "not-a-time", MessageID: "m1"})
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	if _, _, err := decodeGroupMessageCursor(token); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("invalid sent_at err = %v, want ErrInvalidCursor", err)
	}

	// 合法游标：时间按 UTC 解析、message_id 原样返回。
	token, err = platformcursor.Encode("group_messages.history", groupMessageCursor{SentAt: "2026-01-01T08:00:00.123456789+08:00", MessageID: "gm_cursor_1"})
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	sentAt, messageID, err := decodeGroupMessageCursor(token)
	if err != nil {
		t.Fatalf("decode valid cursor: %v", err)
	}
	if messageID != "gm_cursor_1" {
		t.Fatalf("message_id = %q", messageID)
	}
	if want := "2026-01-01T00:00:00.123456789Z"; sentAt.Format("2006-01-02T15:04:05.999999999Z") != want {
		t.Fatalf("sent_at = %v, want %s", sentAt, want)
	}
	if sentAt.Location().String() != "UTC" {
		t.Fatalf("sent_at location = %v, want UTC", sentAt.Location())
	}
}
