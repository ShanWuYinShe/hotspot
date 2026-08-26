package groups

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGuardGroupIDExtraction(t *testing.T) {
	id, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"group_id":"g1"}`)))
	if !ok || id != "g1" {
		t.Fatalf("normal: ok=%v id=%q", ok, id)
	}
	// 空格裁剪
	id, ok = guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"group_id":"  g2  "}`)))
	if !ok || id != "g2" {
		t.Fatalf("trim: ok=%v id=%q", ok, id)
	}
	// 空 body
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", nil)); ok {
		t.Fatal("nil body must fail")
	}
	// 超长 body
	big := bytes.NewReader(make([]byte, maxGuardBodyBytes+1))
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", big)); ok {
		t.Fatal("oversized body must fail")
	}
	// 非 JSON
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("not-json"))); ok {
		t.Fatal("invalid json must fail")
	}
	// 缺 group_id 字段
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"other":1}`))); ok {
		t.Fatal("missing group_id must fail")
	}
	// group_id 非字符串
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"group_id":123}`))); ok {
		t.Fatal("numeric group_id must fail")
	}
	// 读取错误（不可读 body）
	req := httptest.NewRequest(http.MethodPost, "/x", io.NopCloser(errReader{}))
	if _, ok := guardGroupID(req); ok {
		t.Fatal("unreadable body must fail")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return parsed
}

func TestServiceCoverageHelpers(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	_ = ctx

	// setter 注入不 panic
	SetBlockedTagChecker(nil)
	svc.SetGroupDismissAuditWriter(nil)
	svc.SetLocationProvider(nil)

	// ownerIDValue 空指针 → ""
	if v := ownerIDValue(nil); v != "" {
		t.Fatalf("nil owner: %q", v)
	}
	owner := "u1"
	if v := ownerIDValue(&owner); v != "u1" {
		t.Fatalf("owner: %q", v)
	}

	// derefString
	if v := derefString(nil); v != "" {
		t.Fatalf("nil deref: %q", v)
	}
	s := "x"
	if v := derefString(&s); v != "x" {
		t.Fatalf("deref: %q", v)
	}

	// groupUserInfo：avatar 为空/非空
	info := groupUserInfo(UserInfo{UserID: "u1", Nickname: "n1"})
	if info["user_id"] != "u1" {
		t.Fatalf("user info: %#v", info)
	}
	avatar := "http://a"
	info = groupUserInfo(UserInfo{UserID: "u2", Nickname: "n2", AvatarURL: &avatar})
	if info["avatar_url"] != "http://a" {
		t.Fatalf("user info avatar: %#v", info)
	}
	emptyAvatar := ""
	info = groupUserInfo(UserInfo{UserID: "u3", Nickname: "n3", AvatarURL: &emptyAvatar})
	if _, has := info["avatar_url"]; has {
		t.Fatalf("empty avatar must be omitted: %#v", info)
	}
}

func TestMatchesSearchCursor(t *testing.T) {
	item := GroupItem{Group: GroupInfo{GroupID: "g1", MembersCount: 10}}
	meta := groupSearchMeta{Distance: 100, CreatedAt: mustParseTime(t, "2025-01-01T00:00:00Z")}
	// distance：不同距离比大小，相同距离比 ID
	cur := groupSearchCursor{SortBy: searchSortDistance, Distance: 50}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("distance greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortDistance, Distance: 100, GroupID: "g0"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("distance equal id greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortDistance, Distance: 100, GroupID: "g2"}
	if matchesSearchCursor(meta, item, cur) {
		t.Fatal("distance equal id lower must not match")
	}

	// newest：时间不同比时间，相同比 ID
	cur = groupSearchCursor{SortBy: searchSortNewest, CreatedAt: "2025-01-02T00:00:00Z"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("older must match")
	}
	cur = groupSearchCursor{SortBy: searchSortNewest, CreatedAt: "2025-01-01T00:00:00Z", GroupID: "g0"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("same time id greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortNewest, CreatedAt: "2025-01-01T00:00:00Z", GroupID: "g2"}
	if matchesSearchCursor(meta, item, cur) {
		t.Fatal("same time id lower must not match")
	}

	// popular：人数不同比人数，相同比 ID
	cur = groupSearchCursor{SortBy: searchSortPopular, MembersCount: 20}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("fewer members must match")
	}
	cur = groupSearchCursor{SortBy: searchSortPopular, MembersCount: 10, GroupID: "g0"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("same count id greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortPopular, MembersCount: 10, GroupID: "g2"}
	if matchesSearchCursor(meta, item, cur) {
		t.Fatal("same count id lower must not match")
	}
}
