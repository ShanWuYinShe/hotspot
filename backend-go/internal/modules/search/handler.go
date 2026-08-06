package search

import (
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prefix         string `json:"prefix"`
		ReferencePoint *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"reference_point"`
		Types  []string `json:"types,omitempty"`
		Radius int      `json:"radius,omitempty"`
	}
	if err := httpx.DecodeStrictJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if body.Prefix == "" || len(body.Prefix) > 50 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// reference_point 可选:缺失时由 service 按回退链解析(见 api_spec_types.md)
	var point GeoPoint
	if body.ReferencePoint != nil && body.ReferencePoint.Latitude != nil && body.ReferencePoint.Longitude != nil {
		point = GeoPoint{Latitude: *body.ReferencePoint.Latitude, Longitude: *body.ReferencePoint.Longitude}
		if !isValidGeoPoint(point) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
	}
	if body.Radius != 0 && body.Radius < 100 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	result, err := h.service.Suggest(r.Context(), SearchSuggestRequest{
		Prefix:         body.Prefix,
		ReferencePoint: point,
		Types:          body.Types,
		Radius:         body.Radius,
	})
	if err != nil {
		httpx.WriteAPIError(w, r, "Suggest", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
