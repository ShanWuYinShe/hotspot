package mapmodule

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

type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type MapCanvasRequest struct {
	Viewport   MapViewport       `json:"viewport"`
	Filter     *MapCanvasFilter  `json:"filter,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type MapHeatBreakdown struct {
	User   int `json:"user,omitempty"`
	Group  int `json:"group,omitempty"`
	Moment int `json:"moment,omitempty"`
}

type MapHeatBucket struct {
	CellToken  string            `json:"cell_token"`
	Level      int               `json:"level"`
	Location   GeoPoint          `json:"location"`
	TotalCount int               `json:"total_count"`
	Breakdown  *MapHeatBreakdown `json:"breakdown,omitempty"`
}

type MapHeatCanvas struct {
	Items []MapHeatBucket `json:"items"`
}

type MapHotspotItem struct {
	ObjectType     string   `json:"object_type"`
	ObjectID       string   `json:"object_id"`
	Title          string   `json:"title"`
	Subtitle       string   `json:"subtitle,omitempty"`
	CoverURL       string   `json:"cover_url,omitempty"`
	Location       GeoPoint `json:"location"`
	CreatedAt      string   `json:"created_at,omitempty"`
	DistanceMeters int      `json:"distance_meters,omitempty"`
}

type MapHotspotsCanvas struct {
	Items      []MapHotspotItem `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type MapCanvasResponse struct {
	Heatmap  *MapHeatCanvas     `json:"heatmap,omitempty"`
	Hotspots *MapHotspotsCanvas `json:"hotspots,omitempty"`
}
