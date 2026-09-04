package moments

import (
	"testing"
	"time"
)

// 迁移自 users 模块 TestInMemoryReactedMomentsCursorUsesReactedAt：已反应列表
// 的 keyset 游标必须以 reacted_at 为序键（而非 created_at）——原内存实现删除后，
// 该契约由 SQL 排序列 + 游标编解码的静态断言与编解码往返测试承载（SQL 语义由
// IT 测试覆盖）。
func TestMomentsByUserCursorRoundTripUsesReactedAt(t *testing.T) {
	// 游标编解码往返： reacted_at (RFC3339Nano) + moment_id 无损。
	at := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	token := encodeMomentsByUserCursor(at.Format(time.RFC3339Nano), "moment_newer")
	if token == "" {
		t.Fatal("encode cursor failed")
	}
	gotAt, gotID, err := decodeMomentsByUserCursor(token)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !gotAt.Equal(at) || gotID != "moment_newer" {
		t.Fatalf("cursor round-trip mismatch: at=%v id=%s", gotAt, gotID)
	}
	// 空游标 = 首页。
	if at0, id0, err := decodeMomentsByUserCursor(""); err != nil || !at0.IsZero() || id0 != "" {
		t.Fatalf("empty cursor should decode to zero values: %v %q %q", err, at0, id0)
	}
	// 非法游标必须报错（不得静默当首页，避免翻页丢数据）。
	if _, _, err := decodeMomentsByUserCursor("garbage"); err == nil {
		t.Fatal("garbage cursor should fail decoding")
	}
	// SQL 排序键静态断言：已反应列表以 reacted_at DESC、moment_id DESC 排序，
	// 与游标 (reacted_at, moment_id) 严格一致——列或顺序变更即本断言失败。
	if !containsOrdered(queryListReactedByUser, "ORDER BY r.reacted_at DESC, m.moment_id DESC") {
		t.Fatal("queryListReactedByUser keyset order drifted from cursor (reacted_at, moment_id)")
	}
	if !containsOrdered(queryListMomentsByAuthor, "ORDER BY m.created_at DESC, m.moment_id DESC") {
		t.Fatal("queryListMomentsByAuthor keyset order drifted from cursor (created_at, moment_id)")
	}
}

func containsOrdered(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
