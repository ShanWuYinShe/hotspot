package mapmodule

// 本文件是**测试专用占位实现，不保证语义等价**（审查 P2-26）：
// Heatmap/Hotspots 返回按 S2 覆盖生成的固定形状数据，而非真实聚合结果。
// module.go 的装配恒使用 NewPostgresRepository，DB 未配置时也不会降级到
// 本实现（无降级路径），故生产不可达；它的存在仅为让不依赖 DB 的单元测试
// 能构造 Service。禁止在任何降级/生产路径引用——需要真实语义请扩展 Postgres
// 实现的测试桩（persistence.Queryable 注入）而不是本类型。
import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/types"
)

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) Heatmap(_ context.Context, query canvasQuery) (MapHeatCanvas, error) {
	covering := geo.CoveringTokens(query.Bounds, query.Level, 32)
	items := make([]MapHeatBucket, 0, len(covering))
	for i, token := range covering {
		if i >= 5 {
			break
		}
		items = append(items, MapHeatBucket{
			CellToken: token,
			Level:     query.Level,
			// 与 postgres 实现同口径：代表坐标取 S2 格心（spec api_spec_public.md
			// 「MapHeatBucket.location」），保证双实现可互换。
			Location:   fallbackHeatBucketCenter(token, GeoPoint{Latitude: query.Center.Latitude, Longitude: query.Center.Longitude}),
			TotalCount: 3,
			Breakdown:  &MapHeatBreakdown{User: intPtr(1), Group: intPtr(1), Moment: intPtr(1)},
		})
	}
	return MapHeatCanvas{Items: items}, nil
}

func (r *InMemoryRepository) Hotspots(_ context.Context, query canvasQuery) (MapHotspotsCanvas, error) {
	point := query.Center
	createdAt := time.Now().UTC().Format(time.RFC3339)
	distance := float64(geo.DistanceMeters(point, point))
	return MapHotspotsCanvas{Items: []MapHotspotItem{{
		ObjectType: "moment",
		ObjectID:   fmt.Sprintf("moment_map_%d", query.Level),
		Title:      "Map hotspot",
		Location: types.RelativeLocation{
			Point:          types.GeoPoint{Latitude: point.Latitude, Longitude: point.Longitude},
			LocationName:   "",
			Distance:       &distance,
			BearingDegrees: intPtrToFloat64(0),
		},
		CreatedAt: &createdAt,
	}}}, nil
}

func intPtr(v int) *int { return &v }

func intPtrToFloat64(v int) *float64 {
	f := float64(v)
	return &f
}
