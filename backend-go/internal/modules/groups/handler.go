package groups

import (
	"encoding/json"
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
		observability.L().Error("handler error",
			"method", "OpenTheDoor",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
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
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		observability.L().Error("handler error",
			"method", "Search",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
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
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
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
			if err == ErrAccessRestricted {
				return httpx.Error(10200, "权限不足")
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
		"items":       items,
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
		"items":       items,
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
		"items":       items,
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
			if err == ErrAlreadyApplied {
				return httpx.ApiResponse[JoinGroupResult]{Code: 10400, Message: "已申请过"}
			}
			if err == ErrGroupFull {
				return httpx.ApiResponse[JoinGroupResult]{Code: 10400, Message: "群组已满"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[JoinGroupResult]{Code: 10200, Message: "该群组仅限邀请加入"}
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
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[GroupAnnouncementItem]{Code: 10200, Message: "权限不足"}
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
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
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
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
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
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
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
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
			}
			if err == ErrAlreadyProcessed {
				return httpx.ApiResponse[struct{}]{Code: 10400, Message: "申请已处理"}
			}
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
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
			}
			if err == ErrAlreadyProcessed {
				return httpx.ApiResponse[struct{}]{Code: 10400, Message: "申请已处理"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) UpdateMySettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupMySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UpdateMySettings(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrNotMember {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "不是群成员"))
			return
		}
		if err == ErrAccessRestricted {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.LeaveGroup(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrNotMember {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "不是群成员"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "群主需先转让群主身份"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListMyApplications(w http.ResponseWriter, r *http.Request) {
	var req MyApplicationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApplicationID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.CancelApplication(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListMyInvites(w http.ResponseWriter, r *http.Request) {
	var req MyInvitesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyInvites(r.Context(), req)
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

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/me/invites/accept", func() httpx.ApiResponse[struct{}] {
		if err := h.service.AcceptInvite(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) RejectInvite(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RejectInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/me/invites/reject", func() httpx.ApiResponse[struct{}] {
		if err := h.service.RejectInvite(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrInviteExpired {
				return httpx.ApiResponse[struct{}]{Code: 10400, Message: "邀请已失效"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/members/invite", func() httpx.ApiResponse[Invitation] {
		item, err := h.service.InviteMember(r.Context(), req)
		if err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[Invitation]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[Invitation]{Code: 10200, Message: "权限不足"}
			}
			if err == ErrAlreadyMember {
				return httpx.ApiResponse[Invitation]{Code: 10400, Message: "已是群成员"}
			}
			return httpx.ApiResponse[Invitation]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) ListSentInvites(w http.ResponseWriter, r *http.Request) {
	var req GroupSentInvitesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListSentInvites(r.Context(), req)
	if err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
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

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	var req RevokeSentInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.RevokeInvite(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) SetMemberRole(w http.ResponseWriter, r *http.Request) {
	var req SetMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" || req.Role == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.SetMemberRole(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) MuteMember(w http.ResponseWriter, r *http.Request) {
	var req MuteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.MuteMember(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) UnmuteMember(w http.ResponseWriter, r *http.Request) {
	var req UnmuteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UnmuteMember(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) BanMember(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req BanMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/members/ban", func() httpx.ApiResponse[struct{}] {
		if err := h.service.BanMember(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) UnbanMember(w http.ResponseWriter, r *http.Request) {
	var req UnbanMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UnbanMember(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListBannedMembers(w http.ResponseWriter, r *http.Request) {
	var req BannedMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListBannedMembers(r.Context(), req)
	if err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
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

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	var req RemoveMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.RemoveMember(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	var req TransferOwnershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.NewOwnerID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.TransferOwnership(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		if err == ErrTargetNotMember {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "目标用户不是群成员"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}
