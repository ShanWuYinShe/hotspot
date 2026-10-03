package admin

import "time"

// 统计分析（POST /admin/api/v1/stats/geo）请求/响应类型。
// 契约见 spec/api_spec_admin.md「统计分析（stats）API」。

// StatsTimeRangeParams 时间窗（各实体 created_at 维度，UTC ISO 8601）。
type StatsTimeRangeParams struct {
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

// AdminGeoStatsRequest 地理统计请求；所有字段可省略（省略 = 全部默认值）。
type AdminGeoStatsRequest struct {
	// Level S2 聚合层级 [4,12]；数字越大网格越细。null/缺省 = 8（≈20km，城市尺度）。
	Level *int `json:"level,omitempty"`
	// TimeRange 参与排名实体的创建时间窗；null/缺省 = 不限时间。
	TimeRange *StatsTimeRangeParams `json:"time_range,omitempty"`
	// Limit 排名条数；0/缺省 = 20，最大 100。
	Limit *int `json:"limit,omitempty"`
	// TrendDays 每日新增分布的天数 [1,90]；null/缺省/0 = 不返回趋势（trends 为空数组）。
	TrendDays *int `json:"trend_days,omitempty"`
}

// StatsEntityBreakdown 参与统计的三类实体拆分计数（恒在、恒为真值；
// 平台管理视角豁免客户端隐私过滤，见 spec/api_spec_authz.md「交叉规则」）。
type StatsEntityBreakdown struct {
	User   int `json:"user"`
	Group  int `json:"group"`
	Moment int `json:"moment"`
}

// AdminGeoStatsTotals 全站现存实体总数（不受时间窗影响）：用户未注销、
// 群组未解散、动态未删除。
type AdminGeoStatsTotals struct {
	User   int `json:"user"`
	Group  int `json:"group"`
	Moment int `json:"moment"`
	Total  int `json:"total"`
}

// AdminRegionRankItem 区域排名项：S2 网格桶（与公开地图 heatmap 同一
// token 体系），center 为网格中心点（代表坐标，供前端落点渲染）。
type AdminRegionRankItem struct {
	CellToken  string               `json:"cell_token"`
	Level      int                  `json:"level"`
	Center     GeoPoint             `json:"center"`
	TotalCount int                  `json:"total_count"`
	Breakdown  StatsEntityBreakdown `json:"breakdown"`
}

// AdminPlaceRankItem 热门地点排名项：按 location_name 精确文本聚合
// （同名合并；自由文本，不做行政区归一化）。
type AdminPlaceRankItem struct {
	LocationName string               `json:"location_name"`
	TotalCount   int                  `json:"total_count"`
	Breakdown    StatsEntityBreakdown `json:"breakdown"`
}

// AdminGeoStatsTrendPoint 每日新增分布单点：date 为 UTC+8 日界的 YYYY-MM-DD
// （中国站点无夏令时，固定偏移即日历日），计数按各实体 created_at 落在该日界内。
type AdminGeoStatsTrendPoint struct {
	Date   string `json:"date"`
	User   int    `json:"user"`
	Group  int    `json:"group"`
	Moment int    `json:"moment"`
}

// AdminGeoStatsResponse 地理统计聚合快照（弱一致；无分页，恒返回 Top-N）。
// Trends 仅在请求携带 trend_days 时有值（窗口逐日补零，无空洞），否则为空数组。
type AdminGeoStatsResponse struct {
	GeneratedAt string                    `json:"generated_at"`
	Totals      AdminGeoStatsTotals       `json:"totals"`
	Regions     []AdminRegionRankItem     `json:"regions"`
	Places      []AdminPlaceRankItem      `json:"places"`
	Trends      []AdminGeoStatsTrendPoint `json:"trends"`
}

// AdminGeoDrilldownRequest 区域下钻请求：按 cell_token + level 定位单个网格桶，
// 返回其中指定类型实体的明细（created_at 降序 + id tie-breaker，cursor keyset）。
type AdminGeoDrilldownRequest struct {
	CellToken  string                `json:"cell_token"`
	Level      *int                  `json:"level,omitempty"`
	EntityType string                `json:"entity_type"` // 'user' | 'group' | 'moment'
	TimeRange  *StatsTimeRangeParams `json:"time_range,omitempty"`
	Pagination *PaginationParams     `json:"pagination,omitempty"`
}

// AdminGeoDrilldownItem 下钻明细行：title 为昵称/群名/动态描述（动态截 60 字）。
type AdminGeoDrilldownItem struct {
	ObjectID     string  `json:"object_id"`
	Title        string  `json:"title"`
	LocationName *string `json:"location_name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	CreatedAt    string  `json:"created_at"`
}

// AdminGeoDrilldownPage 下钻分页响应（next_cursor 恒定 nullable，耗尽为 null）。
type AdminGeoDrilldownPage struct {
	Items      []AdminGeoDrilldownItem `json:"items"`
	NextCursor *string                 `json:"next_cursor"`
}

// geoDrilldownQuery Service→Repository 的内部下钻查询参数。
type geoDrilldownQuery struct {
	Bucket     int64
	Level      int
	EntityType string // user | group | moment
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Cursor     string
}

// geoDrilldownPage 仓储返回的一页下钻明细 + keyset 游标载荷。
type geoDrilldownPage struct {
	Items []AdminGeoDrilldownItem
	Next  *geoDrilldownCursor
}

// geoDrilldownCursor keyset 游标载荷（created_at 精度 RFC3339Nano，与列精度对齐）。
type geoDrilldownCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ObjectID  string    `json:"object_id"`
}

// geoStatsQuery Service→Repository 的内部查询参数（Level 已校验、时间已解析）。
type geoStatsQuery struct {
	Level      int
	StartTime  time.Time // 零值 = 不限
	EndTime    time.Time // 零值 = 不限
	Limit      int
	TrendDays  int       // 0 = 不返回趋势
	TrendStart time.Time // 趋势窗口起点（UTC+8 日界），TrendDays > 0 时有效
}

// geoBucketCount 一条聚合行（区域或地点维度共用）。
type geoBucketCount struct {
	Key    string // 区域维度为 S2 token；地点维度为 location_name
	Center GeoPoint
	Total  int
	User   int
	Group  int
	Moment int
}

// stats 模块级参数（聚合排名 Top-N，与通用分页 Default/MaxPageSize 语义不同）
const (
	statsDefaultLimit = 20
	statsMaxLimit     = 100
	statsDefaultLevel = 8
	statsMinLevel     = 4
	statsMaxLevel     = 12

	statsTrendMinDays = 1
	statsTrendMaxDays = 90
)
