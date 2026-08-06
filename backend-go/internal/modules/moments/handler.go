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
		observability.LogBusinessError(r, "Create", int(err.Code), err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Description == "" || (req.Location.Point.Latitude == 0 && req.Location.Point.Longitude == 0 && req.Location.LocationName == "") {
		observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if utf8.RuneCountInString(req.Description) > momentDescriptionMaxRunes {
		observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "内容超过长度限制")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "内容超过长度限制"))
		return
	}
	// Reject out-of-range coordinates before they corrupt the S2 index.
	if !geo.IsValidPoint(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}) {
		observability.LogBusinessError(r, "Create", int(httpx.CodeInvalidRequest), "参数错误: 经纬度超出有效范围")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误: 经纬度超出有效范围"))
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
		observability.LogBusinessError(r, "OpenTheDoor", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.MomentID)
	if err != nil {
		observability.LogServerError(r, "OpenTheDoor", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	if !ok {
		observability.LogBusinessError(r, "OpenTheDoor", int(httpx.CodeNotFound), "瞬间不存在")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "瞬间不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	var req MyMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		observability.LogBusinessError(r, "ListMine", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	list, err := h.service.ListMine(r.Context(), req)
	if err != nil {
		observability.LogServerError(r, "ListMine", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		observability.LogBusinessError(r, "Search", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if req.Keyword == nil && req.Tag == nil && req.Radius == nil {
		observability.LogBusinessError(r, "Search", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if req.SortBy != "" {
		switch req.SortBy {
		case "relevance", "distance", "newest", "popular":
		default:
			observability.LogBusinessError(r, "Search", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
	}
	result, err := h.service.Search(r.Context(), req)
	if err != nil {
		observability.LogServerError(r, "Search", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req EditMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		observability.LogBusinessError(r, "Update", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	item, err := h.service.Update(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			observability.LogBusinessError(r, "Update", int(httpx.CodeNotFound), "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "瞬间不存在"))
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			observability.LogBusinessError(r, "Update", int(httpx.CodeForbidden), "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		if errors.Is(err, ErrTooManyImages) {
			observability.LogBusinessError(r, "Update", int(httpx.CodeInvalidRequest), "图片数量超过上限")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "图片数量超过上限"))
			return
		}
		if errors.Is(err, ErrImagesNotOwned) {
			observability.LogBusinessError(r, "Update", int(httpx.CodeInvalidRequest), "图片校验失败")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "图片校验失败"))
			return
		}
		observability.LogServerError(r, "Update", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		observability.LogBusinessError(r, "Delete", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.Delete(r.Context(), req.MomentID); err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			observability.LogBusinessError(r, "Delete", int(httpx.CodeNotFound), "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "瞬间不存在"))
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			observability.LogBusinessError(r, "Delete", int(httpx.CodeForbidden), "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
			return
		}
		observability.LogServerError(r, "Delete", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

// Favorite is the unified two-state favorite endpoint. active=true favorites
// the moment; active=false unfavorites it. Idempotent.
func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	var req FavoriteMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		observability.LogBusinessError(r, "Favorite", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		observability.LogServerError(r, "Favorite", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	var req MyFavoritesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		observability.LogBusinessError(r, "ListFavorites", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	result, err := h.service.ListFavorites(r.Context(), req)
	if err != nil {
		observability.LogServerError(r, "ListFavorites", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
