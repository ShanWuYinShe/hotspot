package search

import (
	"context"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func explainHasIndex(t *testing.T, ctx context.Context, qdb persistence.QueryableDB, wantIndex, query string, args ...any) {
	t.Helper()
	rows, err := qdb.Query(ctx, "EXPLAIN "+query, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan explain: %v", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("explain rows: %v", err)
	}
	plan := strings.Join(lines, "\n")
	if !strings.Contains(plan, wantIndex) {
		t.Fatalf("expected index %q in plan:\n%s", wantIndex, plan)
	}
}

// TestPostgresITSuggestShortKeywordUsesPGroonga：1-2 字关键词（尤其中文）
// 的 suggest 查询必须能走 PGroonga GIN 索引，且匹配语义与旧 ILIKE 子串一致。
func TestPostgresITSuggestShortKeywordUsesPGroonga(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		testdb.InsertUser(t, txFactory, "it_search_linyi", "林一")
		testdb.InsertUser(t, txFactory, "it_search_lin", "Lin")
		testdb.InsertUser(t, txFactory, "it_search_owner", "搜索作者")
		if err := qdb.Exec(ctx, "INSERT INTO user_settings (user_id) VALUES ($1), ($2), ($3) ON CONFLICT DO NOTHING", "it_search_linyi", "it_search_lin", "it_search_owner"); err != nil {
			t.Fatalf("insert settings: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO groups (group_id, name, description, location_name, latitude, longitude, owner_id, is_searchable) VALUES ('it_search_group','林间小队','','',0,0,$1,TRUE)", "it_search_owner"); err != nil {
			t.Fatalf("insert group: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id) VALUES ('it_search_moment',$1,'林间散步','',0,0,'public',0)", "it_search_owner"); err != nil {
			t.Fatalf("insert moment: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO moment_tags (moment_id, tag_name) VALUES ('it_search_moment','林间'), ('it_search_moment','目标标签')"); err != nil {
			t.Fatalf("insert tags: %v", err)
		}
		if err := qdb.Exec(ctx, `INSERT INTO users (user_id, nickname, location_name, latitude, longitude, s2_cell_id, created_at, last_online, presence_status, user_type)
SELECT 'it_search_bulk_'||i, '批量用户'||i, '', 0, 0, 0, now(), now(), 'offline', 'normal' FROM generate_series(1, 3000) i`); err != nil {
			t.Fatalf("insert bulk users: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO user_settings (user_id) SELECT user_id FROM users WHERE user_id LIKE 'it_search_bulk_%' ON CONFLICT DO NOTHING"); err != nil {
			t.Fatalf("insert bulk settings: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id) VALUES ('it_search_bulk_moment',$1,'批量瞬间','',0,0,'public',0)", "it_search_owner"); err != nil {
			t.Fatalf("insert bulk moment: %v", err)
		}
		if err := qdb.Exec(ctx, `INSERT INTO moment_tags (moment_id, tag_name)
SELECT 'it_search_bulk_moment', '批量标签'||i FROM generate_series(1, 3000) i`); err != nil {
			t.Fatalf("insert bulk tags: %v", err)
		}
		for _, table := range []string{"users", "user_settings", "groups", "moments", "moment_tags"} {
			if err := qdb.Exec(ctx, "ANALYZE "+table); err != nil {
				t.Fatalf("analyze %s: %v", table, err)
			}
		}
		repo := NewPostgresRepository(txFactory)
		resp, err := repo.Suggest(ctx, SearchSuggestRequest{Prefix: "林"})
		if err != nil {
			t.Fatalf("suggest 1-char: %v", err)
		}
		found := map[string]bool{}
		for _, item := range resp.Items {
			switch payload := item.Payload.(type) {
			case SearchSuggestUserPayload:
				if payload.User.UserID == "it_search_linyi" {
					found["user"] = true
				}
			case SearchSuggestGroupPayload:
				if payload.Group.Name == "林间小队" {
					found["group"] = true
				}
			case SearchSuggestMomentPayload:
				if strings.Contains(payload.DescriptionPreview, "林间散步") {
					found["moment"] = true
				}
			case TagItem:
				if payload.Name == "林间" {
					found["tag"] = true
				}
			}
		}
		for _, typ := range []string{"user", "group", "moment", "tag"} {
			if !found[typ] {
				t.Fatalf("1-char suggest missing %s, items=%#v", typ, resp.Items)
			}
		}
		resp2, err := repo.Suggest(ctx, SearchSuggestRequest{Prefix: "林一", Types: []string{"user"}})
		if err != nil {
			t.Fatalf("suggest 2-char: %v", err)
		}
		if len(resp2.Items) != 1 {
			t.Fatalf("2-char suggest = %#v", resp2.Items)
		}
		if payload, ok := resp2.Items[0].Payload.(SearchSuggestUserPayload); !ok || payload.User.UserID != "it_search_linyi" {
			t.Fatalf("2-char suggest = %#v", resp2.Items)
		}
		resp3, err := repo.Suggest(ctx, SearchSuggestRequest{Prefix: "lin", Types: []string{"user"}})
		if err != nil {
			t.Fatalf("suggest case-insensitive: %v", err)
		}
		hasLin := false
		for _, item := range resp3.Items {
			if payload, ok := item.Payload.(SearchSuggestUserPayload); ok && payload.User.UserID == "it_search_lin" {
				hasLin = true
			}
		}
		if !hasLin {
			t.Fatalf("case-insensitive suggest missed Lin: %#v", resp3.Items)
		}
		explainHasIndex(t, ctx, qdb, "idx_users_nickname_pgroonga", "SELECT 1 FROM users WHERE nickname &@ $1", "林一")
		explainHasIndex(t, ctx, qdb, "idx_moment_tags_name_pgroonga", "SELECT 1 FROM moment_tags WHERE tag_name &@ $1", "目标标签")
	})
}
