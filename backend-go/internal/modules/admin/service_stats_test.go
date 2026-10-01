package admin

import (
	"context"
	"testing"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

func groupsmoduleGroupInfo(id, name string) groupsmodule.GroupInfo {
	return groupsmodule.GroupInfo{GroupID: id, Name: name}
}

func groupsmoduleLocation(lat, lng float64, name string) groupsmodule.Location {
	var loc groupsmodule.Location
	loc.Point.Latitude = lat
	loc.Point.Longitude = lng
	loc.LocationName = name
	return loc
}

// seedStatsFixtures 整体替换内存夹具为受控数据集：
// 北京 ×2（同簇同名"北京市朝阳区"）、上海 ×1、广州群 ×1、无位置用户 ×1。
// 期望：区域 = 北京(2) > 上海(1)/广州(1)；地点 = 北京市朝阳区(2) > 其余(1)。
func seedStatsFixtures(repo *InMemoryRepository) {
	repo.adminUsers = map[string]AdminUserItem{
		"u_bj1": {
			User:     UserInfo{UserID: "u_bj1", Nickname: "北京一"},
			Location: &usersmodule.RelativeLocation{Point: types.GeoPoint{Latitude: 39.9042, Longitude: 116.4074}, LocationName: "北京市朝阳区"},
		},
		"u_bj2": {
			User:     UserInfo{UserID: "u_bj2", Nickname: "北京二"},
			Location: &usersmodule.RelativeLocation{Point: types.GeoPoint{Latitude: 39.9100, Longitude: 116.4100}, LocationName: "北京市朝阳区"},
		},
		"u_sh": {
			User:     UserInfo{UserID: "u_sh", Nickname: "上海"},
			Location: &usersmodule.RelativeLocation{Point: types.GeoPoint{Latitude: 31.2304, Longitude: 121.4737}, LocationName: "上海市黄浦区"},
		},
		"u_noloc": {User: UserInfo{UserID: "u_noloc", Nickname: "无位置"}},
	}
	repo.groups = map[string]AdminGroupItem{
		"g_gz": {
			Group:    groupsmoduleGroupInfo("g_gz", "广州群"),
			Location: groupsmoduleLocation(23.1291, 113.2644, "广州市天河区"),
		},
	}
}

func TestServiceStatsGeoRejectsNoAuth(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, apiErr, err := svc.StatsGeo(context.Background(), AdminGeoStatsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeUnauthorized {
		t.Fatalf("expected httpx.CodeUnauthorized, got %v", apiErr)
	}
}

func TestServiceStatsGeoRejectsNonAdmin(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, apiErr, err := svc.StatsGeo(adminCtx("nobody"), AdminGeoStatsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected httpx.CodeForbidden, got %v", apiErr)
	}
}

func TestServiceStatsGeoRequiresStatsPermission(t *testing.T) {
	svc, repo, _, _ := newTestService()
	// 管理员但未授予 admin.stats.read（无角色 = 无权限）。
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin_limited"})
	_, apiErr, err := svc.StatsGeo(adminCtx("admin_limited"), AdminGeoStatsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected insufficient_admin_permission forbidden, got %v", apiErr)
	}
}

func TestServiceStatsGeoValidatesParams(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	cases := []AdminGeoStatsRequest{
		{Level: intPtr(3)},  // 低于下限
		{Level: intPtr(13)}, // 高于上限
		{TimeRange: &StatsTimeRangeParams{StartTime: "2026-01-02T00:00:00Z", EndTime: "2026-01-01T00:00:00Z"}}, // 倒置
		{TimeRange: &StatsTimeRangeParams{StartTime: "not-a-time"}},                                            // 非法时间
		{Limit: intPtr(-1)}, // 负数
	}
	for i, req := range cases {
		_, apiErr, err := svc.StatsGeo(adminCtx("admin1"), req)
		if err != nil {
			t.Fatalf("case %d unexpected error: %v", i, err)
		}
		if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
			t.Fatalf("case %d expected CodeInvalidRequest, got %v", i, apiErr)
		}
	}
}

func TestServiceStatsGeoSuccess(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	seedStatsFixtures(repo)

	resp, apiErr, err := svc.StatsGeo(adminCtx("admin1"), AdminGeoStatsRequest{Limit: intPtr(2)})
	if err != nil || apiErr != nil {
		t.Fatalf("unexpected failure: err=%v apiErr=%v", err, apiErr)
	}
	// 全站存量（含无位置用户）。
	if resp.Totals.User != 4 || resp.Totals.Group != 1 || resp.Totals.Moment != 0 || resp.Totals.Total != 5 {
		t.Fatalf("unexpected totals: %+v", resp.Totals)
	}
	// Top-2 区域：北京（2 用户）第一；上海/广州各 1 并列取 key 升序。
	if len(resp.Regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(resp.Regions))
	}
	top := resp.Regions[0]
	if top.TotalCount != 2 || top.Breakdown.User != 2 || top.Breakdown.Group != 0 || top.Breakdown.Moment != 0 {
		t.Fatalf("unexpected top region: %+v", top)
	}
	if top.CellToken == "" || top.Level != statsDefaultLevel {
		t.Fatalf("unexpected region identity: %+v", top)
	}
	if top.Center.Latitude == 0 && top.Center.Longitude == 0 {
		t.Fatalf("expected non-zero center: %+v", top)
	}
	if resp.Regions[1].TotalCount != 1 {
		t.Fatalf("unexpected second region: %+v", resp.Regions[1])
	}
	// Top-2 地点："北京市朝阳区" ×2 第一。
	if len(resp.Places) != 2 {
		t.Fatalf("expected 2 places, got %d", len(resp.Places))
	}
	if resp.Places[0].LocationName != "北京市朝阳区" || resp.Places[0].TotalCount != 2 || resp.Places[0].Breakdown.User != 2 {
		t.Fatalf("unexpected top place: %+v", resp.Places[0])
	}
	if resp.Places[1].TotalCount != 1 {
		t.Fatalf("unexpected second place: %+v", resp.Places[1])
	}
}

func TestServiceStatsGeoDefaultLimit(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	seedStatsFixtures(repo)
	resp, apiErr, err := svc.StatsGeo(adminCtx("admin1"), AdminGeoStatsRequest{})
	if err != nil || apiErr != nil {
		t.Fatalf("unexpected failure: err=%v apiErr=%v", err, apiErr)
	}
	// 默认 limit=20 > 簇数 3，全量返回。
	if len(resp.Regions) != 3 || len(resp.Places) != 3 {
		t.Fatalf("expected all 3 regions/places, got %d/%d", len(resp.Regions), len(resp.Places))
	}
	if resp.Totals.User != 4 || resp.Totals.Group != 1 {
		t.Fatalf("unexpected totals: %+v", resp.Totals)
	}
}

func intPtr(v int) *int { return &v }
