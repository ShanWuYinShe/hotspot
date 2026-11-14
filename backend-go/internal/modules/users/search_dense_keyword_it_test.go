package users

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITDenseKeywordPaginationBeyondCandidateCap：关键词命中数超过旧
// searchCandidateCap=500 时，ranked 路径仍能完整翻页（keyset），且 relevance
// 全序（昵称精确/前缀/包含/bio）跨页不重不漏。
func TestPostgresITDenseKeywordPaginationBeyondCandidateCap(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		const keyword = "密集关键词"
		// 650 个前缀命中（昵称）用户，last_online/created_at 逐条递减，保证全序稳定。
		if err := qdb.Exec(ctx, `INSERT INTO users (user_id, nickname, location_name, latitude, longitude, s2_cell_id, created_at, last_online, presence_status, user_type)
SELECT 'it_dense_'||i, '密集关键词测试'||i, '', 0, 0, 0,
       now() - make_interval(secs => i), now() - make_interval(secs => i), 'offline', 'normal'
FROM generate_series(1, 650) i`); err != nil {
			t.Fatalf("insert bulk users: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO user_settings (user_id) SELECT user_id FROM users WHERE user_id LIKE 'it_dense_%' ON CONFLICT DO NOTHING"); err != nil {
			t.Fatalf("insert bulk settings: %v", err)
		}
		insertSpecial := func(userID, nickname, bio string) {
			testdb.InsertUser(t, txFactory, userID, nickname)
			if bio != "" {
				if err := qdb.Exec(ctx, "UPDATE users SET bio = $2 WHERE user_id = $1", userID, bio); err != nil {
					t.Fatalf("set bio: %v", err)
				}
			}
			if err := qdb.Exec(ctx, "INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT DO NOTHING", userID); err != nil {
				t.Fatalf("insert settings: %v", err)
			}
		}
		insertSpecial("it_dense_exact", "密集关键词", "")
		insertSpecial("it_dense_prefix", "密集关键词前缀", "")
		insertSpecial("it_dense_contains", "用户密集关键词后缀", "")
		insertSpecial("it_dense_bio", "无关昵称", "个人简介包含密集关键词")

		repo := NewPostgresRepository(txFactory)
		collect := func(sortBy string) ([]string, int, string) {
			t.Helper()
			ids := make([]string, 0, 700)
			cursor := ""
			pages := 0
			for {
				req := SearchUsersRequest{Keyword: keyword, SortBy: sortBy}
				if cursor != "" {
					req.Pagination = &PaginationParams{Limit: 100, Cursor: cursor}
				} else {
					req.Pagination = &PaginationParams{Limit: 100}
				}
				page, err := repo.SearchProfiles(ctx, req)
				if err != nil {
					t.Fatalf("search %s page: %v", sortBy, err)
				}
				if page.InvalidCursor {
					t.Fatalf("unexpected invalid cursor on %s", sortBy)
				}
				pages++
				for _, item := range page.Items {
					ids = append(ids, item.UserID)
				}
				if page.NextCursor == "" {
					return ids, pages, cursor
				}
				cursor = page.NextCursor
				if pages > 20 {
					t.Fatalf("pagination did not terminate: %d pages", pages)
				}
			}
		}

		// relevance：654 = 650 前缀命中 + exact + prefix + contains + bio。
		ids, pages, _ := collect("")
		if len(ids) != 654 {
			t.Fatalf("relevance rows = %d, want 654 (pages=%d)", len(ids), pages)
		}
		if pages < 6 {
			t.Fatalf("expected >500 rows to require >=6 pages, got pages=%d", pages)
		}
		seen := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate user across pages: %s", id)
			}
			seen[id] = struct{}{}
		}
		if ids[0] != "it_dense_exact" {
			t.Fatalf("first relevance row = %q, want exact nickname match", ids[0])
		}
		if ids[1] != "it_dense_prefix" {
			t.Fatalf("second relevance row = %q, want prefix match", ids[1])
		}
		containsIdx, bioIdx := -1, -1
		for i, id := range ids {
			switch id {
			case "it_dense_contains":
				containsIdx = i
			case "it_dense_bio":
				bioIdx = i
			}
		}
		if containsIdx <= 0 || bioIdx <= containsIdx {
			t.Fatalf("relevance order wrong: contains=%d bio=%d", containsIdx, bioIdx)
		}

		// newest：同样跨页完整（证明新路径不再受 500 候选截断）。
		newestIDs, newestPages, _ := collect("newest")
		if len(newestIDs) != 654 || newestPages < 6 {
			t.Fatalf("newest rows=%d pages=%d, want 654/>=6", len(newestIDs), newestPages)
		}

		// 使用 relevance 游标请求 newest 必须被拒绝（opaque 游标绑定排序）。
		first, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: keyword, Pagination: &PaginationParams{Limit: 10}})
		if err != nil || first.NextCursor == "" {
			t.Fatalf("first page cursor: %#v err=%v", first, err)
		}
		cross, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: keyword, SortBy: "newest", Pagination: &PaginationParams{Limit: 10, Cursor: first.NextCursor}})
		if err != nil {
			t.Fatalf("cross-sort cursor: %v", err)
		}
		if !cross.InvalidCursor {
			t.Fatalf("cross-sort cursor must be invalid, got %#v", cross)
		}
	})
}

// TestPostgresITRankedPhoneExactMatch：ranked 路径（keyword 非空 + 无 radius +
// sort_by 缺省 relevance）此前只做昵称/bio 的 PGroonga 子串匹配，丢失候选路径
// 的手机号精确匹配分支。锁定：登录手机号（auth_accounts.login_id）精确匹配在
// ranked SQL 中生效，且受 us.phone_searchable 控制。
func TestPostgresITRankedPhoneExactMatch(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		seedPhoneUser := func(userID, nickname, loginID string, phoneSearchable bool) {
			t.Helper()
			testdb.InsertUser(t, txFactory, userID, nickname)
			if err := qdb.Exec(ctx, "INSERT INTO auth_accounts (login_id, user_id, password_hash) VALUES ($1, $2, 'x')", loginID, userID); err != nil {
				t.Fatalf("insert auth account %s: %v", userID, err)
			}
			if err := qdb.Exec(ctx, "INSERT INTO user_settings (user_id, phone_searchable) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET phone_searchable = EXCLUDED.phone_searchable", userID, phoneSearchable); err != nil {
				t.Fatalf("insert settings %s: %v", userID, err)
			}
		}
		seedPhoneUser("it_ranked_phone_on", "手机号可搜", "13900001111", true)
		seedPhoneUser("it_ranked_phone_off", "手机号不可搜", "13900002222", false)

		repo := NewPostgresRepository(txFactory)
		searchIDs := func(keyword string) []string {
			t.Helper()
			// 不传 radius、sort_by 缺省：命中 ranked SQL 路径。
			page, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: keyword})
			if err != nil {
				t.Fatalf("search %q: %v", keyword, err)
			}
			ids := make([]string, 0, len(page.Items))
			for _, item := range page.Items {
				ids = append(ids, item.UserID)
			}
			return ids
		}

		if ids := searchIDs("13900001111"); len(ids) != 1 || ids[0] != "it_ranked_phone_on" {
			t.Fatalf("phone_searchable=true should hit by exact phone, got %#v", ids)
		}
		if ids := searchIDs("13900002222"); len(ids) != 0 {
			t.Fatalf("phone_searchable=false must not hit by phone, got %#v", ids)
		}
		// 手机号匹配为精确语义：前缀不命中（昵称不含数字，PGroonga 子串亦不命中）。
		if ids := searchIDs("1390000111"); len(ids) != 0 {
			t.Fatalf("phone match must be exact, got %#v", ids)
		}
	})
}
