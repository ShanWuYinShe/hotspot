package admin

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresIT_StatsGeoAggregation 真库口径验证：S2 桶聚合、无效位置剔除、
// 时间窗过滤、三实体 breakdown（与内存实现同语义，防双实现漂移）。
// 真库可能存在其它测试的存量数据，断言全部采用「播种前基线 + 增量」口径，
// 并把种子坐标放在偏远网格（南大西洋），避免与既有桶重叠。
func TestPostgresIT_StatsGeoAggregation(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no tx queryable")
		}
		repo := NewPostgresRepository(txFactory)
		// 播种前基线（存量不受本测试影响，仅作增量参照）。
		base, err := repo.StatsGeo(ctx, geoStatsQuery{Level: 8, Limit: 10})
		if err != nil {
			t.Fatalf("baseline StatsGeo: %v", err)
		}

		seed := func(sql string, args ...any) {
			t.Helper()
			if err := q.Exec(ctx, sql, args...); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
		// A 簇（南大西洋）：2 用户 + 1 现存动态 + 1 40 天前动态；B 簇：1 群组。
		leafA1 := geo.LeafCellID(geo.Point{Latitude: -62.20, Longitude: -58.60})
		leafA2 := geo.LeafCellID(geo.Point{Latitude: -62.21, Longitude: -58.61})
		leafB := geo.LeafCellID(geo.Point{Latitude: -63.50, Longitude: -57.00})
		nameA := "IT_TEST_南大西洋A"
		nameB := "IT_TEST_南大西洋B"
		seed(`INSERT INTO users (user_id, nickname, location_name, latitude, longitude, s2_cell_id) VALUES
			('it_stats_u1', '海岛一', $2, -62.20, -58.60, $1),
			('it_stats_u2', '海岛二', $2, -62.21, -58.61, $3)`, leafA1, nameA, leafA2)
		// 无有效位置（s2_cell_id=0）：计入 totals、不参与地理排名。
		seed(`INSERT INTO users (user_id, nickname, location_name, latitude, longitude) VALUES ('it_stats_unoloc', '无位置', '', 0, 0)`)
		seed(`INSERT INTO groups (group_id, name, location_name, latitude, longitude, s2_cell_id, owner_id)
			VALUES ('it_stats_g1', '海岛群', $2, -63.50, -57.00, $1, 'it_stats_u1')`, leafB, nameB)
		seed(`INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, s2_cell_id, created_at)
			VALUES ('it_stats_m_now', 'it_stats_u1', '现存动态', $2, -62.20, -58.60, $1, NOW())`, leafA1, nameA)
		seed(`INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, s2_cell_id, created_at)
			VALUES ('it_stats_m_old', 'it_stats_u1', '旧动态', $2, -62.20, -58.60, $1, NOW() - INTERVAL '40 days')`, leafA1, nameA)

		findRegion := func(resp AdminGeoStatsResponse, latMin, latMax float64) *AdminRegionRankItem {
			for i := range resp.Regions {
				if resp.Regions[i].Center.Latitude > latMin && resp.Regions[i].Center.Latitude < latMax {
					return &resp.Regions[i]
				}
			}
			return nil
		}
		findPlace := func(resp AdminGeoStatsResponse, name string) *AdminPlaceRankItem {
			for i := range resp.Places {
				if resp.Places[i].LocationName == name {
					return &resp.Places[i]
				}
			}
			return nil
		}

		// 不限时间：A 簇 = 2 用户 + 2 动态 = 4；B 簇 = 1 群组。
		resp, err := repo.StatsGeo(ctx, geoStatsQuery{Level: 8, Limit: 100})
		if err != nil {
			t.Fatalf("StatsGeo: %v", err)
		}
		if resp.Totals.User != base.Totals.User+3 || resp.Totals.Group != base.Totals.Group+1 || resp.Totals.Moment != base.Totals.Moment+2 {
			t.Fatalf("unexpected totals delta: %+v base %+v", resp.Totals, base.Totals)
		}
		regionA := findRegion(resp, -63, -61)
		if regionA == nil {
			t.Fatalf("A region missing: %+v", resp.Regions)
		}
		if regionA.TotalCount != 4 || regionA.Breakdown.User != 2 || regionA.Breakdown.Moment != 2 || regionA.Breakdown.Group != 0 {
			t.Fatalf("unexpected A breakdown: %+v", regionA)
		}
		regionB := findRegion(resp, -64.5, -62.5)
		if regionB == nil {
			t.Fatalf("B region missing: %+v", resp.Regions)
		}
		if regionB.TotalCount != 1 || regionB.Breakdown.Group != 1 {
			t.Fatalf("unexpected B breakdown: %+v", regionB)
		}
		placeA := findPlace(resp, nameA)
		if placeA == nil || placeA.TotalCount != 4 || placeA.Breakdown.User != 2 || placeA.Breakdown.Moment != 2 {
			t.Fatalf("unexpected place A: %+v", placeA)
		}
		if findPlace(resp, nameB) == nil {
			t.Fatalf("place B missing: %+v", resp.Places)
		}

		// 时间窗（最近 7 天）：40 天前动态被排除；totals 不受时间窗影响。
		windowEnd := time.Now().UTC()
		windowStart := windowEnd.AddDate(0, 0, -7)
		resp, err = repo.StatsGeo(ctx, geoStatsQuery{Level: 8, Limit: 100, StartTime: windowStart, EndTime: windowEnd})
		if err != nil {
			t.Fatalf("StatsGeo windowed: %v", err)
		}
		if resp.Totals.Moment != base.Totals.Moment+2 || resp.Totals.User != base.Totals.User+3 {
			t.Fatalf("totals should ignore time window: %+v", resp.Totals)
		}
		regionA = findRegion(resp, -63, -61)
		if regionA == nil || regionA.Breakdown.Moment != 1 || regionA.TotalCount != 3 {
			t.Fatalf("windowed A region: %+v", regionA)
		}
		placeA = findPlace(resp, nameA)
		if placeA == nil || placeA.TotalCount != 3 {
			t.Fatalf("windowed place A: %+v", placeA)
		}
	})
}
