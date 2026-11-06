package users

import (
	"net/http"

	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/httpx"
)

// ── 匿名节点（/public/v1）专属类型 ──────────────────────────────────────
// spec api_spec_public.md「类型体系」：匿名版响应使用 Public* 类型，去掉
// relation / social_context 等查看者字段；batch_fetch 按 spec 复用共享
// UsersBatchResult（本就不含查看者字段）。

// PublicUserItem：UserItem 去 relation / social_context。
type PublicUserItem struct {
	User           UserInfo            `json:"user"`
	Bio            *string             `json:"bio"`
	Tags           []string            `json:"tags"`
	PresenceStatus *UserPresenceStatus `json:"presence_status"`
	LastOnline     *string             `json:"last_online_at"`
	Location       *RelativeLocation   `json:"location"`
	Stats          *UserStats          `json:"stats"`
}

type PublicPaginatedUsers struct {
	Items      []PublicUserItem `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

type PublicSearchUsersRequest struct {
	Keyword        string            `json:"keyword,omitempty"`
	Tag            string            `json:"tag,omitempty"`
	Radius         *int              `json:"radius,omitempty"`
	SortBy         string            `json:"sort_by,omitempty"`
	ReferencePoint GeoPoint          `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type PublicOpenUserDoorRequest struct {
	UserID               string `json:"user_id"`
	IncludeRecentMoments bool   `json:"include_recent_moments,omitempty"`
}

type PublicBatchFetchUsersRequest struct {
	UserIDs []string `json:"user_ids"`
}

// PublicUserProfileResult：open_the_door 公开详情（recent_moments 用 moments
// 模块导出的 PublicMomentItem，保证与 /public/v1/moments/* 同一公开形状）。
type PublicUserProfileResult struct {
	User          PublicUserItem                      `json:"user"`
	RecentMoments *momenttypes.PublicPaginatedMoments `json:"recent_moments"`
}

func (r PublicSearchUsersRequest) toInternal() SearchUsersRequest {
	return SearchUsersRequest{Keyword: r.Keyword, Tag: r.Tag, Radius: r.Radius, SortBy: r.SortBy, ReferencePoint: r.ReferencePoint, Pagination: r.Pagination}
}

func toPublicUserItem(item UserItem) PublicUserItem {
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	return PublicUserItem{
		User:           item.User,
		Bio:            item.Bio,
		Tags:           tags,
		PresenceStatus: item.PresenceStatus,
		LastOnline:     item.LastOnline,
		Location:       item.Location,
		Stats:          item.Stats,
	}
}

func toPublicPaginatedUsers(page PaginatedUsers) PublicPaginatedUsers {
	items := make([]PublicUserItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toPublicUserItem(item))
	}
	return PublicPaginatedUsers{Items: items, NextCursor: page.NextCursor}
}

func toPublicUserProfileResult(bundle UserProfileBundle) PublicUserProfileResult {
	result := PublicUserProfileResult{User: toPublicUserItem(bundle.User)}
	if bundle.RecentMoments != nil {
		page := momenttypes.ToPublicPaginatedMoments(*bundle.RecentMoments)
		result.RecentMoments = &page
	}
	return result
}

// ── 匿名端点 handler（仅挂在 /public/v1/users/*）──────────────────────

func (h *Handler) PublicSearch(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicSearchUsersRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	req := pubReq.toInternal()
	if msg := validateSearchUsersRequest(req); msg != "" {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, msg)
		return
	}
	list, err := h.service.Search(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "PublicSearch", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicPaginatedUsers(list)))
}

func (h *Handler) PublicOpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicOpenUserDoorRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil || pubReq.UserID == "" {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	bundle, ok, err := h.service.GetProfileBundle(r.Context(), pubReq.UserID, pubReq.IncludeRecentMoments)
	if err != nil {
		h.writeServiceError(w, r, "PublicOpenTheDoor", err)
		return
	}
	if !ok {
		httpx.WriteCodeError(w, r, httpx.CodeNotFound, "用户不存在")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicUserProfileResult(bundle)))
}

func (h *Handler) PublicBatchGet(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicBatchFetchUsersRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil || len(pubReq.UserIDs) == 0 {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	if len(pubReq.UserIDs) > 100 {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	users, err := h.service.BatchGet(r.Context(), pubReq.UserIDs)
	if err != nil {
		h.writeServiceError(w, r, "PublicBatchGet", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(UsersBatchResult{Users: users}))
}
