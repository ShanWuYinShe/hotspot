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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Description == "" || (req.Location.Point.Latitude == 0 && req.Location.Point.Longitude == 0 && req.Location.LocationName == "") {
		logBusinessError(r, "Create", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if utf8.RuneCountInString(req.Description) > momentDescriptionMaxRunes {
		logBusinessError(r, "Create", 10000, "内容超过长度限制")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "内容超过长度限制"))
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
			switch {
			case errors.Is(err, ErrTooManyImages):
				logBusinessError(r, "Create", 10000, "图片数量超过上限")
				return httpx.ApiResponse[MomentItem]{Code: 10000, Message: "图片数量超过上限"}
			case errors.Is(err, ErrImagesNotOwned):
				logBusinessError(r, "Create", 10000, "图片校验失败")
				return httpx.ApiResponse[MomentItem]{Code: 10000, Message: "图片校验失败"}
			}
			logServerError(r, "Create", err)
			return httpx.ApiResponse[MomentItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req MomentOpenTheDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
		if err := idempotency.RequireKey(r); err != nil {
			logBusinessError(r, "Update", err.Code, err.Message)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
			return
		}
		var req EditMomentRequest
		if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
			logBusinessError(r, "Update", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		idempotency.ReplayOrStore(w, r, "moments/update", func() httpx.ApiResponse[MomentItem] {
			item, err := h.service.Update(r.Context(), req)
			if err != nil {
				if errors.Is(err, ErrMomentNotFound) {
					logBusinessError(r, "Update", 10300, "瞬间不存在")
					return httpx.ApiResponse[MomentItem]{Code: 10300, Message: "瞬间不存在"}
				}
				if errors.Is(err, ErrPermissionDenied) {
					logBusinessError(r, "Update", 10200, "权限不足")
					return httpx.ApiResponse[MomentItem]{Code: 10200, Message: "权限不足"}
				}
				if errors.Is(err, ErrTooManyImages) {
					logBusinessError(r, "Update", 10000, "图片数量超过上限")
					return httpx.ApiResponse[MomentItem]{Code: 10000, Message: "图片数量超过上限"}
				}
				if errors.Is(err, ErrImagesNotOwned) {
					logBusinessError(r, "Update", 10000, "图片校验失败")
					return httpx.ApiResponse[MomentItem]{Code: 10000, Message: "图片校验失败"}
				}
				logServerError(r, "Update", err)
				return httpx.ApiResponse[MomentItem]{Code: 10600, Message: "服务暂时不可用"}
			}
			return httpx.Success(item)
		})
	}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
		if err := idempotency.RequireKey(r); err != nil {
			logBusinessError(r, "Delete", err.Code, err.Message)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
			return
		}
		var req DeleteMomentRequest
		if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
			logBusinessError(r, "Delete", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		idempotency.ReplayOrStore(w, r, "moments/delete", func() httpx.ApiResponse[struct{}] {
			if err := h.service.Delete(r.Context(), req.MomentID); err != nil {
				if errors.Is(err, ErrMomentNotFound) {
					logBusinessError(r, "Delete", 10300, "瞬间不存在")
					return httpx.ApiResponse[struct{}]{Code: 10300, Message: "瞬间不存在"}
				}
				if errors.Is(err, ErrPermissionDenied) {
					logBusinessError(r, "Delete", 10200, "权限不足")
					return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
				}
				logServerError(r, "Delete", err)
				return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
			}
			return httpx.Success(struct{}{})
		})
	}

// Favorite is the unified two-state favorite endpoint. active=true favorites
// the moment; active=false unfavorites it. Idempotent.
func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	var req FavoriteMomentRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MomentID == "" {
		logBusinessError(r, "Favorite", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
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
			logBusinessError(r, method, 10300, "瞬间不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
			return
		}
		logServerError(r, "Favorite", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	var req MyFavoritesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
