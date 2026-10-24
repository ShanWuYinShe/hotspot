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
	trendsRows   [][]any
	rowsByQuery  map[string][][]any
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
	case queryAdminStatsTrends:
		rows = s.trendsRows
	default:
		if s.rowsByQuery != nil {
			rows = s.rowsByQuery[query]
		}
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
		case **string:
			if r.values[i] == nil {
				*d = nil
			} else {
				s := r.values[i].(string)
				*d = &s
			}
		case *time.Time:
			*d = r.values[i].(time.Time)
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
		case **string:
			if row[i] == nil {
				*d = nil
			} else {
				s := row[i].(string)
				*d = &s
			}
		case *time.Time:
			*d = row[i].(time.Time)
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

// 趋势：SQL 行 (day, user_n, group_n, moment_n) → 补零展开窗口序列。
func TestPostgresStatsGeoTrendsMapping(t *testing.T) {
	// stub 的 day 与下方 start 必须基于同一时区截断，否则跨 UTC 午夜时
	// stub 数据会落到错误的窗口槽位（时区边界 flake）。
	sh := time.FixedZone("UTC+8", 8*60*60)
	day := func(offsetDays int) time.Time {
		now := time.Now().In(sh)
		d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, sh)
		return d.AddDate(0, 0, offsetDays)
	}
	backend := &statsQueryStub{
		totalsRow:  []any{int64(0), int64(0), int64(0)},
		trendsRows: [][]any{{day(-1), int64(2), int64(1), int64(0)}, {day(0), int64(1), int64(0), int64(3)}},
	}
	repo := newStatsRepoWithStub(backend)
	nowSH := time.Now().In(sh)
	start := time.Date(nowSH.Year(), nowSH.Month(), nowSH.Day(), 0, 0, 0, 0, sh).AddDate(0, 0, -6)
	query := geoStatsQuery{TrendDays: 7, TrendStart: start, Level: 8, Limit: 20}
	resp, err := repo.StatsGeo(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Trends) != 7 {
		t.Fatalf("expected 7 points, got %d", len(resp.Trends))
	}
	// 倒数第 2 天（昨天）有 user=2 group=1；最后一天（今天）user=1 moment=3。
	yesterday := resp.Trends[5]
	if yesterday.User != 2 || yesterday.Group != 1 || yesterday.Moment != 0 {
		t.Fatalf("unexpected yesterday: %+v", yesterday)
	}
	today := resp.Trends[6]
	if today.User != 1 || today.Moment != 3 {
		t.Fatalf("unexpected today: %+v", today)
	}
	// 首日补零（夹具只有昨天/今天的数据）。
	if resp.Trends[0].User != 0 || resp.Trends[0].Group != 0 || resp.Trends[0].Moment != 0 {
		t.Fatalf("first day should be zero-filled: %+v", resp.Trends[0])
	}
	// 参数：上界固定为 SQL 内 now()（数据库时钟），只传 TrendStart 下界。
	args := backend.capturedArgs[queryAdminStatsTrends]
	if len(args) != 1 || args[0] == nil {
		t.Fatalf("unexpected trends args: %+v", args)
	}
}

// 下钻：三查询白名单分派 + 行值游标参数 + Next 编码往返。
func TestPostgresStatsGeoDrilldownMapping(t *testing.T) {
	created := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	backend := &statsQueryStub{
		totalsRow: []any{int64(0), int64(0), int64(0)},
		rowsByQuery: map[string][][]any{
			// limit+1 探测（审查 P2-11）：第 limit+1 行仅用于判断「还有下一页」，
			// 不进本页——stub 按「SQL 会返回的行」供数。
			queryAdminStatsDrilldownUsers: {
				{"user_b", "用户B", "北京市朝阳区", 39.9, 116.4, created},
				{"user_a", "用户A", nil, 39.91, 116.41, created.Add(-time.Minute)},
				{"user_probe", "探测行", nil, 39.92, 116.42, created.Add(-2 * time.Minute)},
			},
		},
	}
	repo := newStatsRepoWithStub(backend)
	leaf := geo.LeafCellID(geo.Point{Latitude: 39.9042, Longitude: 116.4074})
	bucket := leaf >> geo.CellLevelShift(8)
	page, err := repo.StatsGeoDrilldown(context.Background(), geoDrilldownQuery{
		Bucket: bucket, Level: 8, EntityType: "user", Limit: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 2 || page.Next == nil {
		t.Fatalf("unexpected page: %+v", page)
	}
	if page.Items[0].ObjectID != "user_b" || page.Items[0].Title != "用户B" {
		t.Fatalf("unexpected first item: %+v", page.Items[0])
	}
	if page.Items[0].LocationName == nil || *page.Items[0].LocationName != "北京市朝阳区" {
		t.Fatalf("unexpected location: %+v", page.Items[0])
	}
	if page.Items[1].LocationName != nil {
		t.Fatalf("expected null location name: %+v", page.Items[1])
	}
	// Next 游标可解码回载荷（service 层编码字符串→repo 解码的往返由服务测试覆盖,
	// 这里直接验证结构体）。
	if page.Next.ObjectID != "user_a" || page.Next.CreatedAt.IsZero() {
		t.Fatalf("unexpected cursor: %+v", page.Next)
	}
	// 参数顺序：(bucket, shift, start, end, cursorTime, cursorID, limit)。
	args := backend.capturedArgs[queryAdminStatsDrilldownUsers]
	var nilTime *time.Time
	var nilID *string
	if len(args) != 7 || args[0] != bucket || args[1] != geo.CellLevelShift(8) || args[2] != any(nilTime) || args[3] != any(nilTime) || args[6] != 3 {
		t.Fatalf("unexpected drilldown args: %+v", args)
	}
	if args[4] != any(nilTime) || args[5] != any(nilID) {
		t.Fatalf("first page cursor should be typed nil: %+v %+v", args[4], args[5])
	}
}

// 审查 P2-11 防回归：桶内条数恰好等于 limit 时，探测行不存在 →
// next_cursor 必须为 null（此前「满页即发」会产出翻到空页的幽灵游标，
// 违反「无更多数据 next_cursor=null」契约）。
func TestPostgresStatsGeoDrilldownExactPageHasNoNextCursor(t *testing.T) {
	created := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	backend := &statsQueryStub{
		totalsRow: []any{int64(0), int64(0), int64(0)},
		rowsByQuery: map[string][][]any{
			queryAdminStatsDrilldownUsers: {
				{"user_b", "用户B", nil, 39.9, 116.4, created},
				{"user_a", "用户A", nil, 39.91, 116.41, created.Add(-time.Minute)},
			},
		},
	}
	repo := newStatsRepoWithStub(backend)
	page, err := repo.StatsGeoDrilldown(context.Background(), geoDrilldownQuery{
		Bucket: 1, Level: 8, EntityType: "user", Limit: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 2 || page.Next != nil {
		t.Fatalf("exact-page must yield nil next_cursor: items=%d next=%+v", len(page.Items), page.Next)
	}
}

func TestPostgresStatsGeoDrilldownInvalidEntityType(t *testing.T) {
	repo := newStatsRepoWithStub(&statsQueryStub{totalsRow: []any{int64(0), int64(0), int64(0)}})
	if _, err := repo.StatsGeoDrilldown(context.Background(), geoDrilldownQuery{Bucket: 1, Level: 8, EntityType: "hack", Limit: 10}); err == nil {
		t.Fatal("expected error for non-whitelisted entity type")
	}
}
