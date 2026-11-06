package users

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITSearchKeywordSemantics：PGroonga &@ 子串匹配的契约语义——
// 大小写不敏感、%/_ 按字面匹配、1 字符 CJK 可命中。移除冗余 ILIKE 复核后由
// 本用例在真实库锁定行为（含 PGroonga NormalizerNFKC 归一）。
func TestPostgresITSearchKeywordSemantics(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		insertSettings := func(userID string) {
			if err := qdb.Exec(ctx, "INSERT INTO user_settings (user_id) VALUES ($1)", userID); err != nil {
				t.Fatalf("insert settings for %s: %v", userID, err)
			}
		}
		alphaID := testdb.InsertUser(t, txFactory, "it_search_alpha", "ContractCaseAlpha")
		insertSettings(alphaID)
		pctID := testdb.InsertUser(t, txFactory, "it_search_pct", "合约%字面")
		insertSettings(pctID)
		repo := NewPostgresRepository(txFactory)

		search := func(keyword string) []string {
			t.Helper()
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
		contains := func(ids []string, want string) bool {
			for _, id := range ids {
				if id == want {
					return true
				}
			}
			return false
		}

		// 大小写不敏感（PGroonga 归一 + ILIKE 复核移除后语义不变）。
		if ids := search("contractcasealpha"); !contains(ids, alphaID) {
			t.Fatalf("lowercase keyword must match mixed-case nickname: %v", ids)
		}
		if ids := search("CONTRACTCASEALPHA"); !contains(ids, alphaID) {
			t.Fatalf("uppercase keyword must match mixed-case nickname: %v", ids)
		}
		// % 按字面量匹配：只有真的包含 % 的昵称命中。
		if ids := search("合约%字面"); !contains(ids, pctID) {
			t.Fatalf("literal %% keyword must match nickname containing %%: %v", ids)
		}
		if ids := search("合约%"); !contains(ids, pctID) {
			t.Fatalf("prefix with literal %% must match: %v", ids)
		}
		// 1 字符 CJK 前缀可命中。
		if ids := search("合约"); !contains(ids, pctID) {
			t.Fatalf("CJK substring must match: %v", ids)
		}
	})
}
