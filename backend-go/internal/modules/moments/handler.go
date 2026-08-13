package moments

import (
	"errors"
	"net/http"
	"unicode/utf8"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

// momentDescriptionMaxRunes 限制动态正文长度（与 ClientConfig.moment_description_max_length 一致）。
const momentDescriptionMaxRunes = 2000

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteAPIError(w, r, "Create", err)
		return
	}
	var req CreateMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Description == "" || (req.Location.Point.Latitude == 0 && req.Location.Point.Longitude == 0 && req.Location.LocationName == "") {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误")
		return
	}
	if utf8.RuneCountInString(req.Description) > momentDescriptionMaxRunes {
		httpx.WriteInvalidRequest(w, r, "Create", "内容超过长度限制")
		return
	}
	// Reject out-of-range coordinates before they corrupt the S2 index.
	if !geo.IsValidPoint(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}) {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误: 经纬度超出有效范围")
		return
	}
	idempotency.ReplayOrStore(w, r, "moments/create", func() httpx.ApiResponse[MomentItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, ErrTooManyImages):
				observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "图片数量超过上限")
				return httpx.ApiResponse[MomentItem]{Code: httpx.CodeInvalidRequest, Message: "图片数量超过上限"}
			case errors.Is(err, ErrImagesNotOwned):
				observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "图片校验失败")
				return httpx.ApiResponse[MomentItem]{Code: httpx.CodeInvalidRequest, Message: "图片校验失败"}
			}
			observability.LogServerError(r, "Create", err)
			return httpx.ApiResponse[MomentItem]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req MomentOpenTheDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		httpx.WriteInvalidRequest(w, r, "OpenTheDoor", "参数错误")
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.MomentID)
	if err != nil {
		httpx.WriteAPIError(w, r, "OpenTheDoor", err)
		return
	}
	if !ok {
		httpx.WriteBusinessError(w, r, "OpenTheDoor", httpx.CodeNotFound, "瞬间不存在")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	var req MyMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "ListMine", "参数错误")
		return
	}
	list, err := h.service.ListMine(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "ListMine", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Search", "参数错误")
		return
	}
	if req.Keyword == nil && req.Tag == nil {
		httpx.WriteInvalidRequest(w, r, "Search", "参数错误")
		return
	}
	if req.SortBy != "" {
		switch req.SortBy {
		case "relevance", "newest", "popular":
		default:
			httpx.WriteInvalidRequest(w, r, "Search", "参数错误")
			return
		}
	}
	result, err := h.service.Search(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "Search", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误")
		return
	}
	item, err := h.service.Update(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeNotFound, "瞬间不存在")
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeForbidden, "权限不足")
			return
		}
		if errors.Is(err, ErrTooManyImages) {
			httpx.WriteInvalidRequest(w, r, "Update", "图片数量超过上限")
			return
		}
		if errors.Is(err, ErrImagesNotOwned) {
			httpx.WriteInvalidRequest(w, r, "Update", "图片校验失败")
			return
		}
		httpx.WriteAPIError(w, r, "Update", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		httpx.WriteInvalidRequest(w, r, "Delete", "参数错误")
		return
	}
	if err := h.service.Delete(r.Context(), req.MomentID); err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			httpx.WriteBusinessError(w, r, "Delete", httpx.CodeNotFound, "瞬间不存在")
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			httpx.WriteBusinessError(w, r, "Delete", httpx.CodeForbidden, "权限不足")
			return
		}
		httpx.WriteAPIError(w, r, "Delete", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

// Favorite is the unified two-state favorite endpoint. active=true favorites
// the moment; active=false unfavorites it. Idempotent.
func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	var req FavoriteMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		httpx.WriteInvalidRequest(w, r, "Favorite", "参数错误")
		return
	}
	var err error
	if req.Active {
		err = h.service.Favorite(r.Context(), req.MomentID)
	} else {
		err = h.service.Unfavorite(r.Context(), req.MomentID)
	}
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			method := "Favorite"
			if !req.Active {
				method = "Unfavorite"
			}
			observability.LogBusinessError(r, method, int(httpx.CodeNotFound), "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "瞬间不存在"))
			return
		}
		httpx.WriteAPIError(w, r, "Favorite", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	var req MyFavoritesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "ListFavorites", "参数错误")
		return
	}
	result, err := h.service.ListFavorites(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "ListFavorites", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
