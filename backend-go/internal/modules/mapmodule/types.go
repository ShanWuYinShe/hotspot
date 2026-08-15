package mapmodule

import "hotspot/backend-go/internal/platform/types"

type Bounds struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
}

type MapViewport struct {
	Bounds Bounds `json:"bounds"`
}

type TimeRangeParams struct {
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

type MapCanvasFilter struct {
	DataType  string           `json:"data_type,omitempty"`
	TimeRange *TimeRangeParams `json:"time_range,omitempty"`
}

type PaginationParams = types.PaginationParams

type MapCanvasRequest struct {
	Viewport   MapViewport       `json:"viewport"`
	Filter     *MapCanvasFilter  `json:"filter,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type GeoPoint = types.GeoPoint

type MapHeatBreakdown struct {
	User   *int `json:"user"`
	Group  *int `json:"group"`
	Moment *int `json:"moment"`
}

type MapHeatBucket struct {
	CellToken  string            `json:"cell_token"`
	Level      int               `json:"level"`
	Location   GeoPoint          `json:"location"`
	TotalCount int               `json:"total_count"`
	Breakdown  *MapHeatBreakdown `json:"breakdown"`
}

type MapHeatCanvas struct {
	Items []MapHeatBucket `json:"items"`
}

type MapHotspotItem struct {
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Title      string                 `json:"title"`
	Subtitle   *string                `json:"subtitle"`
	CoverURL   *string                `json:"cover_url"`
	Location   types.RelativeLocation `json:"location"`
	CreatedAt  *string                `json:"created_at"`
}

type MapHotspotsCanvas struct {
	Items      []MapHotspotItem `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

type MapCanvasResponse struct {
	Heatmap  *MapHeatCanvas     `json:"heatmap,omitempty"`
	Hotspots *MapHotspotsCanvas `json:"hotspots,omitempty"`
}
