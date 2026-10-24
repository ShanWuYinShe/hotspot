package groups

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/types"
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
	// Reject out-of-range coordinates before they corrupt the S2 index: an
	// invalid point yields s2_cell_id=0, which falls outside every BETWEEN
	// range and silently drops the group from map/nearby/search（对齐 moments
	// Create/Update 的同款校验）。
	if !geo.IsValidPoint(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}) {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误: 经纬度超出有效范围")
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/create", func() httpx.ApiResponse[any] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			if apiErr, ok := err.(*httpx.APIError); ok {
				return httpx.ApiResponse[any]{Code: apiErr.Code, Message: apiErr.Message, Data: &apiErr.Data}
			}
			// tags 含被屏蔽标签：10400（spec api_spec_group.md → groups/create
			// 错误码「10400（tags 含被屏蔽标签）」）。此前未映射落入 10600。
			if errors.Is(err, ErrBusinessConflict) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeConflict), "标签包含被屏蔽内容")
				return httpx.ApiResponse[any]{Code: httpx.CodeConflict, Message: "标签包含被屏蔽内容"}
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
	// 契约（api_spec_base.md「搜索与地理约束」）：keyword 1-100 字符、tag 1-30 字符。
	if utf8.RuneCountInString(req.Keyword) > 100 || utf8.RuneCountInString(req.Tag) > 30 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
			httpx.WriteAPIError(w, r, "Search", httpx.Forbidden(httpx.ReasonPrivacySetting, "权限不足"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
	// 同 Create：更新定位时也必须在入库前拒绝非法坐标。
	if req.Location != nil && !geo.IsValidPoint(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}) {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误: 经纬度超出有效范围")
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
			httpx.WriteAPIError(w, r, "Update", httpx.Forbidden(httpx.ReasonNotGroupAdmin, "权限不足"))
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
			httpx.WriteAPIError(w, r, "Dismiss", httpx.Forbidden(httpx.ReasonNotGroupOwner, "权限不足"))
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
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PaginatedGroupMembers{Items: items, NextCursor: types.StrPtr(nextCursor)}))
}

func (h *Handler) SearchMembers(w http.ResponseWriter, r *http.Request) {
	var req SearchGroupMembersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.Keyword == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if utf8.RuneCountInString(req.Keyword) > 100 {
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
			httpx.WriteAPIError(w, r, "SearchMembers", httpx.Forbidden(httpx.ReasonNotGroupMember, "不具备群组成员搜索资格"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PaginatedGroupMembers{Items: items, NextCursor: types.StrPtr(nextCursor)}))
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
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PaginatedGroups{Items: items, NextCursor: types.StrPtr(nextCursor)}))
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
			httpx.WriteAPIError(w, r, "JoinGroup", httpx.Forbidden(httpx.ReasonPrivacySetting, "权限不足"))
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
			httpx.WriteAPIError(w, r, "UpdateAnnouncement", httpx.Forbidden(httpx.ReasonNotGroupAdmin, "权限不足"))
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
		if errors.Is(err, ErrAnnouncementNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "公告不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteAPIError(w, r, "DeleteAnnouncement", httpx.Forbidden(httpx.ReasonNotGroupAdmin, "权限不足"))
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
			httpx.WriteAPIError(w, r, "UpdatePreferences", httpx.Forbidden(httpx.ReasonNotGroupMember, "不是群成员"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteAPIError(w, r, "UpdatePreferences", httpx.Forbidden(httpx.ReasonNotGroupMember, "权限不足"))
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
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PaginatedApplications{Items: items, NextCursor: types.StrPtr(nextCursor)}))
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
		if errors.Is(err, ErrAlreadyProcessed) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "申请已处理"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}
