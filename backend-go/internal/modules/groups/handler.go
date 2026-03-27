package groups

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Location.LocationName == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/create", func() httpx.ApiResponse[GroupItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			return httpx.ApiResponse[GroupItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req GroupOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.GroupID)
	if err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	list, err := h.service.Search(r.Context(), req.Keyword)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.ClearPassword != nil && *req.ClearPassword && req.Password != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "密码参数冲突"))
		return
	}
	if err := h.service.Update(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrPermissionDenied {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req DismissGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/dismiss", func() httpx.ApiResponse[struct{}] {
		if err := h.service.Dismiss(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.Error(10300, "群组不存在")
			}
			if err == ErrPermissionDenied {
				return httpx.Error(10400, "权限不足")
			}
			return httpx.Error(10600, "服务暂时不可用")
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	var req GroupMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListMembers(r.Context(), req)
	if err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":      items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) SearchMembers(w http.ResponseWriter, r *http.Request) {
	var req SearchGroupMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.Keyword == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.SearchMembers(r.Context(), req)
	if err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":      items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	var req MyGroupsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyGroups(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":      items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req JoinGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/join", func() httpx.ApiResponse[JoinGroupResult] {
		result, err := h.service.JoinGroup(r.Context(), req)
		if err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[JoinGroupResult]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrInvalidPassword {
				return httpx.ApiResponse[JoinGroupResult]{Code: 10400, Message: "密码错误"}
			}
			if err == ErrPermissionDenied {
				return httpx.ApiResponse[JoinGroupResult]{Code: 10400, Message: "该群组仅限邀请加入"}
			}
			return httpx.ApiResponse[JoinGroupResult]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(result)
	})
}

func (h *Handler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateGroupAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.Title == "" || req.Content == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/announcements/create", func() httpx.ApiResponse[GroupAnnouncementItem] {
		item, err := h.service.CreateAnnouncement(r.Context(), req)
		if err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[GroupAnnouncementItem]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrPermissionDenied {
				return httpx.ApiResponse[GroupAnnouncementItem]{Code: 10400, Message: "权限不足"}
			}
			return httpx.ApiResponse[GroupAnnouncementItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	var req GroupAnnouncementsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, err := h.service.ListAnnouncements(r.Context(), req)
	if err != nil {
		if err == ErrGroupNotFound {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.AnnouncementID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UpdateAnnouncement(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrPermissionDenied {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req DeleteGroupAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.AnnouncementID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.DeleteAnnouncement(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrPermissionDenied {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	var req GroupApplicationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListApplications(r.Context(), req)
	if err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrPermissionDenied {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":      items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ApproveGroupApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApplicationID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/applications/approve", func() httpx.ApiResponse[struct{}] {
		if err := h.service.ApproveApplication(r.Context(), req); err != nil {
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) RejectApplication(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RejectGroupApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApplicationID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/applications/reject", func() httpx.ApiResponse[struct{}] {
		if err := h.service.RejectApplication(r.Context(), req); err != nil {
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}
