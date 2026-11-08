package groups

import (
	"testing"
	"time"
)

// TestCovSearchSupportEdgeCases 补齐 search_support.go 的边界分支（非法游标、
// 未知排序、无效时间、空关键词等），全部为纯函数直测。
func TestCovSearchSupportEdgeCases(t *testing.T) {
	// normalizedSearchRequest：非法排序回退 relevance。
	req := normalizedSearchRequest(SearchGroupsRequest{Keyword: " hot ", SortBy: "magic"})
	if req.SortBy != searchSortRelevance {
		t.Fatalf("sort_by = %q, want relevance", req.SortBy)
	}
	if req.Keyword != "hot" {
		t.Fatalf("keyword = %q, want trimmed", req.Keyword)
	}
	if req.Radius == nil || *req.Radius != defaultSearchRadius {
		t.Fatalf("radius = %v, want default", req.Radius)
	}

	// searchNameMatchScore：完全不匹配返回 0；空关键词也返回 0。
	if got := searchNameMatchScore(GroupItem{Group: GroupInfo{Name: "alpha"}}, "zzz"); got != 0 {
		t.Fatalf("name match score = %d, want 0", got)
	}
	if got := searchNameMatchScore(GroupItem{Group: GroupInfo{Name: "alpha"}}, "  "); got != 0 {
		t.Fatalf("empty keyword score = %d, want 0", got)
	}

	// groupCreatedAt：nil / 空串 / 非法时间均返回零值。
	for _, item := range []GroupItem{
		{},
		{CreatedAt: strPtrCov("")},
		{CreatedAt: strPtrCov("not-a-time")},
	} {
		if got := groupCreatedAt(item); !got.IsZero() {
			t.Fatalf("groupCreatedAt(%#v) = %v, want zero", item.CreatedAt, got)
		}
	}

	// matchesSearchCursor：relevance 排序的回退比较（name_match / members / id）。
	meta := groupSearchMeta{NameMatch: 1}
	item := GroupItem{Group: GroupInfo{GroupID: "g_b", MembersCount: 5}}
	if !matchesSearchCursor(meta, item, groupSearchCursor{SortBy: searchSortRelevance, NameMatch: 2, MembersCount: 5, GroupID: "g_a"}) {
		t.Fatal("lower name match must sort after cursor")
	}
	if !matchesSearchCursor(groupSearchMeta{NameMatch: 2}, item, groupSearchCursor{SortBy: searchSortRelevance, NameMatch: 2, MembersCount: 5, GroupID: "g_a"}) {
		t.Fatal("equal name match with larger group id must match cursor")
	}
	if !matchesSearchCursor(groupSearchMeta{NameMatch: 2}, GroupItem{Group: GroupInfo{GroupID: "g_b", MembersCount: 4}}, groupSearchCursor{SortBy: searchSortRelevance, NameMatch: 2, MembersCount: 5, GroupID: "g_a"}) {
		t.Fatal("fewer members must match relevance cursor")
	}

	// encodeSearchCursor：空 group_id → 空游标；辅助函数返回空 group id。
	if token, err := encodeSearchCursor(groupSearchCursor{}); err != nil || token != "" {
		t.Fatalf("empty cursor encode = %q, %v", token, err)
	}
	// decodeSearchCursor：非法 base64 / 非法 JSON 均报错。
	if _, err := decodeSearchCursor("!!not-base64!!"); err == nil {
		t.Fatal("expected base64 decode error")
	}
	if _, err := decodeSearchCursor("bm90LWpzb24"); err == nil {
		t.Fatal("expected json unmarshal error")
	}
	// 空游标返回零值。
	if cursor, err := decodeSearchCursor("  "); err != nil || cursor.GroupID != "" {
		t.Fatalf("empty decode = %#v, %v", cursor, err)
	}

	// paginateSearchGroups：非法游标报错。
	if _, err := paginateSearchGroups(nil, nil, searchSortRelevance, &PaginationParams{Cursor: "!!bad!!"}); err == nil {
		t.Fatal("expected paginate cursor error")
	}
	// 游标项已不在列表中 → 走 matchesSearchCursor 回退过滤。
	items := []GroupItem{
		{Group: GroupInfo{GroupID: "g_b", Name: "Beta", MembersCount: 5}},
	}
	cursor, err := encodeSearchCursor(groupSearchCursor{SortBy: searchSortRelevance, GroupID: "g_a", NameMatch: 1, MembersCount: 5})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	metas := map[string]groupSearchMeta{"g_b": {NameMatch: 1}}
	page, err := paginateSearchGroups(items, metas, searchSortRelevance, &PaginationParams{Cursor: cursor})
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Group.GroupID != "g_b" {
		t.Fatalf("fallback page = %#v", page.Items)
	}
	// 游标项命中列表 → 从其后恢复。
	items = []GroupItem{
		{Group: GroupInfo{GroupID: "g_a", Name: "Alpha", MembersCount: 5}, CreatedAt: strPtrCov(time.Now().UTC().Format(time.RFC3339Nano))},
		{Group: GroupInfo{GroupID: "g_b", Name: "Beta", MembersCount: 5}},
	}
	metas = map[string]groupSearchMeta{
		"g_a": {NameMatch: 1},
		"g_b": {NameMatch: 1},
	}
	cursor, err = encodeSearchCursor(searchCursorForGroupItem(items[0], metas["g_a"], searchSortRelevance))
	if err != nil {
		t.Fatalf("encode matched cursor: %v", err)
	}
	page, err = paginateSearchGroups(items, metas, searchSortRelevance, &PaginationParams{Cursor: cursor})
	if err != nil {
		t.Fatalf("paginate matched: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Group.GroupID != "g_b" {
		t.Fatalf("matched page = %#v", page.Items)
	}

	// inviteExpired：nil / 空 / 非法时间视为未过期；过去时间视为过期。
	for _, invite := range []GroupInviteItem{
		{},
		{ExpiresAt: strPtrCov("")},
		{ExpiresAt: strPtrCov("bad")},
	} {
		if inviteExpired(invite) {
			t.Fatalf("invite %#v must not be expired", invite)
		}
	}
	if !inviteExpired(GroupInviteItem{ExpiresAt: strPtrCov(time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano))}) {
		t.Fatal("past invite must be expired")
	}

	// 非法游标错误在 search 入口被识别（decode 错误包装）。
	if _, err := decodeSearchCursor("****"); err == nil {
		t.Fatal("expected invalid cursor error")
	}
}

func strPtrCov(s string) *string { return &s }
