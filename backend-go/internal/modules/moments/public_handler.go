package moments

import (
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

// ── 匿名节点（/public/v1）专属类型 ──────────────────────────────────────
// spec api_spec_public.md「类型体系」：匿名版 Request/Response 使用 Public* 类型，
// 与登录版禁止共用；响应不含查看者状态字段（is_favorited / author_relation /
// recommendation 等）。此处只做类型隔离与序列化映射，服务返回的 MomentItem
// 仍携带内部字段，但 Public 结构体根本不存在这些字段 → 不会外泄。

// PublicReaction：Reaction 去 reacted_by_me。
type PublicReaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

// PublicMomentItem：MomentItem 去 is_favorited / author_relation / recommendation。
type PublicMomentItem struct {
	MomentID      string           `json:"moment_id"`
	Description   string           `json:"description"`
	Images        []MomentImage    `json:"images"`
	Tags          []string         `json:"tags"`
	Mentions      []string         `json:"mentions"`
	CreatedAt     string           `json:"created_at"`
	CommentsCount int              `json:"comments_count"`
	SharesCount   int              `json:"shares_count"`
	User          UserInfo         `json:"user"`
	IsDeleted     bool             `json:"is_deleted"`
	Location      *Location        `json:"location"`
	Reactions     []PublicReaction `json:"reactions"`
}

type PublicPaginatedMoments struct {
	Items      []PublicMomentItem `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}

type PublicSearchMomentsRequest struct {
	Keyword    *string           `json:"keyword,omitempty"`
	Tag        *string           `json:"tag,omitempty"`
	SortBy     string            `json:"sort_by,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicFetchNearbyMomentsRequest struct {
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type PublicOpenMomentDoorRequest struct {
	MomentID string `json:"moment_id"`
}

func (r PublicSearchMomentsRequest) toInternal() SearchMomentsRequest {
	return SearchMomentsRequest{Keyword: r.Keyword, Tag: r.Tag, SortBy: r.SortBy, Pagination: r.Pagination}
}

func (r PublicFetchNearbyMomentsRequest) toInternal() FetchNearbyMomentsRequest {
	return FetchNearbyMomentsRequest{Radius: r.Radius, ReferencePoint: r.ReferencePoint, Pagination: r.Pagination}
}

func ToPublicMomentItem(item MomentItem) PublicMomentItem {
	images := item.Images
	if images == nil {
		images = []MomentImage{}
	}
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	mentions := item.Mentions
	if mentions == nil {
		mentions = []string{}
	}
	reactions := make([]PublicReaction, 0, len(item.Reactions))
	for _, r := range item.Reactions {
		reactions = append(reactions, PublicReaction{Emoji: r.Emoji, Count: r.Count})
	}
	return PublicMomentItem{
		MomentID:      item.MomentID,
		Description:   item.Description,
		Images:        images,
		Tags:          tags,
		Mentions:      mentions,
		CreatedAt:     item.CreatedAt,
		CommentsCount: item.CommentsCount,
		SharesCount:   item.SharesCount,
		User:          item.User,
		IsDeleted:     item.IsDeleted,
		Location:      item.Location,
		Reactions:     reactions,
	}
}

func ToPublicPaginatedMoments(page PaginatedMoments) PublicPaginatedMoments {
	items := make([]PublicMomentItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, ToPublicMomentItem(item))
	}
	return PublicPaginatedMoments{Items: items, NextCursor: page.NextCursor}
}

// ── 匿名端点 handler（仅挂在 /public/v1/moments/*）──────────────────────

func (h *Handler) PublicOpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req PublicOpenMomentDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		httpx.WriteInvalidRequest(w, r, "PublicOpenTheDoor", "参数错误")
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.MomentID)
	if err != nil {
		httpx.WriteAPIError(w, r, "PublicOpenTheDoor", err)
		return
	}
	if !ok {
		httpx.WriteBusinessError(w, r, "PublicOpenTheDoor", httpx.CodeNotFound, "瞬间不存在")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(ToPublicMomentItem(item)))
}

func (h *Handler) PublicSearch(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicSearchMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteInvalidRequest(w, r, "PublicSearch", "参数错误")
		return
	}
	req := pubReq.toInternal()
	if apiErr := validateSearchMomentsRequest(req); apiErr != nil {
		httpx.WriteAPIError(w, r, "PublicSearch", apiErr)
		return
	}
	result, err := h.service.Search(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "PublicSearch", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(ToPublicPaginatedMoments(result)))
}

func (h *Handler) PublicFetchNearby(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchNearbyMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteInvalidRequest(w, r, "PublicFetchNearby", "参数错误")
		return
	}
	result, err := h.service.FetchNearby(r.Context(), pubReq.toInternal())
	if err != nil {
		h.writeFetchError(w, r, "PublicFetchNearby", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(ToPublicPaginatedMoments(result)))
}
