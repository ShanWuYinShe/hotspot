package mapmodule

import (
	"context"
	"sort"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
)

type Service struct{ repo Repository }

const heatmapViewportThresholdMeters = 50000

func NewService() *Service { return NewServiceWithRepo(NewPostgresRepository(nil)) }
func NewServiceWithRepo(repo Repository) *Service {
	if repo == nil {
		repo = NewPostgresRepository(nil)
	}
	return &Service{repo: repo}
}

func (s *Service) Canvas(ctx context.Context, req MapCanvasRequest) (MapCanvasResponse, error) {
	query, err := newCanvasQuery(req)
	if err != nil {
		return MapCanvasResponse{}, err
	}
	if query.SpanMeters > heatmapViewportThresholdMeters {
		// 聚合粒度必须随可视跨度变化：query.Level 继续沿用搜索预过滤口径，
		// 而热力桶改用与可视尺度匹配的层级，避免大陆/全球视口仍显示城市尺度小簇。
		query.Level = heatmapAggregationLevel(query.SpanMeters)
		heatmap, err := s.repo.Heatmap(ctx, query)
		if err != nil {
			observability.L().Error("map.canvas heatmap failed", "error", err)
			return MapCanvasResponse{}, err
		}
		if len(heatmap.Items) == 0 {
			return MapCanvasResponse{Mode: MapCanvasModeEmpty}, nil
		}
		// 就近合并：SQL 层按 S2 祖先格聚合后，质心彼此靠近的相邻桶（常被
		// 格边界切开）再按距离阈值合并——目标桶边长 ≈ 视口跨度的六分之一，
		// 与 heatmapAggregationLevel 的取级口径一致，故阈值取 SpanMeters/6。
		heatmap.Items = mergeHeatBucketsProximity(heatmap.Items, query.SpanMeters/6)
		return MapCanvasResponse{Mode: MapCanvasModeHeatmap, Heatmap: &heatmap}, nil
	}

	hotspots, err := s.repo.Hotspots(ctx, query)
	if err != nil {
		observability.L().Error("map.canvas hotspots failed", "error", err)
		return MapCanvasResponse{}, err
	}
	if len(hotspots.Items) == 0 {
		return MapCanvasResponse{Mode: MapCanvasModeEmpty}, nil
	}
	return MapCanvasResponse{Mode: MapCanvasModeHotspots, Hotspots: &hotspots}, nil
}

// mergeHeatBucketsProximity 就近合并热力桶：以桶质心为节点做贪心聚类——
// 按（计数降序, token 升序）确定性遍历，每个未分配桶吸入其「种子桶」
// 质心距离 ≤ thresholdMeters 的所有桶（种子质心固定不随吸收漂移，保证
// 同一输入产出同一结果）。合并桶继承种子桶的 cell_token/level，计数与
// 三分类拆分相加，代表坐标为全体成员的计数加权重心。相邻但分属不同
// S2 祖先格的小簇由此合并为一个统计桶，消除格边切分的「统计数据碎片」。
func mergeHeatBucketsProximity(items []MapHeatBucket, thresholdMeters float64) []MapHeatBucket {
	if thresholdMeters <= 0 || len(items) <= 1 {
		return items
	}
	sorted := make([]MapHeatBucket, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].TotalCount != sorted[j].TotalCount {
			return sorted[i].TotalCount > sorted[j].TotalCount
		}
		return sorted[i].CellToken < sorted[j].CellToken
	})
	type cluster struct {
		bucket                 MapHeatBucket
		latSum, lngSum, weight float64
	}
	clusters := make([]cluster, 0, len(sorted))
	for _, item := range sorted {
		point := geo.Point{Latitude: item.Location.Latitude, Longitude: item.Location.Longitude}
		merged := false
		for ci := range clusters {
			seed := geo.Point{Latitude: clusters[ci].bucket.Location.Latitude, Longitude: clusters[ci].bucket.Location.Longitude}
			if float64(geo.DistanceMeters(seed, point)) <= thresholdMeters {
				c := &clusters[ci]
				c.bucket.TotalCount += item.TotalCount
				c.bucket.Breakdown = mergeHeatBreakdown(c.bucket.Breakdown, item.Breakdown)
				w := float64(item.TotalCount)
				c.latSum += point.Latitude * w
				c.lngSum += point.Longitude * w
				c.weight += w
				merged = true
				break
			}
		}
		if !merged {
			w := float64(item.TotalCount)
			clusters = append(clusters, cluster{
				bucket: item,
				latSum: point.Latitude * w,
				lngSum: point.Longitude * w,
				weight: w,
			})
		}
	}
	out := make([]MapHeatBucket, 0, len(clusters))
	for _, c := range clusters {
		b := c.bucket
		if c.weight > 0 {
			b.Location = GeoPoint{Latitude: c.latSum / c.weight, Longitude: c.lngSum / c.weight}
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalCount != out[j].TotalCount {
			return out[i].TotalCount > out[j].TotalCount
		}
		return out[i].CellToken < out[j].CellToken
	})
	return out
}

// mergeHeatBreakdown 合并三分类拆分：任一侧非 nil 的维度保留（计数相加，
// nil 视为 0），两侧皆 nil 的维度维持 nil（对齐 spec：user/group/moment 为
// number | null，仅 data_type 过滤出单类型时其余维度可为 null）。
func mergeHeatBreakdown(dst, src *MapHeatBreakdown) *MapHeatBreakdown {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}
	add := func(a, b *int) *int {
		switch {
		case a == nil:
			return b
		case b == nil:
			return a
		default:
			v := *a + *b
			return &v
		}
	}
	dst.User = add(dst.User, src.User)
	dst.Group = add(dst.Group, src.Group)
	dst.Moment = add(dst.Moment, src.Moment)
	return dst
}

// heatmapAggregationLevel 根据热力视口跨度选择 S2 聚合层级。
// 目标是让热力桶边长约占视口跨度的六分之一：相邻 S2 层级边长相差一倍，
// 因此跨度每翻一番就降低一级。大尺度下相邻小簇会被合并到同一祖先桶；
// 小尺度热力视口（刚超过 50km 阈值）仍使用 level 10，保持街道/城区细节。
func heatmapAggregationLevel(spanMeters float64) int {
	switch {
	case spanMeters > 16000000:
		return 1
	case spanMeters > 8000000:
		return 2
	case spanMeters > 4000000:
		return 3
	case spanMeters > 2000000:
		return 4
	case spanMeters > 1000000:
		return 5
	case spanMeters > 500000:
		return 6
	case spanMeters > 250000:
		return 7
	case spanMeters > 125000:
		return 8
	case spanMeters > 62500:
		return 9
	default:
		return 10
	}
}

func newCanvasQuery(req MapCanvasRequest) (canvasQuery, error) {
	bounds := geo.Bounds{North: req.Viewport.Bounds.North, South: req.Viewport.Bounds.South, East: req.Viewport.Bounds.East, West: req.Viewport.Bounds.West}
	spanMeters := geo.ApproxViewportSpanMeters(bounds)
	query := canvasQuery{
		Bounds:     bounds,
		Center:     geo.Center(bounds),
		SpanMeters: spanMeters,
		Level:      geo.SuggestedLevel(spanMeters),
		DataType:   normalizeMapDataType(""),
		Limit:      normalizeMapLimit(0),
	}
	if req.Filter != nil {
		query.DataType = normalizeMapDataType(req.Filter.DataType)
		if req.Filter.TimeRange != nil {
			if req.Filter.TimeRange.StartTime != "" {
				parsed, err := time.Parse(time.RFC3339Nano, req.Filter.TimeRange.StartTime)
				if err != nil {
					return canvasQuery{}, err
				}
				query.StartTime = parsed.UTC()
			}
			if req.Filter.TimeRange.EndTime != "" {
				parsed, err := time.Parse(time.RFC3339Nano, req.Filter.TimeRange.EndTime)
				if err != nil {
					return canvasQuery{}, err
				}
				query.EndTime = parsed.UTC()
			}
		}
	}
	if req.Pagination != nil {
		query.Limit = normalizeMapLimit(req.Pagination.Limit)
		query.Cursor = req.Pagination.Cursor
	}
	return query, nil
}
