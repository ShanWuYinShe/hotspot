package mapmodule

import (
	"errors"
	"net/http"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
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
				start, err = time.Parse(time.RFC3339, req.Filter.TimeRange.StartTime)
				if err != nil {
					httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
					return
				}
			}
			if req.Filter.TimeRange.EndTime != "" {
				end, err = time.Parse(time.RFC3339, req.Filter.TimeRange.EndTime)
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
		observability.LogServerError(r, "Canvas", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}
