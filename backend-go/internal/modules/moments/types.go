package moments

import "hotspot/backend-go/internal/platform/types"

type UserInfo = types.UserInfo

// Location 位置信息。请求侧对应 spec api_spec_types.md 的 Location（point +
// location_name）；响应侧作为 MomentItem.location 承载 RelativeLocation 形状：
// distance / bearing_degrees 必须由服务端基于 reference_point 计算并返回
// （spec api_spec_types.md「位置与地理信息」），客户端不得本地计算。
type Location struct {
	Point struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"point"`
	LocationName string `json:"location_name"`
	// Distance / BearingDegrees 相对参考点字段（spec api_spec_types.md RelativeLocation）：
	// 距离（米）与方位角 [0,360)；nil = 无参考点/不可见，恒输出 null。仅响应侧由
	// repo 在解析出参考点时填充，请求侧传入值在 Service 层剥离。
	Distance       *float64 `json:"distance"`
	BearingDegrees *float64 `json:"bearing_degrees"`
}

// fillRelativeLocation 填充响应侧相对位置的 distance / bearing_degrees
// （spec api_spec_types.md RelativeLocation：两字段均基于 reference_point 由服务端
// 计算；仅在请求解析出参考点的路径调用，其余路径保持 nil → 序列化为 null）。
func fillRelativeLocation(loc *Location, distanceMeters, bearingDegrees int) {
	if loc == nil {
		return
	}
	distance, bearing := float64(distanceMeters), float64(bearingDegrees)
	loc.Distance = &distance
	loc.BearingDegrees = &bearing
}

// RecommendationType 推荐理由枚举（spec api_spec_types.md RecommendationType）。
type RecommendationType string

const (
	RecommendationFriendLiked RecommendationType = "friend_liked"
	RecommendationNearby      RecommendationType = "nearby"
	RecommendationTrending    RecommendationType = "trending"
	RecommendationFollowing   RecommendationType = "following"
	RecommendationTagMatch    RecommendationType = "tag_match"
)

// Reaction mirrors spec api_spec_types.md Reaction (per-emoji aggregate with
// reacted_by_me). Defined here rather than imported from the reactions module
// so moments does not depend on reactions at the type level; the values are
// filled by a reactions.Service adapter at read time.
type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

// MyMomentState 是 Logged-in 查看者的收藏状态内部载体。契约要求 MomentItem
// 恒定输出 is_favorited 布尔（见 api_spec_types.md MomentItem），故响应不再把
// my_state 作为一个嵌套对象序列化；repo 的 LoadMyState 仍以 MyMomentState
// 返回，由 Service 中转为 item.IsFavorited。
type MyMomentState struct {
	IsFavorited bool `json:"is_favorited"`
}

type MomentImage struct {
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnail_url"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
	MimeType     *string `json:"mime_type"`
}

// MomentItem 是瞬间的公开固定形状（spec api_spec_types.md MomentItem）。
// 拥有者专属字段 privacy_level 不在此类型（见 OwnerMomentItem）；
// privacy_level 仅作 repo 内部存储/过滤载体，json:"-" 保证不对外输出。
type MomentItem struct {
	MomentID    string        `json:"moment_id"`
	Description string        `json:"description"`
	Images      []MomentImage `json:"images"`
	Tags        []string      `json:"tags"`
	Mentions    []string      `json:"mentions"`
	CreatedAt   string        `json:"created_at"`
	// LikesCount 兼容保留（reactions 聚合之前的历史计数，契约已删 likes_count），
	// 仅供内部/测试使用，不对外序列化。
	LikesCount    int      `json:"-"`
	CommentsCount int      `json:"comments_count"`
	SharesCount   int      `json:"shares_count"`
	User          UserInfo `json:"user"`
	IsDeleted     bool     `json:"is_deleted"`
	// location 恒定 nullable（spec MomentItem）：nil = 隐私隐藏/未设置，恒输出 null。
	Location *Location `json:"location"`
	// is_favorited 恒定布尔（spec MomentItem）：匿名/未收藏为 false，恒输出。
	IsFavorited bool `json:"is_favorited"`
	// Reactions 表情反应聚合（spec MomentItem reactions: 空数组 = 无反应）。
	Reactions []Reaction `json:"reactions"`
	// recommendation 恒定存在（spec api_spec_types.md MomentItem：Feed 场景返回推荐
	// 理由，非 Feed 场景为 null）。Feed 域已废除（能力收口至 moments/search 热门模式
	// 与 fetch_nearby/fetch_following，spec api_spec_moment.md），当前所有端点均为
	// 非 Feed 场景，恒序列化 null（指针留空）。
	Recommendation *RecommendationType `json:"recommendation"`
	// PrivacyLevel 仅 repo 内部使用（过滤/收藏/拥有者端点），不对外序列化。
	PrivacyLevel string `json:"-"`
	// FavoritedAt 仅 list_favorites 返回（spec moment.md → FavoritedMomentItem）。
	FavoritedAt string `json:"favorited_at,omitempty"`
}

// OwnerMomentItem 拥有者视角扩展（spec moment.md → OwnerMomentItem）：
// = MomentItem + privacy_level 必填。仅拥有者端点返回
// （moments/fetch_created、moments/create、moments/update）。
type OwnerMomentItem struct {
	MomentItem
	PrivacyLevel string `json:"privacy_level"`
}

type CreateMomentRequest struct {
	Description  string    `json:"description"`
	Location     *Location `json:"location"`
	PrivacyLevel string    `json:"privacy_level,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	ImageFileIDs []string  `json:"image_file_ids,omitempty"`
	Mentions     []string  `json:"mentions,omitempty"`
}

type MomentOpenTheDoorRequest struct {
	MomentID string `json:"moment_id"`
}

// PaginatedMoments 公开瞬间分页容器（spec moment.md → moments/search）。
type PaginatedMoments struct {
	Items      []MomentItem `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}

// PaginatedOwnerMoments 拥有者瞬间分页容器（spec moment.md → moments/fetch_created）。
type PaginatedOwnerMoments struct {
	Items      []OwnerMomentItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

// SearchMomentsRequest：全球 keyword/tag 搜索（spec api_spec_moment.md moments/search——
// 已删除 radius/sort_by='distance'/reference_point；搜索为全局匹配，附近发现唯一
// 入口为 moments/fetch_nearby）
type SearchMomentsRequest struct {
	Keyword    *string           `json:"keyword,omitempty"`
	Tag        *string           `json:"tag,omitempty"`
	SortBy     string            `json:"sort_by,omitempty"` // relevance | newest | popular
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type GeoPoint = types.GeoPoint

type PaginationParams = types.PaginationParams

type UpdateMomentRequest struct {
	MomentID     string    `json:"moment_id"`
	Description  *string   `json:"description,omitempty"`
	Location     *Location `json:"location,omitempty"`
	PrivacyLevel *string   `json:"privacy_level,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	ImageFileIDs []string  `json:"image_file_ids,omitempty"`
	Mentions     []string  `json:"mentions,omitempty"`
}

type DeleteMomentRequest struct {
	MomentID string `json:"moment_id"`
}

type FavoriteMomentRequest struct {
	MomentID string `json:"moment_id"`
	Active   bool   `json:"active"`
}

type UnfavoriteMomentRequest struct {
	MomentID string `json:"moment_id"`
}

// ListCreatedMomentsRequest 我创建的瞬间列表（spec moment.md。
// 原 list_my 已废除，统一为 list_created）。返回拥有者视角 OwnerMomentItem。
type ListCreatedMomentsRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type MyFavoritesRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// FetchFollowingMomentsRequest 关注动态请求（spec moment.md → moments/fetch_following，
// 原 feed/following 迁移而来）。无匿名版（依赖查看者关注关系）。
type FetchFollowingMomentsRequest struct {
	ReferencePoint *GeoPoint         `json:"reference_point,omitempty"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

// FetchNearbyMomentsRequest 附近瞬间请求（spec moment.md → moments/fetch_nearby，
// 原 feed/nearby_moments 迁移而来；双版本端点，匿名版见 api_spec_public.md）。
type FetchNearbyMomentsRequest struct {
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}
