package share

import (
	"encoding/json"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ShareType == "" || req.Payload == nil {
		logBusinessError(r, "Create", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "share/create", func() httpx.ApiResponse[ShareCreated] {
		created, err := h.service.Create(r.Context(), req)
		if err != nil {
			logServerError(r, "Create", err)
			return httpx.ApiResponse[ShareCreated]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(created)
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var req GetShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ShareCode == "" {
		logBusinessError(r, "Get", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	detail, ok, err := h.service.Get(r.Context(), req.ShareCode)
	if err != nil {
		logServerError(r, "Get", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	if !ok {
		logBusinessError(r, "Get", 10300, "分享不存在")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "分享不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(detail))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logBusinessError(r, "List", 10000, "请求体格式错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	list, err := h.service.List(r.Context())
	if err != nil {
		logServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	var req RevokeShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ShareID == "" {
		logBusinessError(r, "Revoke", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	ok, err := h.service.Revoke(r.Context(), req.ShareID)
	if err != nil {
		logServerError(r, "Revoke", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	if !ok {
		logBusinessError(r, "Revoke", 10300, "分享不存在")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "分享不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
