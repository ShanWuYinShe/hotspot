package groups

import (
	"errors"
	"net/http"
	"strings"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Name == "" || req.Location.LocationName == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/create", func() httpx.ApiResponse[any] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			if apiErr, ok := err.(*httpx.APIError); ok {
				return httpx.ApiResponse[any]{Code: apiErr.Code, Message: apiErr.Message, Data: &apiErr.Data}
			}
			observability.LogServerError(r, "Create", err)
			return httpx.ApiResponse[any]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		var data any = item
		return httpx.ApiResponse[any]{Code: httpx.CodeOK, Data: &data}
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req GroupOpenTheDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.GroupID)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		httpx.WriteAPIError(w, r, "OpenTheDoor", err)
		return
	}
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchGroupsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if !hasSearchCriteria(req) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "至少需要提供 keyword、tag、radius 之一"))
		return
	}
	if req.Radius != nil && (*req.Radius < minSearchRadius || *req.Radius > maxSearchRadius) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "radius 超出范围"))
		return
	}
	if sortBy := normalizedSearchSort(req.SortBy); sortBy == "" && strings.TrimSpace(req.SortBy) != "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "sort_by 不支持"))
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
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		httpx.WriteAPIError(w, r, "Search", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if req.ClearPassword != nil && *req.ClearPassword && req.Password != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "密码参数冲突"))
		return
	}
	item, err := h.service.Update(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	// 内容类更新返回更新后完整对象（spec api_spec_base.md「设计规范」）
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	var req DismissGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.Dismiss(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrInvalidReason) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	var req GroupMembersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListMembers(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) SearchMembers(w http.ResponseWriter, r *http.Request) {
	var req SearchGroupMembersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.Keyword == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.SearchMembers(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrNotMember) {
			// spec api_spec_group.md members/search：非成员且 members_searchable=false 时
			// 返回 10200 + not_group_member（原 10400 已迁移，见 handler_test.go 说明）
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "不具备群组成员搜索资格"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	var req MyGroupsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyGroups(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	var req EnterGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 自然去重（spec api_spec_base.md）：approval 路径命中已有 pending 申请返回已有
	// application_id（不重复创建/通知）；direct/invite 路径成员关系唯一。无需 Idempotency-Key。
	result, err := h.service.JoinGroup(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrInvalidPassword) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "密码错误"))
			return
		}
		if errors.Is(err, ErrAlreadyApplied) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "已申请过"))
			return
		}
		if errors.Is(err, ErrGroupFull) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "群组已满"))
			return
		}
		if errors.Is(err, ErrBusinessConflict) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "该群组仅限邀请加入，请通过邀请加入"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateGroupAnnouncementRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.Title == "" || req.Content == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/announcements/create", func() httpx.ApiResponse[any] {
		item, err := h.service.CreateAnnouncement(r.Context(), req)
		if err != nil {
			if apiErr, ok := err.(*httpx.APIError); ok {
				return httpx.ApiResponse[any]{Code: apiErr.Code, Message: apiErr.Message, Data: &apiErr.Data}
			}
			if errors.Is(err, ErrGroupNotFound) {
				return httpx.ApiResponse[any]{Code: httpx.CodeNotFound, Message: "群组不存在"}
			}
			if errors.Is(err, ErrAccessRestricted) {
				data := any(httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonNotGroupAdmin})
				return httpx.ApiResponse[any]{Code: httpx.CodeForbidden, Message: "权限不足", Data: &data}
			}
			return httpx.ApiResponse[any]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		var data any = item
		return httpx.ApiResponse[any]{Code: httpx.CodeOK, Data: &data}
	})
}

func (h *Handler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	var req GroupAnnouncementsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	items, err := h.service.ListAnnouncements(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	resp := map[string]any{"items": items}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupAnnouncementRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.AnnouncementID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	item, err := h.service.UpdateAnnouncement(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		if errors.Is(err, ErrAnnouncementNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "公告不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	// 内容类更新返回更新后完整对象（spec api_spec_base.md「设计规范」）
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req DeleteGroupAnnouncementRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.AnnouncementID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.DeleteAnnouncement(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupPreferencesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.UpdatePreferences(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrNotMember) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "不是群成员"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	var req LeaveGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.LeaveGroup(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrNotMember) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "不是群成员"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "群主需先转让群主身份"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListMyApplications(w http.ResponseWriter, r *http.Request) {
	var req MyApplicationsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyApplications(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) CancelApplication(w http.ResponseWriter, r *http.Request) {
	var req CancelApplicationRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.ApplicationID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.CancelApplication(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrApplicationNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "申请不存在"))
			return
		}
		if errors.Is(err, ErrInvalidReason) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}
