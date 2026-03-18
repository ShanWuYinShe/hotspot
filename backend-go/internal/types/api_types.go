package types

// 与 spec/api_spec_types.md 和各模块 spec 对应的 Go 端类型定义。
// 目前包含地图与认证模块所需的部分类型，后续可按需扩展。

// GeoPoint 对应规范中的 GeoPoint，使用 WGS-84 坐标。
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ---- Map 模块相关类型 ----

type MapDataType string

const (
	MapDataTypeUser   MapDataType = "user"
	MapDataTypeGroup  MapDataType = "group"
	MapDataTypeMoment MapDataType = "moment"
	MapDataTypeAll    MapDataType = "all"
)

// Bounds 与 spec/api_spec_map.md 中的 Bounds 一致。
type Bounds struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
}

// MapViewport 仅由 bounds 决定，不暴露 zoom。
type MapViewport struct {
	Bounds Bounds `json:"bounds"`
}

// TimeRangeParams 按规范表示时间过滤区间（UTC ISO8601 字符串）。
type TimeRangeParams struct {
	StartTime *string `json:"start_time,omitempty"`
	EndTime   *string `json:"end_time,omitempty"`
}

// MapHeatBreakdown 为某个网格内按类型拆分的计数。
type MapHeatBreakdown struct {
	User   *int64 `json:"user,omitempty"`
	Group  *int64 `json:"group,omitempty"`
	Moment *int64 `json:"moment,omitempty"`
}

// S2CellToken 对应规范中的 S2CellToken。
type S2CellToken string

// MapHeatBucket 表示一个 S2 网格的聚合结果。
type MapHeatBucket struct {
	CellToken S2CellToken       `json:"cell_token"`
	Level     int               `json:"level"`
	Location  GeoPoint          `json:"location"`
	Total     int64             `json:"total_count"`
	Breakdown *MapHeatBreakdown `json:"breakdown,omitempty"`
}

// MapHeatCanvas 为热力统计结果集合。
type MapHeatCanvas struct {
	Items []MapHeatBucket `json:"items"`
}

type HotspotObjectType string

const (
	HotspotObjectUser   HotspotObjectType = "user"
	HotspotObjectGroup  HotspotObjectType = "group"
	HotspotObjectMoment HotspotObjectType = "moment"
)

// MapHotspotItem 表示一个具体的热点对象。
type MapHotspotItem struct {
	ObjectType     HotspotObjectType `json:"object_type"`
	ObjectID       string            `json:"object_id"`
	Title          string            `json:"title"`
	Subtitle       *string           `json:"subtitle,omitempty"`
	CoverURL       *string           `json:"cover_url,omitempty"`
	Location       GeoPoint          `json:"location"`
	CreatedAt      *string           `json:"created_at,omitempty"`
	DistanceMeters *float64          `json:"distance_meters,omitempty"`
}

// MapHotspotsCanvas 为热点明细结果集合。
type MapHotspotsCanvas struct {
	Items      []MapHotspotItem `json:"items"`
	NextCursor *string          `json:"next_cursor,omitempty"`
}

// MapCanvasFilter 复用 data_type / time_range。
type MapCanvasFilter struct {
	DataType  *MapDataType     `json:"data_type,omitempty"`
	TimeRange *TimeRangeParams `json:"time_range,omitempty"`
}

// PaginationParams 来自基础 spec，这里只按字段名对齐，具体约束见 api_spec_base。
type PaginationParams struct {
	Cursor *string `json:"cursor,omitempty"`
	Limit  *int32  `json:"limit,omitempty"`
}

// MapCanvasRequest 对应 POST /api/v1/map/canvas 的请求。
type MapCanvasRequest struct {
	Viewport   MapViewport       `json:"viewport"`
	Filter     *MapCanvasFilter  `json:"filter,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// MapCanvasResponse 对应响应，只会出现 heatmap 或 hotspots 之一。
type MapCanvasResponse struct {
	Heatmap  *MapHeatCanvas     `json:"heatmap,omitempty"`
	Hotspots *MapHotspotsCanvas `json:"hotspots,omitempty"`
}

// ---- Auth 模块相关类型 ----

type ClientType string

const (
	ClientTypeWeb ClientType = "web"
	ClientTypeApp ClientType = "app"
)

// Web 端 data 为空对象。
type AuthTokenPairWeb struct{}

// App 端 Token 对。
type AuthTokenPairApp struct {
	UserID               string `json:"user_id"`
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
}

// RegisterRequest 对应 /auth/register。
type RegisterRequest struct {
	LoginID    string     `json:"login_id"`
	Nickname   string     `json:"nickname"`
	Password   string     `json:"password"`
	ClientType ClientType `json:"client_type"`
}

// LoginRequest 对应 /auth/login。
type LoginRequest struct {
	LoginID    string     `json:"login_id"`
	Password   string     `json:"password"`
	ClientType ClientType `json:"client_type"`
}

// GuestRequest 对应 /auth/guest。
type GuestRequest struct {
	Nickname   string     `json:"nickname"`
	ClientType ClientType `json:"client_type"`
}

// RefreshTokenRequest 对应 /auth/refresh_token。
type RefreshTokenRequest struct {
	ClientType   ClientType `json:"client_type"`
	RefreshToken *string    `json:"refresh_token,omitempty"`
}

// AuthStatus 对应 /auth/check_status 响应中的 data。
type AuthStatus struct {
	IsValid   bool    `json:"is_valid"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// RequestPasswordResetRequest 对应 /auth/request_password_reset。
type RequestPasswordResetRequest struct {
	LoginID    string     `json:"login_id"`
	ClientType ClientType `json:"client_type"`
}

// ResetPasswordRequest 对应 /auth/reset_password。
type ResetPasswordRequest struct {
	LoginID          string     `json:"login_id"`
	VerificationCode string     `json:"verification_code"`
	NewPassword      string     `json:"new_password"`
	ClientType       ClientType `json:"client_type"`
}

// ChangePasswordRequest 对应 /auth/me/change_password。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// SessionsRequest 对应 /auth/me/sessions/list 请求体（空对象）。
type SessionsRequest struct{}

// SessionItem 对应会话列表中的一项。
type SessionItem struct {
	SessionID    string  `json:"session_id"`
	DeviceType   *string `json:"device_type,omitempty"`
	DeviceName   *string `json:"device_name,omitempty"`
	IPAddress    *string `json:"ip_address,omitempty"`
	IPLocation   *string `json:"ip_location,omitempty"`
	CreatedAt    string  `json:"created_at"`
	LastActiveAt string  `json:"last_active_at"`
	IsCurrent    bool    `json:"is_current"`
}

// RevokeSessionRequest 对应 /auth/me/sessions/revoke。
type RevokeSessionRequest struct {
	SessionID string `json:"session_id"`
}

// RevokeAllResult 对应 /auth/me/sessions/revoke_all 响应 data。
type RevokeAllResult struct {
	RevokedCount int `json:"revoked_count"`
}
