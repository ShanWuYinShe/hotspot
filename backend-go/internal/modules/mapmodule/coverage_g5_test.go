package mapmodule

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
)

// TestModuleMountsPublicCanvasRoute 装配 NewModule 并挂载路由：地图画布只有
// public 节点，Mount 为 no-op；MountPublic 必须命中 /map/canvas（非法请求体走
// 10000 校验错误，不依赖数据库）。
func TestModuleMountsPublicCanvasRoute(t *testing.T) {
	module := NewModule(nil)
	if module == nil || module.handler == nil {
		t.Fatalf("expected wired module, got %#v", module)
	}

	apiRouter := chi.NewRouter()
	module.Mount(apiRouter) // 公开只读模块：/api/v1 无路由

	publicRouter := chi.NewRouter()
	module.MountPublic(publicRouter)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/map/canvas", strings.NewReader("{"))
	publicRouter.ServeHTTP(rr, req)
	assertMapCode(t, rr, httpx.CodeInvalidRequest)

	// 合法请求会真正进入 handler -> Service -> repo（nil factory 走
	// ErrNotImplemented），证明路由绑定的确实是 Canvas handler。
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/map/canvas", strings.NewReader("{\"viewport\":{\"bounds\":{\"north\":0.2,\"south\":0,\"east\":0.2,\"west\":0}}}"))
	publicRouter.ServeHTTP(rr, req)
	assertMapCode(t, rr, httpx.CodeInternalUnavailable)
}

// TestNewServiceFallsBackToPostgresRepository 覆盖默认构造与 nil repo 回退分支。
func TestNewServiceFallsBackToPostgresRepository(t *testing.T) {
	svc := NewService()
	if svc == nil || svc.repo == nil {
		t.Fatalf("NewService returned unusable service: %#v", svc)
	}
	if _, ok := svc.repo.(*PostgresRepository); !ok {
		t.Fatalf("NewService repo = %T, want *PostgresRepository", svc.repo)
	}
	fallback := NewServiceWithRepo(nil)
	if fallback == nil || fallback.repo == nil {
		t.Fatalf("NewServiceWithRepo(nil) returned unusable service: %#v", fallback)
	}
}

// TestCanvasRejectsInvalidEndTimeFormat 覆盖 handler 只给 end_time 且格式非法
// 的分支（start_time 缺失时也必须拒绝）。
func TestCanvasRejectsInvalidEndTimeFormat(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&handlerMapRepo{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/canvas", strings.NewReader("{\"viewport\":{\"bounds\":{\"north\":1,\"south\":0,\"east\":1,\"west\":0}},\"filter\":{\"time_range\":{\"end_time\":\"not-a-time\"}}}"))

	handler.Canvas(rr, req)
	assertMapCode(t, rr, httpx.CodeInvalidRequest)
}

// TestAggregateHeatBucketsCombinesTypesAndTruncates 单元覆盖按 S2 网格聚合的
// 分类计数、并列排序与 >200 桶截断分支（该函数仅保留给内存/回退实现使用）。
func TestAggregateHeatBucketsCombinesTypesAndTruncates(t *testing.T) {
	items := []mapCandidate{
		{ObjectType: "user", ObjectID: "u1", Location: GeoPoint{Latitude: 35.0, Longitude: 112.0}},
		{ObjectType: "group", ObjectID: "g1", Location: GeoPoint{Latitude: 35.00001, Longitude: 112.00001}},
		{ObjectType: "moment", ObjectID: "m1", Location: GeoPoint{Latitude: 35.00002, Longitude: 112.00002}},
		// 另一个格：仅 moment，用于并列计数时按 cell_token 排序（level-20 单格约 0.15m）。
		{ObjectType: "moment", ObjectID: "m2", Location: GeoPoint{Latitude: 35.001, Longitude: 112.001}},
	}
	buckets := aggregateHeatBuckets(items, 20)
	if len(buckets) != 2 {
		t.Fatalf("buckets = %d, want 2", len(buckets))
	}
	var combined *MapHeatBucket
	for i := range buckets {
		if buckets[i].TotalCount == 3 {
			combined = &buckets[i]
		}
	}
	if combined == nil {
		t.Fatalf("missing combined bucket: %#v", buckets)
	}
	if combined.Breakdown == nil || combined.Breakdown.User == nil || *combined.Breakdown.User != 1 ||
		combined.Breakdown.Group == nil || *combined.Breakdown.Group != 1 ||
		combined.Breakdown.Moment == nil || *combined.Breakdown.Moment != 1 {
		t.Fatalf("unexpected breakdown: %#v", combined.Breakdown)
	}

	// 201 个不同 level-20 网格（单格约 0.15m）触发 >200 截断。
	many := make([]mapCandidate, 0, 201)
	for i := 0; i < 201; i++ {
		many = append(many, mapCandidate{
			ObjectType: "user",
			ObjectID:   "u",
			Location:   GeoPoint{Latitude: 30.0 + float64(i)*0.01, Longitude: 110.0 + float64(i)*0.01},
		})
	}
	truncated := aggregateHeatBuckets(many, 20)
	if len(truncated) != 200 {
		t.Fatalf("truncated buckets = %d, want 200", len(truncated))
	}
}

// TestCompareMapCandidatesTieBreakers 覆盖排序比较器的全部并列降级分支：
// 距离相同 -> 类型 -> 创建时间 -> 对象 ID -> 完全相等。
func TestCompareMapCandidatesTieBreakers(t *testing.T) {
	created := "2026-03-03T00:00:00Z"
	other := "2026-03-02T00:00:00Z"
	base := mapCandidate{Distance: 10, ObjectType: "user", CreatedAt: created, ObjectID: "b"}

	cases := []struct {
		name  string
		left  mapCandidate
		right mapCandidate
		want  int
	}{
		{"distance less", base, mapCandidate{Distance: 20}, -10},
		{"type less", mapCandidate{Distance: 10, ObjectType: "group"}, mapCandidate{Distance: 10, ObjectType: "moment"}, -1},
		{"type greater", mapCandidate{Distance: 10, ObjectType: "moment"}, mapCandidate{Distance: 10, ObjectType: "group"}, 1},
		{"created newer first", base, mapCandidate{Distance: 10, ObjectType: "user", CreatedAt: other, ObjectID: "b"}, -1},
		{"created older later", mapCandidate{Distance: 10, ObjectType: "user", CreatedAt: other, ObjectID: "b"}, base, 1},
		{"id less", mapCandidate{Distance: 10, ObjectType: "user", CreatedAt: created, ObjectID: "a"}, base, -1},
		{"id greater", mapCandidate{Distance: 10, ObjectType: "user", CreatedAt: created, ObjectID: "c"}, base, 1},
		{"equal", base, base, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := compareMapCandidates(tc.left, tc.right); got != tc.want {
				t.Fatalf("compareMapCandidates = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestFindMapCursorStartBranches 覆盖游标定位的三个退出分支：命中后一页、
// 位于全部候选之前、以及落在候选之后（返回 len）。
func TestFindMapCursorStartBranches(t *testing.T) {
	items := []mapCandidate{
		{Distance: 1, ObjectType: "user", CreatedAt: "2026-03-03T00:00:00Z", ObjectID: "a"},
		{Distance: 2, ObjectType: "user", CreatedAt: "2026-03-03T00:00:00Z", ObjectID: "b"},
	}
	if got := findMapCursorStart(items, mapCursor{Distance: 1, Type: "user", CreatedAt: "2026-03-03T00:00:00Z", ID: "a"}); got != 1 {
		t.Fatalf("exact match start = %d, want 1", got)
	}
	if got := findMapCursorStart(items, mapCursor{Distance: 0, Type: "user", CreatedAt: "2026-03-03T00:00:00Z", ID: "a"}); got != 0 {
		t.Fatalf("before-all start = %d, want 0", got)
	}
	if got := findMapCursorStart(items, mapCursor{Distance: 99, Type: "user", CreatedAt: "2026-03-03T00:00:00Z", ID: "a"}); got != len(items) {
		t.Fatalf("after-all start = %d, want %d", got, len(items))
	}
}

// TestDecodeMapCursorRejectsMissingIdentity 覆盖解码后 type/id 缺失的非法游标。
func TestDecodeMapCursorRejectsMissingIdentity(t *testing.T) {
	token, err := mapCursorCodec.Encode(mapCursor{})
	if err != nil {
		t.Fatalf("encode empty cursor: %v", err)
	}
	if _, err := decodeMapCursor(token); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("decodeMapCursor(missing identity) err = %v, want ErrInvalidCursor", err)
	}
	if decoded, err := decodeMapCursor(""); err != nil || decoded != nil {
		t.Fatalf("decodeMapCursor(empty) = (%v, %v), want (nil, nil)", decoded, err)
	}
}

// TestFallbackHeatBucketCenterUsesFallbackForInvalidToken 覆盖 token 无法解出
// 格心（返回零值）时回退到调用方坐标的分支。
func TestFallbackHeatBucketCenterUsesFallbackForInvalidToken(t *testing.T) {
	fallback := GeoPoint{Latitude: 35.5, Longitude: 112.5}
	if got := fallbackHeatBucketCenter("not-a-cell-token", fallback); got != fallback {
		t.Fatalf("fallback center = %#v, want %#v", got, fallback)
	}
	validToken := geo.CellTokenAtLevel(geo.Point{Latitude: 35.0, Longitude: 112.0}, 12)
	got := fallbackHeatBucketCenter(validToken, fallback)
	if got == fallback || (got == GeoPoint{}) {
		t.Fatalf("valid token center = %#v, want S2 cell center", got)
	}
}

// TestPostgresRepositoryHeatmapAppliesTimeWindow 覆盖 Heatmap 在 moments 分支
// 把零值/非零时间窗转成 nil / *time.Time 的两条路径。
func TestPostgresRepositoryHeatmapAppliesTimeWindow(t *testing.T) {
	shift := geo.ParentShift(12)
	leaf := geo.LeafCellID(geo.Point{Latitude: 35.0, Longitude: 112.0})
	bucketCell := int64((uint64(leaf) >> uint(shift)) << uint(shift))
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryHeatmapMoments: {{bucketCell, int64(1), 35.0, 112.0}},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	resp, err := repo.Heatmap(context.Background(), canvasQuery{
		Bounds:    testMapBounds(),
		Center:    testMapCenter(),
		Level:     12,
		DataType:  "moment",
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].TotalCount != 1 {
		t.Fatalf("unexpected heatmap: %#v", resp.Items)
	}
}

// TestPostgresRepositoryPropagatesGroupMomentQueryErrors 覆盖 Hotspots 分组/瞬间
// 候选查询的错误上抛分支。
func TestPostgresRepositoryPropagatesGroupMomentQueryErrors(t *testing.T) {
	for _, query := range []string{queryMapCanvasGroups, queryMapCanvasMoments} {
		query := query
		t.Run(query, func(t *testing.T) {
			repo := NewPostgresRepository(newMapFactory(&mapBackendStub{queryErrs: map[string]error{query: errors.New("boom")}}))
			_, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 10, DataType: "all"})
			if err == nil || err.Error() != "boom" {
				t.Fatalf("expected query error, got %v", err)
			}
		})
	}
}

// TestPostgresRepositoryPropagatesHeatmapQueryErrors 覆盖热力图片段查询的错误
// 上抛分支（user/group/moment 三条查询）。
func TestPostgresRepositoryPropagatesHeatmapQueryErrors(t *testing.T) {
	for _, query := range []string{queryHeatmapUsers, queryHeatmapGroups, queryHeatmapMoments} {
		query := query
		t.Run(query, func(t *testing.T) {
			repo := NewPostgresRepository(newMapFactory(&mapBackendStub{queryErrs: map[string]error{query: errors.New("boom")}}))
			_, err := repo.Heatmap(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Level: 12, DataType: "all"})
			if err == nil || err.Error() != "boom" {
				t.Fatalf("expected query error, got %v", err)
			}
		})
	}
}
