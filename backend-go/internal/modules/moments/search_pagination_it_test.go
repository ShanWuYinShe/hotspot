package moments

import (
	"context"
	"fmt"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITSearchKeysetPaginationNoDupNoSkip 锁定 moments/search 三种排序的
// keyset 翻页不重不漏：
//   - 游标分值与时间必须来自 SQL 实际排序输出（sort_score + 全精度 created_at），
//     不能由 Go 侧重算（浮点 ulp 差异）或用秒级展示时间（同秒行被跳过）；
//   - 数据构造为「同一秒内不同微秒 + 相近 interaction_count」，专门覆盖
//     popular 分值近似相等与 relevance/newest 同秒并列的两类页边界。
func TestPostgresITSearchKeysetPaginationNoDupNoSkip(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		userID := testdb.InsertUser(t, txFactory, "it_moment_search_page_u1", "Pager")
		base := time.Now().UTC().Truncate(time.Second).Add(-4 * time.Hour)
		const total = 12
		insert := func(momentID string, microOffset int, interactionCount int) {
			t.Helper()
			createdAt := base.Add(time.Duration(microOffset) * time.Microsecond)
			if err := qdb.Exec(ctx,
				"INSERT INTO moments (moment_id, user_id, description, created_at, location_name, latitude, longitude, privacy_level, s2_cell_id, interaction_count) VALUES ($1,$2,$3,$4,'',0,0,'public',0,$5)",
				momentID, userID, "分页测试瞬间", createdAt, interactionCount); err != nil {
				t.Fatalf("insert moment %s: %v", momentID, err)
			}
		}
		for i := 0; i < total; i++ {
			insert(fmt.Sprintf("it_search_page_%02d", i), (i*37)%1000000, 10+i%4)
		}

		repo := NewPostgresRepository(txFactory)
		pageThrough := func(name string, req SearchMomentsRequest, limit int) []string {
			var out []string
			cursor := ""
			for page := 0; page < 200; page++ {
				r := req
				r.Pagination = &PaginationParams{Limit: limit, Cursor: cursor}
				result, err := repo.Search(ctx, r, "")
				if err != nil {
					t.Fatalf("%s page %d: %v", name, page, err)
				}
				for _, item := range result.Items {
					out = append(out, item.MomentID)
				}
				if result.NextCursor == nil {
					return out
				}
				cursor = *result.NextCursor
			}
			t.Fatalf("%s pagination did not terminate", name)
			return nil
		}

		keyword := "分页测试"
		cases := []struct {
			name string
			req  SearchMomentsRequest
		}{
			{"popular", SearchMomentsRequest{SortBy: "popular"}},
			{"relevance", SearchMomentsRequest{Keyword: &keyword, SortBy: "relevance"}},
			{"newest", SearchMomentsRequest{Keyword: &keyword, SortBy: "newest"}},
		}
		for _, tc := range cases {
			full := pageThrough(tc.name, tc.req, 100)
			if len(full) != total {
				t.Fatalf("%s: expected %d rows in one full page, got %d: %v", tc.name, total, len(full), full)
			}
			for _, limit := range []int{1, 2, 3, 5} {
				paged := pageThrough(tc.name, tc.req, limit)
				if len(paged) != len(full) {
					t.Fatalf("%s limit=%d: paginated %d rows, want %d (dup/skip): %v", tc.name, limit, len(paged), len(full), paged)
				}
				for i := range full {
					if paged[i] != full[i] {
						t.Fatalf("%s limit=%d: page order drift at %d: want %s got %s", tc.name, limit, i, full[i], paged[i])
					}
				}
			}
		}
	})
}
