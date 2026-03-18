package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend-go/internal/api"
	apierr "backend-go/internal/errors"
	"backend-go/internal/service"
	"backend-go/internal/types"
)

type MapHandler struct {
	service service.MapService
}

func NewMapHandler(svc service.MapService) *MapHandler {
	return &MapHandler{service: svc}
}

// RegisterMapRoutes 在路由树下注册地图相关端点。
func RegisterMapRoutes(r chi.Router, h *MapHandler) {
	r.Route("/api/v1/map", func(r chi.Router) {
		r.Post("/canvas", h.Canvas)
	})
}

// Canvas 对应 POST /api/v1/map/canvas。
func (h *MapHandler) Canvas(w http.ResponseWriter, r *http.Request) {
	var req types.MapCanvasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, r, apierr.ErrInvalidRequest("invalid_request_body").Code, apierr.ErrInvalidRequest("invalid_request_body").Message)
		return
	}

	resp, err := h.service.Canvas(r.Context(), req)
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		// 未归类错误统一映射为内部错误
		e := apierr.ErrInternal("")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}

	api.WriteSuccess(w, r, resp)
}
