package mapmodule

import (
	"errors"
	"net/http"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func isValidMapDataType(dataType string) bool {
	switch normalizeMapDataType(dataType) {
	case "all", "user", "group", "moment":
		return true
	default:
		return false
	}
}

func (h *Handler) Canvas(w http.ResponseWriter, r *http.Request) {
	var req MapCanvasRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	// 跨日期变更线视口（East < West）在此拒绝——spec api_spec_public.md 约定
	// 由客户端拆分请求，服务端暂不支持（见 geo.ValidateBounds 注释）。
	if err := geo.ValidateBounds(geo.Bounds{North: req.Viewport.Bounds.North, South: req.Viewport.Bounds.South, East: req.Viewport.Bounds.East, West: req.Viewport.Bounds.West}); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if req.Filter != nil {
		if !isValidMapDataType(req.Filter.DataType) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		if req.Filter.TimeRange != nil {
			var start, end time.Time
			var err error
			if req.Filter.TimeRange.StartTime != "" {
				start, err = time.Parse(time.RFC3339Nano, req.Filter.TimeRange.StartTime)
				if err != nil {
					httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
					return
				}
			}
			if req.Filter.TimeRange.EndTime != "" {
				end, err = time.Parse(time.RFC3339Nano, req.Filter.TimeRange.EndTime)
				if err != nil {
					httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
					return
				}
			}
			if !start.IsZero() && !end.IsZero() && start.After(end) {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
				return
			}
		}
	}
	resp, err := h.service.Canvas(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		httpx.WriteAPIError(w, r, "Canvas", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}
