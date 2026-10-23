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

type SearchSuggestItem struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type SearchSuggestResponse struct {
	Items []SearchSuggestItem `json:"items"`
}
