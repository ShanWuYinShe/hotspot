package service

import (
	"context"
	"time"

	"backend-go/internal/geo/s2util"
	"backend-go/internal/repository"
	"backend-go/internal/types"
)

// MapService 定义地图模块的核心能力。
type MapService interface {
	Canvas(ctx context.Context, req types.MapCanvasRequest) (*types.MapCanvasResponse, error)
}

type mapService struct {
	geoRepo repository.GeoRepository
}

func NewMapService(geoRepo repository.GeoRepository) MapService {
	return &mapService{geoRepo: geoRepo}
}

// Canvas 根据视口和过滤条件，返回热力统计或热点明细。
// 当前实现优先返回热力统计（heatmap），hotspots 分支将在后续迭代中补全。
func (s *mapService) Canvas(ctx context.Context, req types.MapCanvasRequest) (*types.MapCanvasResponse, error) {
	bounds := req.Viewport.Bounds
	region := s2util.BoundsToRegion(bounds)
	level := s2util.PickLevelByBounds(bounds)
	cells := s2util.CoveringCells(region, level, s2util.DefaultMaxCells)
	if len(cells) == 0 {
		return &types.MapCanvasResponse{}, nil
	}

	// 将 S2 CellID 转为 int64 供 Repository 查询预聚合统计。
	cellIDs := make([]int64, len(cells))
	for i, c := range cells {
		cellIDs[i] = int64(c)
	}

	// 暂时按“当天”聚合；后续可根据 TimeRangeParams 扩展多日聚合逻辑。
	date := time.Now().UTC().Format("2006-01-02")

	statsRows, err := s.geoRepo.AggregateS2Stats(ctx, date, int16(level), cellIDs)
	if err != nil {
		return nil, err
	}

	// 将统计结果放入 map，便于按 cellID 查找。
	statsByID := make(map[int64]repository.S2StatRow, len(statsRows))
	for _, row := range statsRows {
		statsByID[row.CellID] = row
	}

	buckets := make([]types.MapHeatBucket, 0, len(cells))
	for _, c := range cells {
		center := s2util.CellIDCenter(c)
		row, ok := statsByID[int64(c)]
		if !ok {
			// 没有预聚合数据时，total_count 视为 0，不填 breakdown。
			buckets = append(buckets, types.MapHeatBucket{
				CellToken: s2util.CellIDToToken(c),
				Level:     level,
				Location:  center,
				Total:     0,
			})
			continue
		}

		var total int64
		var breakdown *types.MapHeatBreakdown

		// 依据 data_type 决定 total 和是否返回 breakdown。
		if req.Filter != nil && req.Filter.DataType != nil {
			switch *req.Filter.DataType {
			case types.MapDataTypeUser:
				total = row.UserCount
			case types.MapDataTypeGroup:
				total = row.GroupCount
			case types.MapDataTypeMoment:
				total = row.MomentCount
			case types.MapDataTypeAll:
				fallthrough
			default:
				total = row.UserCount + row.GroupCount + row.MomentCount
			}
			// 当 data_type != "all" 时不强制返回 breakdown。
		} else {
			// 未指定 data_type 时，视为 all，并返回 breakdown。
			u := row.UserCount
			g := row.GroupCount
			m := row.MomentCount
			total = u + g + m
			breakdown = &types.MapHeatBreakdown{
				User:   &u,
				Group:  &g,
				Moment: &m,
			}
		}

		buckets = append(buckets, types.MapHeatBucket{
			CellToken: s2util.CellIDToToken(c),
			Level:     level,
			Location:  center,
			Total:     total,
			Breakdown: breakdown,
		})
	}

	return &types.MapCanvasResponse{
		Heatmap: &types.MapHeatCanvas{Items: buckets},
	}, nil
}
