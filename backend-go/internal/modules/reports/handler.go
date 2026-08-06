package reports

import (
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateReportRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetType == "" || req.TargetID == "" || req.Reason == "" {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误")
		return
	}
	created, err := h.service.Create(r.Context(), req)
	if err != nil {
		switch err.Error() {
		case "reports: invalid request":
			observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		case "reports: target not found":
			observability.LogBusinessError(r, "Create", int(httpx.CodeNotFound), "举报目标不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "举报目标不存在"))
		case "reports: self report":
			observability.LogBusinessError(r, "Create", int(httpx.CodeConflict), "举报被拒绝")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "举报被拒绝"))
		default:
			observability.LogServerError(r, "Create", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(created))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req MyReportsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "List", "请求体格式错误")
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "List", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}
