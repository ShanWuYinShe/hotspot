package search

import "hotspot/backend-go/internal/platform/types"

type GeoPoint = types.GeoPoint

type RelativeLocation = types.RelativeLocation

type SearchSuggestRequest struct {
	Prefix string `json:"prefix"`
	// ReferencePoint 指针语义表达「是否传入」：nil=未传（service 走回退链）；
	// 显式传入 (0,0) 等合法坐标不得被零值哨兵误判为未传（对齐 moments 的
	// 修法，spec api_spec_types.md「reference_point 解析规则」）。
	ReferencePoint *GeoPoint `json:"reference_point"`
	Types          []string  `json:"types,omitempty"`
	Radius         int       `json:"radius,omitempty"`
}

type UserSuggestItem struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Bio       string `json:"bio,omitempty"`
}

type GroupSuggestItem struct {
	GroupID      string `json:"group_id"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	Description  string `json:"description,omitempty"`
	MembersCount int    `json:"members_count"`
}

type MomentSuggestItem struct {
	MomentID           string `json:"moment_id"`
	DescriptionPreview string `json:"description_preview"`
	Author             struct {
		UserID   string `json:"user_id"`
		Nickname string `json:"nickname"`
	} `json:"author"`
}

// 建议项 payload 判别联合：Type 决定 Payload 的具体类型。
// 登录版使用 *Payload（含 relation/social_context/viewer_state 等查看者字段）；
// /public/v1 使用 Public*Payload（spec api_spec_public.md 类型体系）。
type SearchSuggestItem struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type SearchSuggestUserRelation struct {
	IsFollowing  bool `json:"is_following"`
	IsFollowedBy bool `json:"is_followed_by"`
	IsBlocked    bool `json:"is_blocked"`
}

type SearchSuggestUserStats struct {
	MomentsCount   int `json:"moments_count"`
	FriendsCount   int `json:"friends_count"`
	FollowersCount int `json:"followers_count"`
	FollowingCount int `json:"following_count"`
	GroupsCount    int `json:"groups_count"`
}

type SearchSuggestUser struct {
	UserID    string  `json:"user_id"`
	Nickname  string  `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
}

// SearchSuggestUserPayload 是登录视角的用户建议 payload（spec UserItem 形状）。
type SearchSuggestUserPayload struct {
	User           SearchSuggestUser          `json:"user"`
	Bio            *string                    `json:"bio"`
	Tags           []string                   `json:"tags"`
	PresenceStatus *string                    `json:"presence_status"`
	LastOnlineAt   *string                    `json:"last_online_at"`
	Location       *RelativeLocation          `json:"location"`
	Stats          *SearchSuggestUserStats    `json:"stats"`
	Relation       *SearchSuggestUserRelation `json:"relation"`
	SocialContext  any                        `json:"social_context"`
}

type SearchSuggestGroup struct {
	GroupID      string  `json:"group_id"`
	Name         string  `json:"name"`
	MembersCount int     `json:"members_count"`
	AvatarURL    *string `json:"avatar_url"`
}

// SearchSuggestGroupPayload 是登录视角的群组建议 payload（spec GroupItem 形状）。
type SearchSuggestGroupPayload struct {
	Group              SearchSuggestGroup `json:"group"`
	Description        *string            `json:"description"`
	Tags               []string           `json:"tags"`
	JoinMethod         string             `json:"join_method"`
	RequiresApproval   bool               `json:"requires_approval"`
	Location           *RelativeLocation  `json:"location"`
	LastActivityAt     *string            `json:"last_activity_at"`
	OnlineMembersCount int                `json:"online_members_count"`
	NextActivityAt     *string            `json:"next_activity_at"`
	CreatedAt          *string            `json:"created_at"`
	OwnerID            *string            `json:"owner_id"`
	ViewerState        any                `json:"viewer_state"`
	FriendsInGroup     *int               `json:"friends_in_group"`
}

// SearchSuggestMomentPayload 是瞬间建议 payload（MomentSummary + author + location；
// 无查看者字段，登录/匿名共用）。
type SearchSuggestMomentPayload struct {
	MomentID           string            `json:"moment_id"`
	DescriptionPreview string            `json:"description_preview"`
	CoverURL           *string           `json:"cover_url"`
	Author             SearchSuggestUser `json:"author"`
	Location           *RelativeLocation `json:"location"`
}

// TagItem 是 tag 建议 payload（spec api_spec_types.md，无查看者字段）。
type TagItem struct {
	Name       string `json:"name"`
	UsageCount int    `json:"usage_count"`
}

// ── /public/v1 匿名 payload（无查看者字段）────────────────────────────

type PublicSearchSuggestUserPayload struct {
	User           SearchSuggestUser       `json:"user"`
	Bio            *string                 `json:"bio"`
	Tags           []string                `json:"tags"`
	PresenceStatus *string                 `json:"presence_status"`
	LastOnlineAt   *string                 `json:"last_online_at"`
	Location       *RelativeLocation       `json:"location"`
	Stats          *SearchSuggestUserStats `json:"stats"`
}

type PublicSearchSuggestGroupPayload struct {
	Group              SearchSuggestGroup `json:"group"`
	Description        *string            `json:"description"`
	Tags               []string           `json:"tags"`
	JoinMethod         string             `json:"join_method"`
	RequiresApproval   bool               `json:"requires_approval"`
	Location           *RelativeLocation  `json:"location"`
	LastActivityAt     *string            `json:"last_activity_at"`
	OnlineMembersCount int                `json:"online_members_count"`
	NextActivityAt     *string            `json:"next_activity_at"`
	CreatedAt          *string            `json:"created_at"`
	OwnerID            *string            `json:"owner_id"`
}

type PublicSearchSuggestItem struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type PublicSearchSuggestResponse struct {
	Items []PublicSearchSuggestItem `json:"items"`
}

type SearchSuggestResponse struct {
	Items []SearchSuggestItem `json:"items"`
}
