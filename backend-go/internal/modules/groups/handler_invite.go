package groups

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

func (h *Handler) ListMyInvites(w http.ResponseWriter, r *http.Request) {
	var req MyInvitesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyInvites(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.handler", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PaginatedInvites{Items: items, NextCursor: types.StrPtr(nextCursor)}))
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req AcceptInviteRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.AcceptInvite(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrBannedInGroup) {
			httpx.WriteAPIError(w, r, "AcceptInvite", httpx.Forbidden(httpx.ReasonBannedInGroup, "已被群封禁，无法加入"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) RejectInvite(w http.ResponseWriter, r *http.Request) {
	var req RejectInviteRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.RejectInvite(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrInviteExpired) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "邀请已失效"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	var req InviteMemberRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	item, err := h.service.InviteMember(r.Context(), req)
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
			httpx.WriteAPIError(w, r, "groups.handler", httpx.Forbidden(httpx.ReasonNotGroupAdmin, "权限不足"))
			return
		}
		if errors.Is(err, ErrAlreadyMember) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "已是群成员"))
			return
		}
		// spec api_spec_group.md → invites/create 错误码：邀请已被封禁的用户 10400。
		if errors.Is(err, ErrBannedInGroup) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "该用户已被群封禁，无法邀请"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) ListSentInvites(w http.ResponseWriter, r *http.Request) {
	var req GroupSentInvitesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListSentInvites(r.Context(), req)
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
			httpx.WriteAPIError(w, r, "groups.handler", httpx.Forbidden(httpx.ReasonNotGroupAdmin, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PaginatedSentInvites{Items: items, NextCursor: types.StrPtr(nextCursor)}))
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	var req RevokeSentInviteRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.RevokeInvite(r.Context(), req); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		if errors.Is(err, ErrAccessRestricted) {
			httpx.WriteAPIError(w, r, "groups.handler", httpx.Forbidden(httpx.ReasonNotGroupAdmin, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}
