package users

import (
	"net/http"

	grouptypes "hotspot/backend-go/internal/modules/groups"
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

// ── 关系列表 / 用户群组 / 用户瞬间（匿名版）──────────────────────────

// PublicRelatedUserItem：RelatedUserItem 去 remark（查看者字段）；互关由
// followed_back_at 表达（仅 fetch_friends 返回真值）。
type PublicRelatedUserItem struct {
	User           PublicUserItem `json:"user"`
	FollowedAt     string         `json:"followed_at"`
	FollowedBackAt *string        `json:"followed_back_at"`
}

type PublicPaginatedRelatedUsers struct {
	Items      []PublicRelatedUserItem `json:"items"`
	NextCursor *string                 `json:"next_cursor"`
}

type PublicFetchFriendsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicFetchFollowingRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicFetchFollowersRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicFetchUserGroupsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicFetchUserMomentsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicFetchUserReactedMomentsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

func toPublicRelatedUserItem(item RelatedUserItem) PublicRelatedUserItem {
	return PublicRelatedUserItem{
		User:           toPublicUserItem(item.User),
		FollowedAt:     item.FollowedAt,
		FollowedBackAt: item.FollowedBackAt,
	}
}

func toPublicPaginatedRelatedUsers(items []RelatedUserItem, nextCursor *string) PublicPaginatedRelatedUsers {
	converted := make([]PublicRelatedUserItem, 0, len(items))
	for _, item := range items {
		converted = append(converted, toPublicRelatedUserItem(item))
	}
	return PublicPaginatedRelatedUsers{Items: converted, NextCursor: nextCursor}
}

func (h *Handler) PublicListFriends(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchFriendsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	result, err := h.service.ListFriends(r.Context(), ListFriendsRequest{UserID: pubReq.UserID, Pagination: pubReq.Pagination})
	if err != nil {
		h.writeServiceError(w, r, "PublicListFriends", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicPaginatedRelatedUsers(result.Items, result.NextCursor)))
}

func (h *Handler) PublicListFollowing(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchFollowingRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	result, err := h.service.ListFollowing(r.Context(), ListFollowingRequest{UserID: pubReq.UserID, Pagination: pubReq.Pagination})
	if err != nil {
		h.writeServiceError(w, r, "PublicListFollowing", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicPaginatedRelatedUsers(result.Items, result.NextCursor)))
}

func (h *Handler) PublicListFollowers(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchFollowersRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	result, err := h.service.ListFollowers(r.Context(), ListFollowersRequest{UserID: pubReq.UserID, Pagination: pubReq.Pagination})
	if err != nil {
		h.writeServiceError(w, r, "PublicListFollowers", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicPaginatedRelatedUsers(result.Items, result.NextCursor)))
}

func (h *Handler) PublicListGroups(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchUserGroupsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	result, err := h.service.ListGroups(r.Context(), UserGroupsRequest{UserID: pubReq.UserID, Pagination: pubReq.Pagination})
	if err != nil {
		h.writeServiceError(w, r, "PublicListGroups", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(grouptypes.ToPublicPaginatedGroups(result)))
}

func (h *Handler) PublicListMoments(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchUserMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	result, err := h.service.ListMoments(r.Context(), UserMomentsRequest{UserID: pubReq.UserID, Pagination: pubReq.Pagination})
	if err != nil {
		h.writeServiceError(w, r, "PublicListMoments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(momenttypes.ToPublicPaginatedMoments(result)))
}

func (h *Handler) PublicListReactedMoments(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchUserReactedMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	result, err := h.service.ListReactedMoments(r.Context(), UserReactedMomentsRequest{UserID: pubReq.UserID, Pagination: pubReq.Pagination})
	if err != nil {
		h.writeServiceError(w, r, "PublicListReactedMoments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(momenttypes.ToPublicPaginatedMoments(result)))
}
