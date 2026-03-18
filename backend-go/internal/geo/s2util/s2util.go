package s2util

import (
	"math"

	"github.com/golang/geo/s2"

	"backend-go/internal/types"
)

const (
	// EarthRadiusMeters 近似地球半径，用于距离相关计算。
	EarthRadiusMeters = 6371010.0

	DefaultMaxCells = 200
)

// BoundsToRegion 将规范中的 Bounds 转为 S2 LatLngRect。
func BoundsToRegion(b types.Bounds) s2.Rect {
	sw := s2.LatLngFromDegrees(b.South, b.West)
	ne := s2.LatLngFromDegrees(b.North, b.East)
	rect := s2.RectFromLatLng(sw)
	return rect.AddPoint(ne)
}

// PickLevelByBounds 根据视口对角线估算一个合适的 S2 level。
// 这是一个经验型 heuristic，后续可根据实际数据分布调整。
func PickLevelByBounds(b types.Bounds) int {
	latSpan := math.Abs(b.North - b.South)
	lngSpan := math.Abs(b.East - b.West)
	span := math.Max(latSpan, lngSpan)

	switch {
	case span > 60:
		return 6
	case span > 10:
		return 8
	case span > 1:
		return 10
	case span > 0.1:
		return 12
	default:
		return 14
	}
}

// CoveringCells 在指定 level 上，用 RegionCoverer 覆盖 region，并返回 CellID 列表。
func CoveringCells(region s2.Region, level, maxCells int) []s2.CellID {
	if maxCells <= 0 {
		maxCells = DefaultMaxCells
	}
	coverer := &s2.RegionCoverer{
		MinLevel: level,
		MaxLevel: level,
		MaxCells: maxCells,
	}
	return coverer.Covering(region)
}

// LatLngToCellID 计算指定 level 的 S2CellID。
func LatLngToCellID(lat, lng float64, level int) s2.CellID {
	ll := s2.LatLngFromDegrees(lat, lng)
	return s2.CellIDFromLatLng(ll).Parent(level)
}

// CellIDToToken 将 CellID 转为字符串 token，对外返回时使用。
func CellIDToToken(id s2.CellID) types.S2CellToken {
	return types.S2CellToken(id.ToToken())
}

// CellIDCenter 返回 S2 单元格中心点的 GeoPoint。
func CellIDCenter(id s2.CellID) types.GeoPoint {
	ll := id.LatLng()
	return types.GeoPoint{
		Lat: ll.Lat.Degrees(),
		Lng: ll.Lng.Degrees(),
	}
}
