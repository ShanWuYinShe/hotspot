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

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code int, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Name == "" || req.Location.LocationName == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/create", func() httpx.ApiResponse[GroupItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			logServerError(r, "Create", err)
			return httpx.ApiResponse[GroupItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req GroupOpenTheDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.GroupID)
	if err != nil {
		logServerError(r, "OpenTheDoor", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchGroupsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if !hasSearchCriteria(req) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "至少需要提供 keyword、tag、radius 之一"))
		return
	}
	if req.Radius != nil && (*req.Radius < minSearchRadius || *req.Radius > maxSearchRadius) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "radius 超出范围"))
		return
	}
	if sortBy := normalizedSearchSort(req.SortBy); sortBy == "" && strings.TrimSpace(req.SortBy) != "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "sort_by 不支持"))
		return
	}
	req = normalizedSearchRequest(req)
	list, err := h.service.Search(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		logServerError(r, "Search", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.ClearPassword != nil && *req.ClearPassword && req.Password != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "密码参数冲突"))
		return
	}
	item, err := h.service.Update(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	// 内容类更新返回更新后完整对象（spec api_spec_base.md「设计规范」）
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	var req DismissGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Dismiss(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	var req GroupMembersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListMembers(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.SearchMembers(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyGroups(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	var req JoinGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	// 自然去重（spec api_spec_base.md）：approval 路径命中已有 pending 申请返回已有
	// application_id（不重复创建/通知）；direct/invite 路径成员关系唯一。无需 Idempotency-Key。
	result, err := h.service.JoinGroup(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrInvalidPassword) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "密码错误"))
			return
		}
		if errors.Is(err, ErrAlreadyApplied) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "已申请过"))
			return
		}
		if errors.Is(err, ErrGroupFull) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "群组已满"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "该群组仅限邀请加入"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/announcements/create", func() httpx.ApiResponse[GroupAnnouncementItem] {
		item, err := h.service.CreateAnnouncement(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return httpx.ApiResponse[GroupAnnouncementItem]{Code: 10300, Message: "群组不存在"}
			}
			if errors.Is(err, ErrAccessRestricted) {
				return httpx.ApiResponse[GroupAnnouncementItem]{Code: 10200, Message: "权限不足"}
			}
			return httpx.ApiResponse[GroupAnnouncementItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	var req GroupAnnouncementsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, err := h.service.ListAnnouncements(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{"items": items}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupAnnouncementRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.AnnouncementID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, err := h.service.UpdateAnnouncement(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		if errors.Is(err, ErrAnnouncementNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "公告不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	// 内容类更新返回更新后完整对象（spec api_spec_base.md「设计规范」）
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req DeleteGroupAnnouncementRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.AnnouncementID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.DeleteAnnouncement(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) UpdateMySettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupMySettingsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UpdateMySettings(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrNotMember) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "不是群成员"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	var req LeaveGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.LeaveGroup(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrNotMember) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "不是群成员"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "群主需先转让群主身份"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListMyApplications(w http.ResponseWriter, r *http.Request) {
	var req MyApplicationsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyApplications(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.ApplicationID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.CancelApplication(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if errors.Is(err, ErrApplicationNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "申请不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}
