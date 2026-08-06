package share

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteAPIError(w, r, "Create", err)
		return
	}
	var req CreateShareRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误")
		return
	}
	idempotency.ReplayOrStore(w, r, "share/create", func() httpx.ApiResponse[ShareCreated] {
		created, err := h.service.Create(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidShareRequest):
				return httpx.ApiResponse[ShareCreated]{Code: httpx.CodeInvalidRequest, Message: "参数错误"}
			case errors.Is(err, ErrShareNotFound):
				return httpx.ApiResponse[ShareCreated]{Code: httpx.CodeNotFound, Message: "资源不存在"}
			default:
				observability.LogServerError(r, "Create", err)
				return httpx.ApiResponse[ShareCreated]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
			}
		}
		return httpx.Success(created)
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var req GetShareRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Get", "参数错误")
		return
	}
	detail, err := h.service.Get(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidShareRequest):
			observability.LogBusinessError(r, "Get", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		case errors.Is(err, ErrShareNotFound):
			observability.LogBusinessError(r, "Get", int(httpx.CodeNotFound), "资源不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "资源不存在"))
		default:
			observability.LogServerError(r, "Get", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(detail))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ShareListRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "List", "参数错误")
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidShareRequest):
			observability.LogBusinessError(r, "List", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		default:
			observability.LogServerError(r, "List", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	var req RevokeShareRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Revoke", "参数错误")
		return
	}
	if err := h.service.Revoke(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrInvalidShareRequest):
			observability.LogBusinessError(r, "Revoke", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		case errors.Is(err, ErrSharePermission):
			// 隐私一致性（spec api_spec_share.md）：share 存在但无权限时统一
			// 返回 httpx.CodeNotFound，不通过 httpx.CodeForbidden 泄露 share 的存在性。
			observability.LogBusinessError(r, "Revoke", int(httpx.CodeNotFound), "资源不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "资源不存在"))
		case errors.Is(err, ErrShareNotFound):
			observability.LogBusinessError(r, "Revoke", int(httpx.CodeNotFound), "资源不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "资源不存在"))
		default:
			observability.LogServerError(r, "Revoke", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
