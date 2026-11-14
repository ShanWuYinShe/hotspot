package users

import (
	"context"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/geo"
)

// TestSearchProfilesPhoneExactMatch：keyword 为手机号格式（纯数字）时，
// phone_searchable=true 的用户可被登录手机号精确匹配（spec user.md →
// users/search keyword 说明；内存版以 profile.Phone 承载 login_id 回填值）。
func TestSearchProfilesPhoneExactMatch(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	phoneOwner := validProfile("u_phone")
	phoneOwner.Phone = "13800138000"
	if err := repo.PutProfile(ctx, phoneOwner); err != nil {
		t.Fatalf("put phone owner: %v", err)
	}
	// phone_searchable 默认 false：不命中
	if err := repo.PutSettings(ctx, "u_phone", defaultSettings()); err != nil {
		t.Fatalf("put settings: %v", err)
	}

	page, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "13800138000"})
	if err != nil {
		t.Fatalf("search default: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("phone_searchable=false must not match by phone, got %#v", page.Items)
	}

	// phone_searchable=true：精确匹配命中
	settings := defaultSettings()
	settings.PhoneSearchable = true
	if err := repo.PutSettings(ctx, "u_phone", settings); err != nil {
		t.Fatalf("put settings: %v", err)
	}
	page, err = repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "13800138000"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].UserID != "u_phone" {
		t.Fatalf("expected exact phone match, got %#v", page.Items)
	}

	// 精确匹配：前缀不命中
	page, err = repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "138001380"})
	if err != nil {
		t.Fatalf("search prefix: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("phone match must be exact, got %#v", page.Items)
	}
}

// TestBuildSearchUsersQueryPhoneBranch：SQL 构建——手机号格式 keyword 追加
// auth_accounts 精确匹配分支（受 phone_searchable 控制）；非数字 keyword 不追加。
func TestBuildSearchUsersQueryPhoneBranch(t *testing.T) {
	// 手机号 keyword：含 auth_accounts 子查询
	sql, args := buildSearchUsersQuery(geo.Bounds{}, "13800138000", "", false, false)
	if sql == "" {
		t.Fatal("expected non-empty sql")
	}
	if !strings.Contains(sql, "auth_accounts") || !strings.Contains(sql, "phone_searchable") {
		t.Fatalf("expected phone exact-match branch in sql: %s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("expected escaped-keyword + keyword + limit args, got %#v", args)
	}

	// 非手机号 keyword：无 auth_accounts 分支
	sql, args = buildSearchUsersQuery(geo.Bounds{}, "alice", "", false, false)
	if strings.Contains(sql, "auth_accounts") {
		t.Fatalf("non-numeric keyword must not include phone branch: %s", sql)
	}
	if !strings.Contains(sql, "&@") {
		t.Fatalf("non-numeric keyword must use pgroonga prefilter: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected raw-keyword + limit args, got %#v", args)
	}
}

// TestBuildRankedSearchUsersQueryPhoneBranch：ranked 路径（keyword 非空 + 无
// radius + relevance/newest 排序）的 SQL 同样要带 auth_accounts 手机号精确匹配
// 分支（受 phone_searchable 控制），否则该路径会丢失手机号搜索能力；非手机号
// keyword 不追加该分支。
func TestBuildRankedSearchUsersQueryPhoneBranch(t *testing.T) {
	sql, args := buildRankedSearchUsersQuery("13800138000", "", "relevance", searchCursor{}, false, 21)
	if !strings.Contains(sql, "auth_accounts") || !strings.Contains(sql, "phone_searchable") {
		t.Fatalf("expected phone exact-match branch in ranked sql: %s", sql)
	}
	// 原有 PGroonga 昵称/bio 匹配必须保留（手机号命中与模糊匹配并集）。
	if !strings.Contains(sql, "&@") {
		t.Fatalf("ranked sql must keep pgroonga prefilter: %s", sql)
	}
	// 参数顺序：keyword、手机号精确匹配 keyword、limit。
	if len(args) != 3 {
		t.Fatalf("expected keyword + phone + limit args, got %#v", args)
	}
	if args[0] != "13800138000" || args[1] != "13800138000" {
		t.Fatalf("expected keyword + phone args, got %#v", args)
	}
	if !strings.Contains(sql, "aa.login_id = $2") {
		t.Fatalf("phone branch placeholder mismatch: %s", sql)
	}

	// 非手机号 keyword：无 auth_accounts 分支，参数只有 keyword + limit。
	sql, args = buildRankedSearchUsersQuery("alice", "", "relevance", searchCursor{}, false, 21)
	if strings.Contains(sql, "auth_accounts") {
		t.Fatalf("non-numeric keyword must not include phone branch: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected keyword + limit args, got %#v", args)
	}
}
