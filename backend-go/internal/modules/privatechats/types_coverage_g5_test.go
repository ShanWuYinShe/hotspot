package privatechats

import (
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/privatemessages"
	platformcursor "hotspot/backend-go/internal/platform/cursor"
)

// TestCoverageG5ChatCursorRoundTrip 覆盖游标编码/解码的正常往返（含置顶维度）。
func TestCoverageG5ChatCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 5, 1, 12, 0, 0, 123456789, time.UTC)
	cursor := encodeChatCursor(true, at.Format(time.RFC3339Nano), "u2")
	if cursor == "" {
		t.Fatal("expected non-empty cursor")
	}

	pinned, gotAt, targetID, err := decodeChatCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !pinned || !gotAt.Equal(at) || targetID != "u2" {
		t.Fatalf("roundtrip mismatch: pinned=%v at=%v target=%q", pinned, gotAt, targetID)
	}
}

func TestCoverageG5DecodeChatCursorEmpty(t *testing.T) {
	pinned, at, targetID, err := decodeChatCursor("")
	if err != nil || pinned || !at.IsZero() || targetID != "" {
		t.Fatalf("empty cursor must decode to zero values: pinned=%v at=%v target=%q err=%v", pinned, at, targetID, err)
	}
}

// TestCoverageG5DecodeChatCursorInvalidTimestamp 覆盖「密文合法但时间字段
// 无法解析」的分支（此前只用原始乱码覆盖了 decrypt 失败路径）。
func TestCoverageG5DecodeChatCursorInvalidTimestamp(t *testing.T) {
	bad, err := platformcursor.Encode("private_chats.list", chatCursor{
		LastActivityAt: "not-a-timestamp",
		UserID:         "u2",
	})
	if err != nil {
		t.Fatalf("encode cursor fixture: %v", err)
	}
	if _, _, _, err := decodeChatCursor(bad); !errors.Is(err, privatemessages.ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

// TestCoverageG5DecodeChatCursorMissingUserID 覆盖解码后 user_id 为空的校验。
func TestCoverageG5DecodeChatCursorMissingUserID(t *testing.T) {
	at := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	missingUser, err := platformcursor.Encode("private_chats.list", chatCursor{
		LastActivityAt: at.Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("encode cursor fixture: %v", err)
	}
	if _, _, _, err := decodeChatCursor(missingUser); !errors.Is(err, privatemessages.ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}
