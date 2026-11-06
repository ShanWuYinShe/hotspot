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

// decodeSuggestRequest 解析并校验 suggest 请求体（登录版/匿名版共用）。
// 失败时已写入 10000 envelope，返回 ok=false。
func decodeSuggestRequest(w http.ResponseWriter, r *http.Request) (SearchSuggestRequest, bool) {
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
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return SearchSuggestRequest{}, false
	}
	if body.Prefix == "" || len(body.Prefix) > 50 {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return SearchSuggestRequest{}, false
	}
	// reference_point 可选:缺失时由 service 按回退链解析(见 api_spec_types.md)。
	// 以指针传递「是否传入」，(0,0) 等合法坐标不会被误判为未传（P2-11）。
	var point *GeoPoint
	if body.ReferencePoint != nil && body.ReferencePoint.Latitude != nil && body.ReferencePoint.Longitude != nil {
		p := GeoPoint{Latitude: *body.ReferencePoint.Latitude, Longitude: *body.ReferencePoint.Longitude}
		if !isValidGeoPoint(p) {
			httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
			return SearchSuggestRequest{}, false
		}
		point = &p
	}
	if body.Radius != 0 && body.Radius < 100 {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "参数错误")
		return SearchSuggestRequest{}, false
	}
	return SearchSuggestRequest{Prefix: body.Prefix, ReferencePoint: point, Types: body.Types, Radius: body.Radius}, true
}

func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeSuggestRequest(w, r)
	if !ok {
		return
	}
	result, err := h.service.Suggest(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "Suggest", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

// PublicSuggest 是 /public/v1/search/suggest 的匿名实现：返回 Public* 判别
// 联合 payload，剥离 relation/social_context/viewer_state/friends_in_group
// 等查看者字段（spec api_spec_public.md 类型体系）。
func (h *Handler) PublicSuggest(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeSuggestRequest(w, r)
	if !ok {
		return
	}
	result, err := h.service.Suggest(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "PublicSuggest", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(toPublicSuggestResponse(result)))
}
