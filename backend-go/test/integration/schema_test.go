package integration

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	platformgeo "hotspot/backend-go/internal/platform/geo"
	platformmigrate "hotspot/backend-go/internal/platform/migrate"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRequiredTablesExist(t *testing.T) {
	requireIntegrationEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, getenvOrFail(t, "POSTGRES_DSN"))
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	missing := make([]string, 0)
	for _, table := range platformmigrate.RequiredTables() {
		var present bool
		if err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = $1
)
`, table).Scan(&present); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !present {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("missing required tables: %s", strings.Join(missing, ", "))
	}
}

func TestExplainUsesExpectedS2Indexes(t *testing.T) {
	requireIntegrationEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, getenvOrFail(t, "POSTGRES_DSN"))
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	t.Run("small fixture baseline", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		hitCell, _ := seedExplainFixture(t, ctx, tx)
		rangeArgs := []any{[]int64{hitCell}, []int64{hitCell}}
		cases := []struct {
			name          string
			sql           string
			args          []any
			expectedIndex string
		}{
			{
				name: "map moments uses public partial index",
				sql: `
SELECT m.moment_id, m.description, COALESCE(m.location_name, ''), m.latitude, m.longitude, m.created_at
FROM moments m
WHERE m.privacy_level = 'public'
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)`,
				args:          rangeArgs,
				expectedIndex: "idx_moments_public_s2_cell_id",
			},
			{
				name: "feed nearby moments uses public partial index",
				sql: `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.moment_id AND ml.user_id = $3),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $3),
       EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $3 AND ur.is_following = TRUE)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)`,
				args:          []any{[]int64{hitCell}, []int64{hitCell}, "viewer_1"},
				expectedIndex: "idx_moments_public_s2_cell_id",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				plan, err := explainQueryPlan(ctx, tx, tc.sql, tc.args...)
				if err != nil {
					t.Fatalf("explain query: %v", err)
				}
				assertPlanUsesIndex(t, plan, tc.expectedIndex)
			})
		}
	})

	t.Run("scaled data uses groups s2 partial index", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		hitCell, _ := seedScaledExplainFixture(t, ctx, tx, 1000, 50)
		plan, err := explainQueryPlan(ctx, tx, mapGroupsExplainSQL(), []int64{hitCell}, []int64{hitCell})
		if err != nil {
			t.Fatalf("explain query: %v", err)
		}
		assertPlanUsesIndex(t, plan, "idx_groups_visible_s2_cell_id")

		plan, err = explainQueryPlan(ctx, tx, buildGroupSearchExplainSQL(), 35.0, 112.0, []int64{hitCell}, []int64{hitCell}, 5000, 3)
		if err != nil {
			t.Fatalf("explain group search: %v", err)
		}
		assertPlanUsesIndex(t, plan, "idx_groups_visible_s2_cell_id")
	})

	t.Run("moments non-private search uses dedicated partial index", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		hitCell, _ := seedScaledExplainFixture(t, ctx, tx, 1000, 50)

		orPlan, err := explainQueryPlan(ctx, tx, feedAllMomentsExplainSQL(), []int64{hitCell}, []int64{hitCell}, "viewer_1")
		if err != nil {
			t.Fatalf("explain OR query: %v", err)
		}
		assertPlanUsesIndex(t, orPlan, "idx_moments_public_s2_cell_id")
		assertPlanHasNodeType(t, orPlan, "Bitmap Heap Scan")

		momentsPlan, err := explainQueryPlan(ctx, tx, buildSearchMomentsExplainSQL(), []int64{hitCell}, []int64{hitCell})
		if err != nil {
			t.Fatalf("explain moments search: %v", err)
		}
		assertPlanUsesIndex(t, momentsPlan, "idx_moments_non_private_s2_cell_id")
	})

	t.Run("corner case datasets", func(t *testing.T) {
		t.Run("following-only OR still keeps public partial index in plan", func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin tx: %v", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			missCell, err := seedFollowingOnlyExplainFixture(t, ctx, tx)
			if err != nil {
				t.Fatalf("seed following-only fixture: %v", err)
			}
			plan, err := explainQueryPlan(ctx, tx, feedAllMomentsExplainSQL(), []int64{missCell}, []int64{missCell}, "viewer_1")
			if err != nil {
				t.Fatalf("explain OR query: %v", err)
			}
			assertPlanUsesAnyIndex(t, plan, "idx_moments_public_s2_cell_id", "idx_moments_non_private_s2_cell_id")
			assertPlanHasNodeType(t, plan, "Bitmap Heap Scan")
		})

		t.Run("partial predicate pollution prefers partial index when searchable live rows dominate", func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin tx: %v", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			hitCell, _ := seedGroupsPredicateCornerCaseFixture(t, ctx, tx)
			plan, err := explainQueryPlan(ctx, tx, mapGroupsExplainSQL(), []int64{hitCell}, []int64{hitCell})
			if err != nil {
				t.Fatalf("explain groups query: %v", err)
			}
			assertPlanUsesIndex(t, plan, "idx_groups_visible_s2_cell_id")
		})

		t.Run("non-private skew uses dedicated partial index", func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin tx: %v", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			hitCell, _ := seedMomentsPrivacySkewFixture(t, ctx, tx)
			plan, err := explainQueryPlan(ctx, tx, buildSearchMomentsExplainSQL(), []int64{hitCell}, []int64{hitCell})
			if err != nil {
				t.Fatalf("explain moments query: %v", err)
			}
			assertPlanUsesIndex(t, plan, "idx_moments_non_private_s2_cell_id")
		})
	})
}

type explainDocument struct {
	Plan explainPlanNode `json:"Plan"`
}

type explainPlanNode struct {
	NodeType  string            `json:"Node Type"`
	IndexName string            `json:"Index Name"`
	Plans     []explainPlanNode `json:"Plans"`
}

func explainQueryPlan(ctx context.Context, tx pgx.Tx, sql string, args ...any) (explainPlanNode, error) {
	if _, err := tx.Exec(ctx, `SET LOCAL enable_seqscan = off`); err != nil {
		return explainPlanNode{}, err
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `EXPLAIN (FORMAT JSON) `+sql, args...).Scan(&raw); err != nil {
		return explainPlanNode{}, err
	}
	var docs []explainDocument
	if err := json.Unmarshal(raw, &docs); err != nil {
		return explainPlanNode{}, err
	}
	if len(docs) == 0 {
		return explainPlanNode{}, nil
	}
	return docs[0].Plan, nil
}

func findIndexNode(node explainPlanNode, indexName string) (explainPlanNode, bool) {
	if node.IndexName == indexName {
		return node, true
	}
	for _, child := range node.Plans {
		if found, ok := findIndexNode(child, indexName); ok {
			return found, true
		}
	}
	return explainPlanNode{}, false
}

func findNodeType(node explainPlanNode, nodeType string) (explainPlanNode, bool) {
	if node.NodeType == nodeType {
		return node, true
	}
	for _, child := range node.Plans {
		if found, ok := findNodeType(child, nodeType); ok {
			return found, true
		}
	}
	return explainPlanNode{}, false
}

func isIndexNodeType(nodeType string) bool {
	switch nodeType {
	case "Index Scan", "Index Only Scan", "Bitmap Index Scan":
		return true
	default:
		return false
	}
}

func assertPlanUsesIndex(t *testing.T, plan explainPlanNode, indexName string) {
	t.Helper()
	node, ok := findIndexNode(plan, indexName)
	if !ok {
		t.Fatalf("expected plan to use index %q, plan=%#v", indexName, plan)
	}
	if !isIndexNodeType(node.NodeType) {
		t.Fatalf("expected index node for %q, got %q", indexName, node.NodeType)
	}
}

func assertPlanUsesAnyIndex(t *testing.T, plan explainPlanNode, indexNames ...string) {
	t.Helper()
	for _, indexName := range indexNames {
		if node, ok := findIndexNode(plan, indexName); ok {
			if !isIndexNodeType(node.NodeType) {
				t.Fatalf("expected index node for %q, got %q", indexName, node.NodeType)
			}
			return
		}
	}
	t.Fatalf("expected plan to use one of indexes %q, plan=%#v", indexNames, plan)
}

func assertPlanHasNodeType(t *testing.T, plan explainPlanNode, nodeType string) {
	t.Helper()
	if _, ok := findNodeType(plan, nodeType); !ok {
		t.Fatalf("expected plan to contain node type %q, plan=%#v", nodeType, plan)
	}
}

func seedExplainFixture(t *testing.T, ctx context.Context, tx pgx.Tx) (int64, int64) {
	t.Helper()
	hitPoint := platformgeo.Point{Latitude: 35.0, Longitude: 112.0}
	missPoint := platformgeo.Point{Latitude: 35.1, Longitude: 112.1}
	hitCell := platformgeo.LeafCellID(hitPoint)
	missCell := platformgeo.LeafCellID(missPoint)
	mustExec(t, ctx, tx, `
INSERT INTO users (user_id, nickname, location_name, latitude, longitude, s2_cell_id)
VALUES
  ('author_1', 'author 1', 'hit', 35.0, 112.0, $1),
  ('author_2', 'author 2', 'miss', 35.1, 112.1, $2),
  ('viewer_1', 'viewer', 'viewer', 35.0, 112.0, $1)
`, hitCell, missCell)
	mustExec(t, ctx, tx, `
INSERT INTO user_relations (user_id, target_user_id, is_following)
VALUES ('viewer_1', 'author_2', TRUE)
`)
	mustExec(t, ctx, tx, `
INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id)
VALUES
  ('moment_hit_public', 'author_1', 'hit', 'hit', 35.0, 112.0, 'public', $1),
  ('moment_miss_public', 'author_2', 'miss', 'miss', 35.1, 112.1, 'public', $2),
  ('moment_hit_private', 'author_1', 'private', 'hit', 35.0, 112.0, 'private', $1)
`, hitCell, missCell)
	mustExec(t, ctx, tx, `
INSERT INTO groups (group_id, name, location_name, latitude, longitude, is_searchable, owner_id, s2_cell_token, s2_cell_id, tags)
VALUES
  ('group_hit_visible', 'hit group', 'hit', 35.0, 112.0, TRUE, 'author_1', '', $1, '{}'),
  ('group_miss_visible', 'miss group', 'miss', 35.1, 112.1, TRUE, 'author_1', '', $2, '{}'),
  ('group_hit_hidden', 'hidden group', 'hit', 35.0, 112.0, FALSE, 'author_1', '', $1, '{}')
`, hitCell, missCell)
	return hitCell, missCell
}

func seedScaledExplainFixture(t *testing.T, ctx context.Context, tx pgx.Tx, totalRows, hitRows int) (int64, int64) {
	t.Helper()
	hitPoint := platformgeo.Point{Latitude: 35.0, Longitude: 112.0}
	missPoint := platformgeo.Point{Latitude: 35.1, Longitude: 112.1}
	hitCell := platformgeo.LeafCellID(hitPoint)
	missCell := platformgeo.LeafCellID(missPoint)
	seedUsersForRange(t, ctx, tx, hitCell, missCell, totalRows, hitRows)
	mustExec(t, ctx, tx, `
INSERT INTO groups (group_id, name, location_name, latitude, longitude, is_searchable, owner_id, s2_cell_token, s2_cell_id, tags, deleted_at, created_at, last_activity_at)
SELECT 'g_' || gs, 'g_' || gs, 'loc', 35.0, 112.0,
       CASE WHEN gs % 10 = 0 THEN FALSE ELSE TRUE END,
       'u_1', '', CASE WHEN gs <= $2 THEN $1::bigint ELSE $3::bigint END,
       '{}', NULL, NOW() - (gs || ' seconds')::interval, NOW()
FROM generate_series(1, $4) gs
`, hitCell, hitRows, missCell, totalRows)
	mustExec(t, ctx, tx, `
INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id, created_at)
SELECT 'm_' || gs, 'u_1', 'm', 'loc', 35.0, 112.0,
       CASE WHEN gs % 10 = 0 THEN 'private' ELSE 'public' END,
       CASE WHEN gs <= $2 THEN $1::bigint ELSE $3::bigint END,
       NOW() - (gs || ' seconds')::interval
FROM generate_series(1, $4) gs
`, hitCell, hitRows, missCell, totalRows)
	mustExec(t, ctx, tx, `ANALYZE users`)
	mustExec(t, ctx, tx, `ANALYZE groups`)
	mustExec(t, ctx, tx, `ANALYZE moments`)
	return hitCell, missCell
}

func seedFollowingOnlyExplainFixture(t *testing.T, ctx context.Context, tx pgx.Tx) (int64, error) {
	t.Helper()
	hitPoint := platformgeo.Point{Latitude: 35.0, Longitude: 112.0}
	missPoint := platformgeo.Point{Latitude: 35.1, Longitude: 112.1}
	hitCell := platformgeo.LeafCellID(hitPoint)
	missCell := platformgeo.LeafCellID(missPoint)
	mustExec(t, ctx, tx, `
INSERT INTO users (user_id, nickname, location_name, latitude, longitude, s2_cell_id)
VALUES
  ('author_1', 'author 1', 'hit', 35.0, 112.0, $1),
  ('author_2', 'author 2', 'miss', 35.1, 112.1, $2),
  ('viewer_1', 'viewer', 'viewer', 35.0, 112.0, $1)
`, hitCell, missCell)
	mustExec(t, ctx, tx, `
INSERT INTO user_relations (user_id, target_user_id, is_following)
VALUES ('viewer_1', 'author_2', TRUE)
`)
	mustExec(t, ctx, tx, `
INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id)
VALUES
  ('moment_hit_public', 'author_1', 'hit', 'hit', 35.0, 112.0, 'public', $1),
  ('moment_follow_only_public', 'author_2', 'follow-only', 'miss', 35.1, 112.1, 'public', $2)
`, hitCell, missCell)
	mustExec(t, ctx, tx, `ANALYZE users`)
	mustExec(t, ctx, tx, `ANALYZE user_relations`)
	mustExec(t, ctx, tx, `ANALYZE moments`)
	return missCell, nil
}

func seedGroupsPredicateCornerCaseFixture(t *testing.T, ctx context.Context, tx pgx.Tx) (int64, int64) {
	t.Helper()
	hitPoint := platformgeo.Point{Latitude: 35.0, Longitude: 112.0}
	missPoint := platformgeo.Point{Latitude: 35.1, Longitude: 112.1}
	hitCell := platformgeo.LeafCellID(hitPoint)
	missCell := platformgeo.LeafCellID(missPoint)
	seedUsersForRange(t, ctx, tx, hitCell, missCell, 600, 60)
	mustExec(t, ctx, tx, `
INSERT INTO groups (group_id, name, location_name, latitude, longitude, is_searchable, owner_id, s2_cell_token, s2_cell_id, tags, deleted_at, created_at, last_activity_at)
SELECT 'g_live_' || gs, 'g_live_' || gs, 'loc', 35.0, 112.0, TRUE, 'u_1', '', $1::bigint, '{}', NULL, NOW(), NOW()
FROM generate_series(1, 120) gs
`, hitCell)
	mustExec(t, ctx, tx, `
INSERT INTO groups (group_id, name, location_name, latitude, longitude, is_searchable, owner_id, s2_cell_token, s2_cell_id, tags, deleted_at, created_at, last_activity_at)
SELECT 'g_hidden_' || gs, 'g_hidden_' || gs, 'loc', 35.0, 112.0, FALSE, 'u_1', '', $1::bigint, '{}', NULL, NOW(), NOW()
FROM generate_series(1, 200) gs
`, hitCell)
	mustExec(t, ctx, tx, `
INSERT INTO groups (group_id, name, location_name, latitude, longitude, is_searchable, owner_id, s2_cell_token, s2_cell_id, tags, deleted_at, created_at, last_activity_at)
SELECT 'g_deleted_' || gs, 'g_deleted_' || gs, 'loc', 35.0, 112.0, TRUE, 'u_1', '', $1::bigint, '{}', NOW(), NOW(), NOW()
FROM generate_series(1, 200) gs
`, hitCell)
	mustExec(t, ctx, tx, `ANALYZE groups`)
	return hitCell, missCell
}

func seedMomentsPrivacySkewFixture(t *testing.T, ctx context.Context, tx pgx.Tx) (int64, int64) {
	t.Helper()
	hitPoint := platformgeo.Point{Latitude: 35.0, Longitude: 112.0}
	missPoint := platformgeo.Point{Latitude: 35.1, Longitude: 112.1}
	hitCell := platformgeo.LeafCellID(hitPoint)
	missCell := platformgeo.LeafCellID(missPoint)
	seedUsersForRange(t, ctx, tx, hitCell, missCell, 100, 50)
	mustExec(t, ctx, tx, `
INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id, created_at)
SELECT 'm_pub_' || gs, 'u_1', 'pub', 'loc', 35.0, 112.0, 'public', $1::bigint, NOW()
FROM generate_series(1, 20) gs
`, hitCell)
	mustExec(t, ctx, tx, `
INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id, created_at)
SELECT 'm_protected_' || gs, 'u_1', 'protected', 'loc', 35.0, 112.0, 'friends', $1::bigint, NOW()
FROM generate_series(1, 900) gs
`, hitCell)
	mustExec(t, ctx, tx, `
INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id, created_at)
SELECT 'm_private_' || gs, 'u_1', 'private', 'loc', 35.0, 112.0, 'private', $1::bigint, NOW()
FROM generate_series(1, 80) gs
`, hitCell)
	mustExec(t, ctx, tx, `ANALYZE moments`)
	return hitCell, missCell
}

func seedUsersForRange(t *testing.T, ctx context.Context, tx pgx.Tx, hitCell, missCell int64, totalRows, hitRows int) {
	t.Helper()
	mustExec(t, ctx, tx, `
INSERT INTO users (user_id, nickname, location_name, latitude, longitude, s2_cell_id)
SELECT 'u_' || gs, 'u_' || gs, 'loc', 35.0, 112.0,
       CASE WHEN gs <= $2 THEN $1::bigint ELSE $3::bigint END
FROM generate_series(1, $4) gs
`, hitCell, hitRows, missCell, totalRows)
	mustExec(t, ctx, tx, `
INSERT INTO users (user_id, nickname, location_name, latitude, longitude, s2_cell_id)
VALUES ('viewer_1', 'viewer', 'viewer', 35.0, 112.0, $1)
`, hitCell)
	mustExec(t, ctx, tx, `
INSERT INTO user_relations (user_id, target_user_id, is_following)
VALUES ('viewer_1', 'u_2', TRUE)
`)
	mustExec(t, ctx, tx, `ANALYZE users`)
	mustExec(t, ctx, tx, `ANALYZE user_relations`)
}

func mapGroupsExplainSQL() string {
	return `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.location_name, ''), g.latitude, g.longitude
FROM groups g
WHERE g.is_searchable = TRUE
  AND g.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)`
}

func buildSearchMomentsExplainSQL() string {
	return `SELECT moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level FROM moments WHERE privacy_level <> 'private' AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi) WHERE s2_cell_id BETWEEN t.lo AND t.hi)`
}

func feedAllMomentsExplainSQL() string {
	return `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.moment_id AND ml.user_id = $3),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $3),
       EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $3 AND ur.is_following = TRUE)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND (
    EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
            WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)
    OR EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $3 AND ur.is_following = TRUE)
  )`
}

func buildGroupSearchExplainSQL() string {
	distanceExpr := "CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(latitude - $1) / 2), 2) + COS(RADIANS($1)) * COS(RADIANS(latitude)) * POWER(SIN(RADIANS(longitude - $2) / 2), 2))) AS INTEGER)"
	return "SELECT * FROM (" +
		"SELECT g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.is_searchable, g.max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}')" +
		", (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = g.group_id) AS members_count" +
		", " + distanceExpr + " AS distance_meters, 0 AS name_match_score" +
		" FROM groups g WHERE g.deleted_at IS NULL AND g.is_searchable = true" +
		" AND EXISTS (SELECT 1 FROM unnest($3::bigint[], $4::bigint[]) AS t(lo, hi) WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)" +
		") search WHERE 1=1 AND distance_meters <= $5 ORDER BY name_match_score DESC, members_count DESC, group_id ASC LIMIT $6"
}

func mustExec(t *testing.T, ctx context.Context, tx pgx.Tx, sql string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("exec sql: %v\nSQL:%s", err, sql)
	}
}

func getenvOrFail(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("%s not set", key)
	}
	return value
}
