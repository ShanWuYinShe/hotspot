package moments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITSearchKeywordSemantics：moments/search 移除冗余 ILIKE 复核后，
// PGroonga &@ 的大小写不敏感、%/_ 字面、CJK 子串语义由真实库锁定。
func TestPostgresITSearchKeywordSemantics(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		userID := testdb.InsertUser(t, txFactory, "it_moment_search_u1", "Moment Searcher")
		insertMoment := func(momentID, description string) {
			if err := qdb.Exec(ctx, "INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id) VALUES ($1,$2,$3,'',0,0,'public',0)", momentID, userID, description); err != nil {
				t.Fatalf("insert moment %s: %v", momentID, err)
			}
		}
		insertMoment("it_search_case", "ContractCaseMoment")
		insertMoment("it_search_pct", "合约%瞬间")
		repo := NewPostgresRepository(txFactory)
		search := func(keyword string) []string {
			t.Helper()
			page, err := repo.Search(ctx, SearchMomentsRequest{Keyword: &keyword, SortBy: "newest"}, "")
			if err != nil {
				t.Fatalf("search %q: %v", keyword, err)
			}
			ids := make([]string, 0, len(page.Items))
			for _, item := range page.Items {
				ids = append(ids, item.MomentID)
			}
			return ids
		}
		contains := func(ids []string, want string) bool {
			for _, id := range ids {
				if id == want {
					return true
				}
			}
			return false
		}
		if ids := search("contractcasemoment"); !contains(ids, "it_search_case") {
			t.Fatalf("lowercase keyword must match mixed-case description: %v", ids)
		}
		if ids := search("CONTRACTCASEMOMENT"); !contains(ids, "it_search_case") {
			t.Fatalf("uppercase keyword must match: %v", ids)
		}
		if ids := search("合约%瞬间"); !contains(ids, "it_search_pct") {
			t.Fatalf("literal %% keyword must match: %v", ids)
		}
		if ids := search("合约"); !contains(ids, "it_search_pct") {
			t.Fatalf("CJK substring must match: %v", ids)
		}
	})
}
