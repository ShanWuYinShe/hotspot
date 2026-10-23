package groups

import (
	"testing"
	"time"
)

// TestSearchCursorNewestNanoPrecision 防回归（审查 P2-1，AGENTS.md「写入
// 精度 × 游标精度 × 谓词一致」）：newest 游标携带 RFC3339Nano 全精度。
// 此前秒级截断的游标在 fallback 过滤（cursor 项不在候选集，matchesSearch-
// Cursor 以 Before 谓词重定位）时，会把同一秒内早于 cursor 的行静默丢弃
// （cursorTime 被截断到秒首，同秒内更早的行不满足 Before(秒首)）。
func TestSearchCursorNewestNanoPrecision(t *testing.T) {
	base := time.Date(2026, 3, 30, 10, 0, 5, 0, time.UTC)
	mk := func(id string, at time.Time) GroupItem {
		s := at.Format(time.RFC3339Nano)
		return GroupItem{Group: GroupInfo{GroupID: id, MembersCount: 1}, CreatedAt: &s}
	}
	// newest 降序：g3(.800) > g2(.500) > g1(.200)，三条同秒不同纳秒。
	items := []GroupItem{
		mk("g3", base.Add(800*time.Millisecond)),
		mk("g2", base.Add(500*time.Millisecond)),
		mk("g1", base.Add(200*time.Millisecond)),
	}
	metas := map[string]groupSearchMeta{}
	for _, it := range items {
		metas[it.Group.GroupID] = groupSearchMeta{CreatedAt: groupCreatedAt(it)}
	}
	cursorOf := func(t *testing.T, item GroupItem) string {
		t.Helper()
		token, err := encodeSearchCursor(searchCursorForGroupItem(item, metas[item.Group.GroupID], searchSortNewest))
		if err != nil {
			t.Fatalf("encode cursor: %v", err)
		}
		return token
	}

	// 主定位路径：cursor 指向 g2 且在候选集内 → 下一页应只剩 g1。
	page, err := paginateSearchGroups(items, metas, searchSortNewest, &PaginationParams{Limit: 10, Cursor: cursorOf(t, items[1])})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Group.GroupID != "g1" {
		t.Fatalf("primary path: expected [g1], got %v", page.Items)
	}

	// fallback 路径：cursor 项（g2）已不在候选集（翻页间隙被解散），游标
	// 仍指向 .500 —— 同秒内更早的 g1 必须保留（秒级截断会把它丢掉）。
	remaining := []GroupItem{items[0], items[2]} // g3(.800) 与 g1(.200)
	cursorToken := cursorOf(t, items[1])
	page, err = paginateSearchGroups(remaining, metas, searchSortNewest, &PaginationParams{Limit: 10, Cursor: cursorToken})
	if err != nil {
		t.Fatalf("paginate fallback: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Group.GroupID != "g1" {
		t.Fatalf("fallback path: expected same-second earlier row [g1] to survive, got %v", page.Items)
	}
}
