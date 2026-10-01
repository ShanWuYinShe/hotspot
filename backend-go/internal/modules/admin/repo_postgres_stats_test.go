package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

// statsQueryStub 按查询常量精确分派的 Queryable 测试替身：
// 校验参数顺序与空值约定，返回 canned 行（映射层断言用）。
type statsQueryStub struct {
	totalsRow    []any
	regionsRows  [][]any
	placesRows   [][]any
	capturedArgs map[string][]any
	queryErr     map[string]error
}

func (s *statsQueryStub) Exec(context.Context, string, ...any) error { return nil }
func (s *statsQueryStub) Ping(context.Context) error                 { return nil }
func (s *statsQueryStub) Close() error                               { return nil }

func (s *statsQueryStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	return &statsRow{values: s.totalsRow}
}

func (s *statsQueryStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := s.queryErr[query]; err != nil {
		return nil, err
	}
	if s.capturedArgs == nil {
		s.capturedArgs = map[string][]any{}
	}
	s.capturedArgs[query] = args
	var rows [][]any
	switch query {
	case queryAdminStatsRegions:
		rows = s.regionsRows
	case queryAdminStatsPlaces:
		rows = s.placesRows
	}
	return &statsRows{values: rows}, nil
}

type statsRow struct {
	values []any
	err    error
}

func (r *statsRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *int64:
			*d = r.values[i].(int64)
		case *string:
			*d = r.values[i].(string)
		}
	}
	return nil
}

type statsRows struct {
	values [][]any
	idx    int
	err    error
}

func (r *statsRows) Next() bool { return r.idx < len(r.values) }
func (r *statsRows) Scan(dest ...any) error {
	row := r.values[r.idx]
	for i := range dest {
		switch d := dest[i].(type) {
		case *int64:
			*d = row[i].(int64)
		case *string:
			*d = row[i].(string)
		}
	}
	r.idx++
	return nil
}
func (r *statsRows) Close() error { return nil }
func (r *statsRows) Err() error   { return r.err }

func newStatsRepoWithStub(backend *statsQueryStub) *PostgresRepository {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return NewPostgresRepository(factory)
}

// 北京叶级 cell 的两个桶（模拟 SQL GROUP BY 产出）；一个负 int64（face 4/5）
// 桶验证符号保持与 token 还原。
func TestPostgresStatsGeoRowMapping(t *testing.T) {
	bjLeaf := geo.LeafCellID(geo.Point{Latitude: 39.9042, Longitude: 116.4074})
	nzLeaf := geo.LeafCellID(geo.Point{Latitude: -41.2865, Longitude: 174.7762}) // 惠灵顿（face 值大的半球）
	shift := int64(geo.CellLevelShift(8))
	backend := &statsQueryStub{
		totalsRow: []any{int64(3), int64(2), int64(1)},
		regionsRows: [][]any{
			{bjLeaf >> shift, int64(2), int64(2), int64(0), int64(0)},
			{nzLeaf >> shift, int64(3), int64(0), int64(1), int64(2)},
		},
		placesRows: [][]any{
			{"北京市朝阳区", int64(2), int64(2), int64(0), int64(0)},
			{"上海市黄浦区", int64(1), int64(1), int64(0), int64(0)},
		},
	}
	repo := newStatsRepoWithStub(backend)

	resp, err := repo.StatsGeo(context.Background(), geoStatsQuery{Level: 8, Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GeneratedAt == "" {
		t.Fatal("expected generated_at stamp")
	}
	// 全站存量直通。
	if resp.Totals.User != 3 || resp.Totals.Group != 2 || resp.Totals.Moment != 1 || resp.Totals.Total != 6 {
		t.Fatalf("unexpected totals: %+v", resp.Totals)
	}
	// 区域映射：token 与 geo 内存聚合路径一致、center 为网格中心。
	if len(resp.Regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(resp.Regions))
	}
	wantBJ := geo.CellTokenAtLevel(geo.Point{Latitude: 39.9042, Longitude: 116.4074}, 8)
	if resp.Regions[0].CellToken != wantBJ {
		t.Fatalf("unexpected beijing token: %q want %q", resp.Regions[0].CellToken, wantBJ)
	}
	if resp.Regions[0].Level != 8 || resp.Regions[0].TotalCount != 2 || resp.Regions[0].Breakdown.User != 2 {
		t.Fatalf("unexpected beijing item: %+v", resp.Regions[0])
	}
	center := geo.CellCenter(wantBJ)
	if resp.Regions[0].Center.Latitude != center.Latitude || resp.Regions[0].Center.Longitude != center.Longitude {
		t.Fatalf("unexpected center: %+v want %+v", resp.Regions[0].Center, center)
	}
	// 负 int64 桶（南半球 face）映射不漂移。
	wantNZ := geo.CellTokenAtLevel(geo.Point{Latitude: -41.2865, Longitude: 174.7762}, 8)
	if resp.Regions[1].CellToken != wantNZ {
		t.Fatalf("unexpected wellington token: %q want %q", resp.Regions[1].CellToken, wantNZ)
	}
	if resp.Regions[1].Breakdown.Group != 1 || resp.Regions[1].Breakdown.Moment != 2 {
		t.Fatalf("unexpected wellington breakdown: %+v", resp.Regions[1])
	}
	// 地点映射。
	if len(resp.Places) != 2 || resp.Places[0].LocationName != "北京市朝阳区" || resp.Places[0].TotalCount != 2 {
		t.Fatalf("unexpected places: %+v", resp.Places)
	}
	// 参数顺序：regions = (limit, shift, start, end)；时间为零值时传 nil。
	args := backend.capturedArgs[queryAdminStatsRegions]
	var nilTime *time.Time
	if len(args) != 4 || args[0] != 2 || args[1] != geo.CellLevelShift(8) || args[2] != any(nilTime) || args[3] != any(nilTime) {
		t.Fatalf("unexpected regions args: %+v", args)
	}
	args = backend.capturedArgs[queryAdminStatsPlaces]
	if len(args) != 3 || args[0] != 2 || args[1] != any(nilTime) || args[2] != any(nilTime) {
		t.Fatalf("unexpected places args: %+v", args)
	}
}

func TestPostgresStatsGeoQueryError(t *testing.T) {
	backend := &statsQueryStub{
		totalsRow: []any{int64(0), int64(0), int64(0)},
		queryErr:  map[string]error{queryAdminStatsRegions: errors.New("regions failed")},
	}
	repo := newStatsRepoWithStub(backend)
	if _, err := repo.StatsGeo(context.Background(), geoStatsQuery{Level: 8, Limit: 20}); err == nil {
		t.Fatal("expected regions query error to propagate")
	}
}
