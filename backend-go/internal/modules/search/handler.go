package search

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if body.Prefix == "" || len(body.Prefix) > 50 || body.ReferencePoint == nil || body.ReferencePoint.Latitude == nil || body.ReferencePoint.Longitude == nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	point := GeoPoint{Latitude: *body.ReferencePoint.Latitude, Longitude: *body.ReferencePoint.Longitude}
	if !isValidGeoPoint(point) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if body.Radius != 0 && body.Radius < 100 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.Suggest(r.Context(), SearchSuggestRequest{
		Prefix:         body.Prefix,
		ReferencePoint: point,
		Types:          body.Types,
		Radius:         body.Radius,
	})
	if err != nil {
		observability.L().Error("handler error",
			"method", "Suggest",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) SearchAll(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Keyword        string  `json:"keyword,omitempty"`
		Tag            string  `json:"tag,omitempty"`
		Radius         int     `json:"radius,omitempty"`
		SortBy         string  `json:"sort_by,omitempty"`
		ReferencePoint *GeoPoint `json:"reference_point"`
		Pagination     *PaginationParams `json:"pagination,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if body.Keyword == "" && body.Tag == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if body.ReferencePoint == nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if !isValidGeoPoint(*body.ReferencePoint) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.SearchAll(r.Context(), SearchAllRequest{
		Keyword:        body.Keyword,
		Tag:            body.Tag,
		Radius:         body.Radius,
		SortBy:         body.SortBy,
		ReferencePoint: *body.ReferencePoint,
		Pagination:     body.Pagination,
	})
	if err != nil {
		observability.L().Error("handler error",
			"method", "SearchAll",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
