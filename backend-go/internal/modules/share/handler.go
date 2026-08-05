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

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code int, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		logBusinessError(r, "Create", err.Code, err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateShareRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "Create", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "share/create", func() httpx.ApiResponse[ShareCreated] {
		created, err := h.service.Create(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidShareRequest):
				return httpx.ApiResponse[ShareCreated]{Code: 10000, Message: "参数错误"}
			case errors.Is(err, ErrShareNotFound):
				return httpx.ApiResponse[ShareCreated]{Code: 10300, Message: "资源不存在"}
			default:
				logServerError(r, "Create", err)
				return httpx.ApiResponse[ShareCreated]{Code: 10600, Message: "服务暂时不可用"}
			}
		}
		return httpx.Success(created)
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var req GetShareRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "Get", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	detail, err := h.service.Get(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidShareRequest):
			logBusinessError(r, "Get", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		case errors.Is(err, ErrShareNotFound):
			logBusinessError(r, "Get", 10300, "资源不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "资源不存在"))
		default:
			logServerError(r, "Get", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(detail))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ShareListRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "List", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidShareRequest):
			logBusinessError(r, "List", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		default:
			logServerError(r, "List", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	var req RevokeShareRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "Revoke", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Revoke(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrInvalidShareRequest):
			logBusinessError(r, "Revoke", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		case errors.Is(err, ErrSharePermission):
			// 隐私一致性（spec api_spec_share.md）：share 存在但无权限时统一
			// 返回 10300，不通过 10200 泄露 share 的存在性。
			logBusinessError(r, "Revoke", 10300, "资源不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "资源不存在"))
		case errors.Is(err, ErrShareNotFound):
			logBusinessError(r, "Revoke", 10300, "资源不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "资源不存在"))
		default:
			logServerError(r, "Revoke", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
