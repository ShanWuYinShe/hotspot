package reports

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
	var req CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetType == "" || req.TargetID == "" || req.Reason == "" {
		logBusinessError(r, "Create", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "reports/create", func() httpx.ApiResponse[ReportCreated] {
		created, err := h.service.Create(r.Context(), req)
		if err != nil {
			switch err.Error() {
			case "reports: invalid request":
				logBusinessError(r, "Create", 10000, "参数错误")
				return httpx.ApiResponse[ReportCreated]{Code: 10000, Message: "参数错误"}
			case "reports: target not found":
				logBusinessError(r, "Create", 10300, "举报目标不存在")
				return httpx.ApiResponse[ReportCreated]{Code: 10300, Message: "举报目标不存在"}
			case "reports: self report", "reports: duplicate report":
				logBusinessError(r, "Create", 10400, "举报被拒绝")
				return httpx.ApiResponse[ReportCreated]{Code: 10400, Message: "举报被拒绝"}
			default:
				logServerError(r, "Create", err)
				return httpx.ApiResponse[ReportCreated]{Code: 10600, Message: "服务暂时不可用"}
			}
		}
		return httpx.Success(created)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req MyReportsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logBusinessError(r, "List", 10000, "请求体格式错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		logServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}
