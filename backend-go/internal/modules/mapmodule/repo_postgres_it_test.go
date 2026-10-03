package mapmodule

import (
	"context"
	"math"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func insertMapTestUser(t *testing.T, qdb persistence.QueryableDB, userID, nickname string, lat, lng float64, privacyLevel string, locVisible, searchable bool) {
	t.Helper()
	s2CellID := geo.LeafCellID(geo.Point{Latitude: lat, Longitude: lng})
	queryUser := `INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, s2_cell_id, presence_status, last_online, created_at, status)
VALUES ($1, $2, 'https://example.com/avatar.png', 'normal', '', '', '', '', 'Loc', $3, $4, $5, 'online', NOW(), NOW(), 'normal')`
	if err := qdb.Exec(context.Background(), queryUser, userID, nickname, lat, lng, s2CellID); err != nil {
		t.Fatalf("insertMapTestUser: %v", err)
	}
	querySettings := `INSERT INTO user_settings (user_id, privacy_level, location_visible, searchable)
VALUES ($1, $2, $3, $4)`
	if err := qdb.Exec(context.Background(), querySettings, userID, privacyLevel, locVisible, searchable); err != nil {
		t.Fatalf("insertMapTestUser settings: %v", err)
	}
}

func insertMapTestGroup(t *testing.T, qdb persistence.QueryableDB, groupID, name, ownerID string, lat, lng float64, isSearchable bool) {
	t.Helper()
	s2CellID := geo.LeafCellID(geo.Point{Latitude: lat, Longitude: lng})
	query := `INSERT INTO groups (group_id, name, avatar_url, description, location_name, latitude, longitude, is_searchable, owner_id, s2_cell_id, created_at, last_activity_at)
VALUES ($1, $2, 'https://example.com/group.png', '', 'Loc', $3, $4, $5, $6, $7, NOW(), NOW())`
	if err := qdb.Exec(context.Background(), query, groupID, name, lat, lng, isSearchable, ownerID, s2CellID); err != nil {
		t.Fatalf("insertMapTestGroup: %v", err)
	}
}

func insertMapTestMoment(t *testing.T, qdb persistence.QueryableDB, momentID, userID, desc string, lat, lng float64, privacyLevel string, isDeleted bool, createdAt time.Time) {
	t.Helper()
	s2CellID := geo.LeafCellID(geo.Point{Latitude: lat, Longitude: lng})
	query := `INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, is_deleted, s2_cell_id, created_at)
VALUES ($1, $2, $3, 'Loc', $4, $5, $6, $7, $8, $9)`
	if err := qdb.Exec(context.Background(), query, momentID, userID, desc, lat, lng, privacyLevel, isDeleted, s2CellID, createdAt); err != nil {
		t.Fatalf("insertMapTestMoment: %v", err)
	}
}

func TestPostgresIT_MapCanvas_Hotspots_S2SpatialFiltering(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb := txFactory.DB.(persistence.QueryableDB)
		repo := NewPostgresRepository(txFactory)

		// 独特测试参考中心坐标，避免受预置城市种子数据影响
		center := geo.Point{Latitude: 42.1234, Longitude: 88.5678}
		bounds := geo.BoundsAround(center, 5000) // 5km 视口

		// 1. 视口内部有效数据 (~300m ~ 1km)
		insertMapTestUser(t, qdb, "u_near_valid", "NearUser", 42.1260, 88.5678, "public", true, true)
		insertMapTestGroup(t, qdb, "g_near_valid", "NearGroup", "u_near_valid", 42.1270, 88.5678, true)
		insertMapTestMoment(t, qdb, "m_near_valid", "u_near_valid", "NearMoment", 42.1250, 88.5678, "public", false, time.Now().UTC())

		// 2. 远距离数据 (上海 ~3000km 外，必须被 S2 视口过滤)
		insertMapTestUser(t, qdb, "u_far_shanghai", "FarUser", 31.2304, 121.4737, "public", true, true)
		insertMapTestGroup(t, qdb, "g_far_shanghai", "FarGroup", "u_far_shanghai", 31.2310, 121.4740, true)
		insertMapTestMoment(t, qdb, "m_far_shanghai", "u_far_shanghai", "FarMoment", 31.2300, 121.4730, "public", false, time.Now().UTC())

		// 3. 视口内部但业务属性不可见的数据 (空间命中但必须被权限/状态过滤)
		insertMapTestUser(t, qdb, "u_near_hidden", "HiddenUser", 42.1255, 88.5678, "public", false, true)
		insertMapTestGroup(t, qdb, "g_near_unsearchable", "HiddenGroup", "u_near_valid", 42.1256, 88.5678, false)
		insertMapTestMoment(t, qdb, "m_near_private", "u_near_valid", "PrivateMoment", 42.1257, 88.5678, "private", false, time.Now().UTC())
		insertMapTestMoment(t, qdb, "m_near_deleted", "u_near_valid", "DeletedMoment", 42.1258, 88.5678, "public", true, time.Now().UTC())

		// 场景 A: DataType = "all" 查询全量热点
		respAll, err := repo.Hotspots(ctx, canvasQuery{
			Bounds:   bounds,
			Center:   center,
			Limit:    20,
			DataType: "all",
		})
		if err != nil {
			t.Fatalf("Hotspots(all): %v", err)
		}

		if len(respAll.Items) != 3 {
			t.Fatalf("expected exactly 3 items in hotspots, got %d: %#v", len(respAll.Items), respAll.Items)
		}

		foundIDs := make(map[string]bool)
		for _, item := range respAll.Items {
			foundIDs[item.ObjectID] = true
			if item.Location.Distance == nil || *item.Location.Distance > 5000 {
				t.Fatalf("item %s distance invalid or exceeds 5km viewport: %#v", item.ObjectID, item.Location.Distance)
			}
		}

		if !foundIDs["u_near_valid"] || !foundIDs["g_near_valid"] || !foundIDs["m_near_valid"] {
			t.Fatalf("expected near items missing from hotspots: %#v", foundIDs)
		}
		if foundIDs["u_far_shanghai"] || foundIDs["g_far_shanghai"] || foundIDs["m_far_shanghai"] {
			t.Fatalf("far items leaked through S2 spatial filtering: %#v", foundIDs)
		}
		if foundIDs["u_near_hidden"] || foundIDs["g_near_unsearchable"] || foundIDs["m_near_private"] || foundIDs["m_near_deleted"] {
			t.Fatalf("invalid/private items leaked through filtering: %#v", foundIDs)
		}

		// 验证排序：按距离升序
		for i := 1; i < len(respAll.Items); i++ {
			if *respAll.Items[i].Location.Distance < *respAll.Items[i-1].Location.Distance {
				t.Fatalf("hotspots not sorted by distance: items[%d].dist=%.1f < items[%d].dist=%.1f",
					i, *respAll.Items[i].Location.Distance, i-1, *respAll.Items[i-1].Location.Distance)
			}
		}

		// 场景 B: 单独查询 DataType = "moment"
		respMoment, err := repo.Hotspots(ctx, canvasQuery{
			Bounds:   bounds,
			Center:   center,
			Limit:    20,
			DataType: "moment",
		})
		if err != nil {
			t.Fatalf("Hotspots(moment): %v", err)
		}
		if len(respMoment.Items) != 1 || respMoment.Items[0].ObjectID != "m_near_valid" {
			t.Fatalf("expected only m_near_valid for moment query, got %#v", respMoment.Items)
		}
	})
}

func TestPostgresIT_MapCanvas_Heatmap_S2Aggregation(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb := txFactory.DB.(persistence.QueryableDB)
		repo := NewPostgresRepository(txFactory)

		center := geo.Point{Latitude: 42.1234, Longitude: 88.5678}
		bounds := geo.BoundsAround(center, 5000)

		insertMapTestUser(t, qdb, "u_heat_1", "User1", 42.1240, 88.5670, "public", true, true)
		insertMapTestGroup(t, qdb, "g_heat_1", "Group1", "u_heat_1", 42.1242, 88.5672, true)
		insertMapTestMoment(t, qdb, "m_heat_1", "u_heat_1", "Moment1", 42.1241, 88.5671, "public", false, time.Now().UTC())

		// 远距离数据（不应进入热力图）
		insertMapTestUser(t, qdb, "u_heat_far", "FarUser", 31.2304, 121.4737, "public", true, true)

		heatmap, err := repo.Heatmap(ctx, canvasQuery{
			Bounds:   bounds,
			Center:   center,
			Level:    14,
			DataType: "all",
		})
		if err != nil {
			t.Fatalf("Heatmap: %v", err)
		}

		if len(heatmap.Items) == 0 {
			t.Fatal("expected non-empty heatmap buckets")
		}

		var totalCount int
		for _, bucket := range heatmap.Items {
			totalCount += bucket.TotalCount
			if bucket.Level != 14 {
				t.Fatalf("expected bucket level 14, got %d", bucket.Level)
			}
			if bucket.CellToken == "" {
				t.Fatal("expected non-empty cell token")
			}
			// 代表坐标必须是桶内成员的球面质心（spec api_spec_public.md
			// 「MapHeatBucket.location」），而不是随聚合层级变化的格心。
			// 三个测试点关于 (42.1241, 88.5671) 对称，算术平均即为期望质心。
			if math.Abs(bucket.Location.Latitude-42.1241) > 1e-9 || math.Abs(bucket.Location.Longitude-88.5671) > 1e-9 {
				t.Fatalf("expected bucket %s location near member centroid (42.1241, 88.5671), got %v", bucket.CellToken, bucket.Location)
			}
		}

		if totalCount != 3 {
			t.Fatalf("expected total count 3 in heatmap, got %d", totalCount)
		}

		topBucket := heatmap.Items[0]
		if topBucket.Breakdown == nil {
			t.Fatal("expected non-nil breakdown")
		}
		if topBucket.Breakdown.User == nil || *topBucket.Breakdown.User != 1 {
			t.Fatalf("expected Breakdown.User=1, got %#v", topBucket.Breakdown.User)
		}
		if topBucket.Breakdown.Group == nil || *topBucket.Breakdown.Group != 1 {
			t.Fatalf("expected Breakdown.Group=1, got %#v", topBucket.Breakdown.Group)
		}
		if topBucket.Breakdown.Moment == nil || *topBucket.Breakdown.Moment != 1 {
			t.Fatalf("expected Breakdown.Moment=1, got %#v", topBucket.Breakdown.Moment)
		}
	})
}
