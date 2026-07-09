package moments

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

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
	var req CreateMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Description == "" || (req.Location.Point.Latitude == 0 && req.Location.Point.Longitude == 0 && req.Location.LocationName == "") {
		logBusinessError(r, "Create", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	// Reject out-of-range coordinates before they corrupt the S2 index.
	if !geo.IsValidPoint(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}) {
		logBusinessError(r, "Create", 10000, "参数错误: 经纬度超出有效范围")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误: 经纬度超出有效范围"))
		return
	}
	idempotency.ReplayOrStore(w, r, "moments/create", func() httpx.ApiResponse[MomentItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			logServerError(r, "Create", err)
			return httpx.ApiResponse[MomentItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req MomentOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "OpenTheDoor", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.MomentID)
	if err != nil {
		logServerError(r, "OpenTheDoor", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	if !ok {
		logBusinessError(r, "OpenTheDoor", 10300, "瞬间不存在")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	var req MyMomentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logBusinessError(r, "ListMine", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	list, err := h.service.ListMine(r.Context(), req)
	if err != nil {
		logServerError(r, "ListMine", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchMomentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logBusinessError(r, "Search", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.Keyword == nil && req.Tag == nil && req.Radius == nil {
		logBusinessError(r, "Search", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.SortBy != "" {
		switch req.SortBy {
		case "relevance", "distance", "newest", "popular":
		default:
			logBusinessError(r, "Search", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
	}
	result, err := h.service.Search(r.Context(), req)
	if err != nil {
		logServerError(r, "Search", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req EditMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "Update", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, err := h.service.Update(r.Context(), req)
	if err != nil {
		if err == ErrMomentNotFound {
			logBusinessError(r, "Update", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		if err == ErrPermissionDenied {
			logBusinessError(r, "Update", 10200, "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		logServerError(r, "Update", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "Delete", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Delete(r.Context(), req.MomentID); err != nil {
		if err == ErrMomentNotFound {
			logBusinessError(r, "Delete", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		if err == ErrPermissionDenied {
			logBusinessError(r, "Delete", 10200, "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		logServerError(r, "Delete", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	var req LikeMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "Like", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Like(r.Context(), req.MomentID); err != nil {
		if err == ErrMomentNotFound {
			logBusinessError(r, "Like", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		logServerError(r, "Like", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) {
	var req UnlikeMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "Unlike", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Unlike(r.Context(), req.MomentID); err != nil {
		if err == ErrMomentNotFound {
			logBusinessError(r, "Unlike", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		logServerError(r, "Unlike", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	var req FavoriteMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "Favorite", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Favorite(r.Context(), req.MomentID); err != nil {
		if err == ErrMomentNotFound {
			logBusinessError(r, "Favorite", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		logServerError(r, "Favorite", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Unfavorite(w http.ResponseWriter, r *http.Request) {
	var req UnfavoriteMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "Unfavorite", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Unfavorite(r.Context(), req.MomentID); err != nil {
		if err == ErrMomentNotFound {
			logBusinessError(r, "Unfavorite", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		logServerError(r, "Unfavorite", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListLikes(w http.ResponseWriter, r *http.Request) {
	var req ListLikesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		logBusinessError(r, "ListLikes", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListLikes(r.Context(), req)
	if err != nil {
		if err == ErrMomentNotFound {
			logBusinessError(r, "ListLikes", 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		logServerError(r, "ListLikes", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PaginatedLikes{Items: items, NextCursor: nextCursor}))
}

func (h *Handler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	var req MyFavoritesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logBusinessError(r, "ListFavorites", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.ListFavorites(r.Context(), req)
	if err != nil {
		logServerError(r, "ListFavorites", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
