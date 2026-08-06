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
		logBusinessError(r, "Create", err.Code, err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" || req.Content == "" {
		logBusinessError(r, "Create", httpx.CodeInvalidRequest, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "comments/create", func() httpx.ApiResponse[CommentItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrMomentNotFound) {
				logBusinessError(r, "Create", httpx.CodeNotFound, "瞬间不存在")
				return httpx.ApiResponse[CommentItem]{Code: httpx.CodeNotFound, Message: "瞬间不存在"}
			}
			if errors.Is(err, ErrMomentNoPermission) {
				logBusinessError(r, "Create", httpx.CodeForbidden, "权限不足")
				return httpx.ApiResponse[CommentItem]{Code: httpx.CodeForbidden, Message: "权限不足"}
			}
			logServerError(r, "Create", err)
			return httpx.ApiResponse[CommentItem]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListCommentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		logBusinessError(r, "List", httpx.CodeInvalidRequest, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			logBusinessError(r, "List", httpx.CodeNotFound, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "瞬间不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			logBusinessError(r, "List", httpx.CodeInvalidRequest, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		logServerError(r, "List", err)
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
		logBusinessError(r, "GetDetail", httpx.CodeInvalidRequest, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	item, found, err := h.service.GetForDetail(r.Context(), req.CommentID)
	if err != nil {
		logServerError(r, "GetDetail", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	if !found {
		logBusinessError(r, "GetDetail", httpx.CodeNotFound, "评论不存在")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "评论不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		logBusinessError(r, "Delete", httpx.CodeInvalidRequest, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	err := h.service.Delete(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			logBusinessError(r, "Delete", httpx.CodeNotFound, "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "评论不存在"))
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			logBusinessError(r, "Delete", httpx.CodeForbidden, "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		logServerError(r, "Delete", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListReplies(w http.ResponseWriter, r *http.Request) {
	var req CommentRepliesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		logBusinessError(r, "ListReplies", httpx.CodeInvalidRequest, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	list, err := h.service.ListReplies(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			logBusinessError(r, "ListReplies", httpx.CodeNotFound, "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "评论不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			logBusinessError(r, "ListReplies", httpx.CodeInvalidRequest, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		logServerError(r, "ListReplies", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code httpx.Code, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}
