package users

import (
	"errors"
	"testing"
	"time"
)

type cursorKeyItem struct {
	at string
	id string
}

func keysetProbe(items []cursorKeyItem) (func(int) (string, string), int) {
	return func(i int) (string, string) { return items[i].at, items[i].id }, len(items)
}

// TestUserListCursorKeysetPositioning 防回归（审查 P2-15）：内存实现的列表
// 游标定位此前要求游标行仍在列表中，游标行被删除（取关/解封）时返回空页且
// 静默成功。PG 是 keyset 谓词，游标行消失后仍应继续翻页——这里固定
// 「DESC by (at, id)」的定位语义。
func TestUserListCursorKeysetPositioning(t *testing.T) {
	at3 := time.Date(2026, 3, 20, 0, 0, 3, 0, time.UTC)
	at2 := time.Date(2026, 3, 20, 0, 0, 2, 0, time.UTC)

	// 正常命中：游标为 u2 → 起点在 u1。
	items := []cursorKeyItem{
		{"2026-03-20T00:00:03Z", "u3"},
		{"2026-03-20T00:00:02Z", "u2"},
		{"2026-03-20T00:00:01Z", "u1"},
	}
	keyAt, n := keysetProbe(items)
	start, err := userListCursorStart(encodeUserCursor(at2, "u2"), keyAt, n)
	if err != nil || start != 2 {
		t.Fatalf("existing cursor: start=%d err=%v want 2", start, err)
	}

	// 游标行已被删除（列表里没有 u2）→ 仍按比较定位到 u1，而不是空页。
	items = []cursorKeyItem{
		{"2026-03-20T00:00:03Z", "u3"},
		{"2026-03-20T00:00:01Z", "u1"},
	}
	keyAt, n = keysetProbe(items)
	start, err = userListCursorStart(encodeUserCursor(at2, "u2"), keyAt, n)
	if err != nil || start != 1 {
		t.Fatalf("deleted cursor row: start=%d err=%v want 1", start, err)
	}

	// 同一时间戳的 tie-break：排序为 ID 升序（u1 < u2 < u3），
	// 游标 (at3,u2) → 下一页从 u3 开始。
	items = []cursorKeyItem{
		{"2026-03-20T00:00:03Z", "u1"},
		{"2026-03-20T00:00:03Z", "u2"},
		{"2026-03-20T00:00:03Z", "u3"},
	}
	keyAt, n = keysetProbe(items)
	start, err = userListCursorStart(encodeUserCursor(at3, "u2"), keyAt, n)
	if err != nil || start != 2 {
		t.Fatalf("same-second tie-break: start=%d err=%v want 2", start, err)
	}

	// 游标指向最后一条 → 下一页为空。
	keyAt, n = keysetProbe(items)
	start, err = userListCursorStart(encodeUserCursor(at3, "u3"), keyAt, n)
	if err != nil || start != n {
		t.Fatalf("tail cursor: start=%d err=%v want %d", start, err, n)
	}

	// 非法游标 → ErrInvalidCursor（handler 映射 10400，与 PG 一致）。
	if _, err := userListCursorStart("not-a-valid-cursor", keyAt, n); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("invalid cursor: err=%v want ErrInvalidCursor", err)
	}
}
