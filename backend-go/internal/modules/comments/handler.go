package comments

import (
	"errors"
	"net/http"

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
		observability.LogBusinessError(r, "Create", int(err.Code), err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" || req.Content == "" {
		observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "comments/create", func() httpx.ApiResponse[CommentItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrMomentNotFound) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeNotFound), "瞬间不存在")
				return httpx.ApiResponse[CommentItem]{Code: httpx.CodeNotFound, Message: "瞬间不存在"}
			}
			if errors.Is(err, ErrMomentNoPermission) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeForbidden), "权限不足")
				return httpx.ApiResponse[CommentItem]{Code: httpx.CodeForbidden, Message: "权限不足"}
			}
			observability.LogServerError(r, "Create", err)
			return httpx.ApiResponse[CommentItem]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListCommentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		observability.LogBusinessError(r, "List", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			observability.LogBusinessError(r, "List", int(httpx.CodeNotFound), "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "瞬间不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			observability.LogBusinessError(r, "List", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		observability.LogServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

// GetDetail handles POST /api/v1/comments/get_detail. Returns a single comment
// by id (spec api_spec_moment.md). found=false maps to httpx.CodeNotFound.
func (h *Handler) GetDetail(w http.ResponseWriter, r *http.Request) {
	var req GetCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		observability.LogBusinessError(r, "GetDetail", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	item, found, err := h.service.GetForDetail(r.Context(), req.CommentID)
	if err != nil {
		observability.LogServerError(r, "GetDetail", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	if !found {
		observability.LogBusinessError(r, "GetDetail", int(httpx.CodeNotFound), "评论不存在")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "评论不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		observability.LogBusinessError(r, "Delete", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	err := h.service.Delete(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			observability.LogBusinessError(r, "Delete", int(httpx.CodeNotFound), "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "评论不存在"))
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			observability.LogBusinessError(r, "Delete", int(httpx.CodeForbidden), "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		observability.LogServerError(r, "Delete", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListReplies(w http.ResponseWriter, r *http.Request) {
	var req CommentRepliesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		observability.LogBusinessError(r, "ListReplies", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	list, err := h.service.ListReplies(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			observability.LogBusinessError(r, "ListReplies", int(httpx.CodeNotFound), "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "评论不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			observability.LogBusinessError(r, "ListReplies", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		observability.LogServerError(r, "ListReplies", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}
