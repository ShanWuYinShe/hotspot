package comments

import (
	"encoding/json"
	"errors"
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
	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" || req.Content == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "comments/create", func() httpx.ApiResponse[CommentItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrMomentNotFound) {
				return httpx.ApiResponse[CommentItem]{Code: 10300, Message: "瞬间不存在"}
			}
			return httpx.ApiResponse[CommentItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListCommentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	list, err := h.service.List(r.Context(), req.MomentID)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CommentID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	err := h.service.Delete(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListReplies(w http.ResponseWriter, r *http.Request) {
	var req CommentRepliesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CommentID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	list, err := h.service.ListReplies(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	var req LikeCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CommentID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	err := h.service.Like(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) {
	var req UnlikeCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CommentID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	err := h.service.Unlike(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListLikes(w http.ResponseWriter, r *http.Request) {
	var req CommentLikesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CommentID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.ListLikes(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "评论不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
