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
		httpx.WriteAPIError(w, r, "Create", err)
		return
	}
	var req CreateCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" || req.Content == "" {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误")
		return
	}
	idempotency.ReplayOrStore(w, r, "comments/create", func() httpx.ApiResponse[any] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrMomentNotFound) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeNotFound), "瞬间不存在")
				return httpx.ApiResponse[any]{Code: httpx.CodeNotFound, Message: "瞬间不存在"}
			}
			if errors.Is(err, ErrCommentTooLong) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "评论内容超过长度限制")
				return httpx.ApiResponse[any]{Code: httpx.CodeInvalidRequest, Message: "评论内容超过长度限制"}
			}
			// 契约(api_spec_errors.md)：10200 必带 data.forbidden_reason
			if errors.Is(err, ErrCommentPrivacyDenied) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeForbidden), "瞬间作者限制评论")
				data := any(httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonPrivacySetting})
				return httpx.ApiResponse[any]{Code: httpx.CodeForbidden, Message: "瞬间作者限制评论", Data: &data}
			}
			if errors.Is(err, ErrCommentBlockedByUser) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeForbidden), "你已被该用户拉黑")
				data := any(httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonBlockedByUser})
				return httpx.ApiResponse[any]{Code: httpx.CodeForbidden, Message: "你已被该用户拉黑", Data: &data}
			}
			if errors.Is(err, ErrMomentNoPermission) {
				observability.LogBusinessError(r, "Create", int(httpx.CodeForbidden), "权限不足")
				data := any(httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonPrivacySetting})
				return httpx.ApiResponse[any]{Code: httpx.CodeForbidden, Message: "权限不足", Data: &data}
			}
			observability.LogServerError(r, "Create", err)
			return httpx.ApiResponse[any]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		var data any = item
		return httpx.ApiResponse[any]{Code: httpx.CodeOK, Data: &data}
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListCommentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		httpx.WriteInvalidRequest(w, r, "List", "参数错误")
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			httpx.WriteBusinessError(w, r, "List", httpx.CodeNotFound, "瞬间不存在")
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteInvalidRequest(w, r, "List", "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "List", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

// GetDetail handles POST /api/v1/comments/open_the_door. Returns a single comment
// by id (spec api_spec_moment.md). found=false maps to httpx.CodeNotFound.
func (h *Handler) GetDetail(w http.ResponseWriter, r *http.Request) {
	var req GetCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		httpx.WriteInvalidRequest(w, r, "GetDetail", "参数错误")
		return
	}
	item, found, err := h.service.GetForDetail(r.Context(), req.CommentID)
	if err != nil {
		httpx.WriteAPIError(w, r, "GetDetail", err)
		return
	}
	if !found {
		httpx.WriteBusinessError(w, r, "GetDetail", httpx.CodeNotFound, "评论不存在")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteCommentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		httpx.WriteInvalidRequest(w, r, "Delete", "参数错误")
		return
	}
	err := h.service.Delete(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteBusinessError(w, r, "Delete", httpx.CodeNotFound, "评论不存在")
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			httpx.WriteAPIError(w, r, "Delete", httpx.Forbidden(httpx.ReasonNotResourceOwner, "仅评论作者可删除"))
			return
		}
		httpx.WriteAPIError(w, r, "Delete", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListReplies(w http.ResponseWriter, r *http.Request) {
	var req CommentRepliesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.CommentID == "" {
		httpx.WriteInvalidRequest(w, r, "ListReplies", "参数错误")
		return
	}
	list, err := h.service.ListReplies(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			httpx.WriteBusinessError(w, r, "ListReplies", httpx.CodeNotFound, "评论不存在")
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteInvalidRequest(w, r, "ListReplies", "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "ListReplies", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}
