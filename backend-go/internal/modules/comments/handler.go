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
		logBusinessError(r, "Create", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "comments/create", func() httpx.ApiResponse[CommentItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrMomentNotFound) {
				logBusinessError(r, "Create", 10300, "瞬间不存在")
				return httpx.ApiResponse[CommentItem]{Code: 10300, Message: "瞬间不存在"}
			}
			if errors.Is(err, ErrMomentNoPermission) {
				logBusinessError(r, "Create", 10200, "权限不足")
				return httpx.ApiResponse[CommentItem]{Code: 10200, Message: "权限不足"}
			}
			logServerError(r, "Create", err)
			return httpx.ApiResponse[CommentItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListCommentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		logBusinessError(r, "List", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			logBusinessError(r, "List", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			logBusinessError(r, "List", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		logServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
		if err := idempotency.RequireKey(r); err != nil {
			logBusinessError(r, "Delete", err.Code, err.Message)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
			return
		}
		var req DeleteCommentRequest
		if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
			logBusinessError(r, "Delete", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		idempotency.ReplayOrStore(w, r, "comments/delete", func() httpx.ApiResponse[struct{}] {
			err := h.service.Delete(r.Context(), req)
			if err != nil {
				if errors.Is(err, ErrCommentNotFound) {
					logBusinessError(r, "Delete", 10300, "评论不存在")
					return httpx.ApiResponse[struct{}]{Code: 10300, Message: "评论不存在"}
				}
				if errors.Is(err, ErrPermissionDenied) {
					logBusinessError(r, "Delete", 10200, "权限不足")
					return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
				}
				logServerError(r, "Delete", err)
				return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
			}
			return httpx.Success(struct{}{})
		})
	}

func (h *Handler) ListReplies(w http.ResponseWriter, r *http.Request) {
	var req CommentRepliesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		logBusinessError(r, "ListReplies", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	list, err := h.service.ListReplies(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			logBusinessError(r, "ListReplies", 10300, "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			logBusinessError(r, "ListReplies", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		logServerError(r, "ListReplies", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	var req LikeCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		logBusinessError(r, "Like", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	err := h.service.Like(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			logBusinessError(r, "Like", 10300, "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		logServerError(r, "Like", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) {
	var req UnlikeCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		logBusinessError(r, "Unlike", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	err := h.service.Unlike(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			logBusinessError(r, "Unlike", 10300, "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		logServerError(r, "Unlike", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListLikes(w http.ResponseWriter, r *http.Request) {
	var req CommentLikesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		logBusinessError(r, "ListLikes", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.ListLikes(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			logBusinessError(r, "ListLikes", 10300, "评论不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			logBusinessError(r, "ListLikes", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		logServerError(r, "ListLikes", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code int, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}
