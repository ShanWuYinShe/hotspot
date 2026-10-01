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

// AdminGeoStatsResponse 地理统计聚合快照（弱一致；无分页，恒返回 Top-N）。
type AdminGeoStatsResponse struct {
	GeneratedAt string                `json:"generated_at"`
	Totals      AdminGeoStatsTotals   `json:"totals"`
	Regions     []AdminRegionRankItem `json:"regions"`
	Places      []AdminPlaceRankItem  `json:"places"`
}

// geoStatsQuery Service→Repository 的内部查询参数（Level 已校验、时间已解析）。
type geoStatsQuery struct {
	Level     int
	StartTime time.Time // 零值 = 不限
	EndTime   time.Time // 零值 = 不限
	Limit     int
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
)
