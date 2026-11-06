package groups

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

// ── 匿名节点（/public/v1）专属类型 ──────────────────────────────────────
// spec api_spec_public.md「类型体系」：匿名版响应不含查看者状态字段。
// PublicGroupItem = GroupItem 去 viewer_state / friends_in_group / last_message；
// 成员项独立 Public 类型（字段与管理视角无关，但类型隔离禁止共用）。

type PublicGroupItem struct {
	Group              GroupInfo         `json:"group"`
	Description        *string           `json:"description"`
	Tags               []string          `json:"tags"`
	JoinMethod         GroupJoinMethod   `json:"join_method"`
	RequiresApproval   bool              `json:"requires_approval"`
	Location           *RelativeLocation `json:"location"`
	LastActivityAt     *string           `json:"last_activity_at"`
	OnlineMembersCount *int              `json:"online_members_count"`
	NextActivityAt     *string           `json:"next_activity_at"`
	CreatedAt          *string           `json:"created_at"`
	OwnerID            *string           `json:"owner_id"`
}

type PublicPaginatedGroups struct {
	Items      []PublicGroupItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type PublicGroupMemberItem struct {
	User           UserInfo            `json:"user"`
	Role           GroupRole           `json:"role"`
	JoinedAt       string              `json:"joined_at"`
	PresenceStatus *UserPresenceStatus `json:"presence_status"`
	LastOnline     *string             `json:"last_online_at"`
}

type PublicPaginatedGroupMembers struct {
	Items      []PublicGroupMemberItem `json:"items"`
	NextCursor *string                 `json:"next_cursor"`
}

type PublicSearchGroupsRequest struct {
	Keyword        string            `json:"keyword,omitempty"`
	Tag            string            `json:"tag,omitempty"`
	Radius         *int              `json:"radius,omitempty"`
	SortBy         string            `json:"sort_by,omitempty"`
	ReferencePoint GeoPoint          `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type PublicOpenGroupDoorRequest struct {
	GroupID string `json:"group_id"`
}

type PublicFetchGroupMembersRequest struct {
	GroupID    string            `json:"group_id"`
	Role       *GroupRole        `json:"role,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PublicSearchGroupMembersRequest struct {
	GroupID    string            `json:"group_id"`
	Keyword    string            `json:"keyword"`
	Role       *GroupRole        `json:"role,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

func (r PublicSearchGroupsRequest) toInternal() SearchGroupsRequest {
	return SearchGroupsRequest{Keyword: r.Keyword, Tag: r.Tag, Radius: r.Radius, SortBy: r.SortBy, ReferencePoint: r.ReferencePoint, Pagination: r.Pagination}
}

func (r PublicFetchGroupMembersRequest) toInternal() GroupMembersRequest {
	return GroupMembersRequest{GroupID: r.GroupID, Role: r.Role, Pagination: r.Pagination}
}

func (r PublicSearchGroupMembersRequest) toInternal() SearchGroupMembersRequest {
	return SearchGroupMembersRequest{GroupID: r.GroupID, Keyword: r.Keyword, Role: r.Role, Pagination: r.Pagination}
}

func ToPublicGroupItem(item GroupItem) PublicGroupItem {
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	return PublicGroupItem{
		Group:              item.Group,
		Description:        item.Description,
		Tags:               tags,
		JoinMethod:         item.JoinMethod,
		RequiresApproval:   item.RequiresApproval,
		Location:           item.Location,
		LastActivityAt:     item.LastActivityAt,
		OnlineMembersCount: item.OnlineMembersCount,
		NextActivityAt:     item.NextActivityAt,
		CreatedAt:          item.CreatedAt,
		OwnerID:            item.OwnerID,
	}
}

func ToPublicPaginatedGroups(page PaginatedGroups) PublicPaginatedGroups {
	items := make([]PublicGroupItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, ToPublicGroupItem(item))
	}
	return PublicPaginatedGroups{Items: items, NextCursor: page.NextCursor}
}

func toPublicGroupMembers(items []GroupMemberItem, nextCursor string) PublicPaginatedGroupMembers {
	members := make([]PublicGroupMemberItem, 0, len(items))
	for _, item := range items {
		members = append(members, PublicGroupMemberItem{User: item.User, Role: item.Role, JoinedAt: item.JoinedAt, PresenceStatus: item.PresenceStatus, LastOnline: item.LastOnline})
	}
	return PublicPaginatedGroupMembers{Items: members, NextCursor: types.StrPtr(nextCursor)}
}

// ── 匿名端点 handler（仅挂在 /public/v1/groups/*）──────────────────────

func (h *Handler) PublicSearch(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicSearchGroupsRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	req := pubReq.toInternal()
	if !hasSearchCriteria(req) {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "至少需要提供 keyword、tag、radius 之一")
		return
	}
	if req.Radius != nil && (*req.Radius < minSearchRadius || *req.Radius > maxSearchRadius) {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "radius 超出范围")
		return
	}
	if utf8.RuneCountInString(req.Keyword) > 100 || utf8.RuneCountInString(req.Tag) > 30 {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	if sortBy := normalizedSearchSort(req.SortBy); sortBy == "" && strings.TrimSpace(req.SortBy) != "" {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "sort_by 不支持")
		return
	}
	req = normalizedSearchRequest(req)
	list, err := h.service.Search(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteAPIError(w, r, "PublicSearch", httpx.Forbidden(httpx.ReasonPrivacySetting, "权限不足"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "PublicSearch", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(ToPublicPaginatedGroups(list)))
}

func (h *Handler) PublicOpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req PublicOpenGroupDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.GroupID)
	if err != nil {
		if _, isAPI := err.(*httpx.APIError); isAPI {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		httpx.WriteAPIError(w, r, "PublicOpenTheDoor", err)
		return
	}
	if !ok {
		httpx.WriteCodeError(w, r, httpx.CodeNotFound, "群组不存在")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(ToPublicGroupItem(item)))
}

func (h *Handler) PublicListMembers(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicFetchGroupMembersRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil || pubReq.GroupID == "" {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	items, nextCursor, err := h.service.ListMembers(r.Context(), pubReq.toInternal())
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteCodeError(w, r, httpx.CodeNotFound, "群组不存在")
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
			return
		}
		httpx.WriteServerError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicGroupMembers(items, nextCursor)))
}

func (h *Handler) PublicSearchMembers(w http.ResponseWriter, r *http.Request) {
	var pubReq PublicSearchGroupMembersRequest
	if err := httpx.DecodeStrictJSON(r, &pubReq); err != nil || pubReq.GroupID == "" || pubReq.Keyword == "" {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	if utf8.RuneCountInString(pubReq.Keyword) > 100 {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return
	}
	items, nextCursor, err := h.service.SearchMembers(r.Context(), pubReq.toInternal())
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteCodeError(w, r, httpx.CodeNotFound, "群组不存在")
			return
		}
		if errors.Is(err, ErrNotMember) {
			httpx.WriteAPIError(w, r, "PublicSearchMembers", httpx.Forbidden(httpx.ReasonNotGroupMember, "不具备群组成员搜索资格"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
			return
		}
		httpx.WriteServerError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicGroupMembers(items, nextCursor)))
}
