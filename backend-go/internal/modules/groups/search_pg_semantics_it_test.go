package groups

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITSearchKeywordSemantics：groups/search 移除冗余 ILIKE 复核后，
// PGroonga &@ 的大小写不敏感、%/_ 字面、CJK 子串语义由真实库锁定。
func TestPostgresITSearchKeywordSemantics(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		ownerID := testdb.InsertUser(t, txFactory, "it_group_search_owner", "Group Owner")
		s2CellID := geo.LeafCellID(geo.Point{Latitude: 0, Longitude: 0})
		insertGroup := func(groupID, name, description string) {
			if err := qdb.Exec(ctx, "INSERT INTO groups (group_id, name, description, location_name, latitude, longitude, owner_id, is_searchable, s2_cell_id) VALUES ($1,$2,$3,'',0,0,$4,TRUE,$5)", groupID, name, description, ownerID, s2CellID); err != nil {
				t.Fatalf("insert group %s: %v", groupID, err)
			}
		}
		insertGroup("it_group_case", "ContractCaseGroup", "")
		insertGroup("it_group_pct", "合约%群", "")
		repo := NewPostgresRepository(txFactory)
		search := func(keyword string) []string {
			t.Helper()
			page, err := repo.Search(ctx, SearchGroupsRequest{Keyword: keyword})
			if err != nil {
				t.Fatalf("search %q: %v", keyword, err)
			}
			ids := make([]string, 0, len(page.Items))
			for _, item := range page.Items {
				ids = append(ids, item.Group.GroupID)
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
		if ids := search("contractcasegroup"); !contains(ids, "it_group_case") {
			t.Fatalf("lowercase keyword must match mixed-case name: %v", ids)
		}
		if ids := search("CONTRACTCASEGROUP"); !contains(ids, "it_group_case") {
			t.Fatalf("uppercase keyword must match: %v", ids)
		}
		if ids := search("合约%群"); !contains(ids, "it_group_pct") {
			t.Fatalf("literal %% keyword must match: %v", ids)
		}
		if ids := search("合约"); !contains(ids, "it_group_pct") {
			t.Fatalf("CJK substring must match: %v", ids)
		}
	})
}
