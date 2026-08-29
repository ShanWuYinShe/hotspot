package moments

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

// momentDescriptionMaxRunes 已下沉到 Service.ensureDescriptionLength：
// 长度上限随 ClientConfig.moment_description_max_length 配置走，
// handler 层不再持有硬编码副本（避免配置调整后两端口径分裂）。

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
	// location 必填（spec moment.md create）；指针判空替代旧的
	// (0,0)+空名哨兵——赤道/本初子午线交点是合法坐标，哨兵会误拒。
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Description == "" || req.Location == nil {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误")
		return
	}
	// privacy_level 限定枚举：任意字符串落库会让 private 过滤（可见性控制）失效。
	if !isValidPrivacyLevel(req.PrivacyLevel) {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误: privacy_level 仅允许 public/private")
		return
	}
	// Reject out-of-range coordinates before they corrupt the S2 index.
	if !geo.IsValidPoint(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}) {
		httpx.WriteInvalidRequest(w, r, "Create", "参数错误: 经纬度超出有效范围")
		return
	}
	idempotency.ReplayOrStore(w, r, "moments/create", func() httpx.ApiResponse[OwnerMomentItem] {
		item, err := h.service.Create(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, ErrDescriptionTooLong):
				return httpx.Business[OwnerMomentItem](r, "Create", httpx.CodeInvalidRequest, "内容超过长度限制")
			case errors.Is(err, ErrTooManyMentions):
				return httpx.Business[OwnerMomentItem](r, "Create", httpx.CodeInvalidRequest, "mentions 数量超过上限")
			case errors.Is(err, ErrTooManyImages):
				return httpx.Business[OwnerMomentItem](r, "Create", httpx.CodeInvalidRequest, "图片数量超过上限")
			case errors.Is(err, ErrImagesNotOwned):
				// image_file_ids 中媒体资源不存在或不属于当前用户 → 10300
				//（spec api_spec_moment.md moments/create 错误码）。
				return httpx.Business[OwnerMomentItem](r, "Create", httpx.CodeNotFound, "图片不存在或不属于当前用户")
			case errors.Is(err, ErrBlockedTag):
				return httpx.Business[OwnerMomentItem](r, "Create", httpx.CodeConflict, "标签包含被屏蔽的内容")
			}
			observability.LogServerError(r, "Create", err)
			return httpx.ApiResponse[OwnerMomentItem]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
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

func (h *Handler) ListCreated(w http.ResponseWriter, r *http.Request) {
	var req ListCreatedMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "ListCreated", "参数错误")
		return
	}
	list, err := h.service.ListCreated(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "ListCreated", err)
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
	if req.SortBy != "" {
		switch req.SortBy {
		case "relevance", "newest", "popular":
		default:
			httpx.WriteInvalidRequest(w, r, "Search", "参数错误")
			return
		}
	}
	// 热门模式（spec moment.md → moments/search）：keyword/tag 均缺省且显式传
	// sort_by='popular' 时进入（承接原 feed/trending 的发现页首屏能力）；
	// 热门模式与输入条件互斥（同传 10000），三者全缺省仍 10000。
	keywordProvided := req.Keyword != nil && *req.Keyword != ""
	tagProvided := req.Tag != nil && *req.Tag != ""
	if req.SortBy == "popular" && (keywordProvided || tagProvided) {
		httpx.WriteInvalidRequest(w, r, "Search", "参数错误")
		return
	}
	if !keywordProvided && !tagProvided && req.SortBy != "popular" {
		httpx.WriteInvalidRequest(w, r, "Search", "参数错误")
		return
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
	// 与 Create 相同的入参防御：非法坐标会把 S2 cell 算成 0 并原样落库，
	// 该瞬间从所有地理查询/地图中消失且污染数据；privacy_level 限定枚举。
	if req.Location != nil && !geo.IsValidPoint(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}) {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误: 经纬度超出有效范围")
		return
	}
	if req.PrivacyLevel != nil && !isValidPrivacyLevel(*req.PrivacyLevel) {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误: privacy_level 仅允许 public/private")
		return
	}
	item, err := h.service.Update(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeNotFound, "瞬间不存在")
			return
		}
		if errors.Is(err, ErrDescriptionTooLong) {
			httpx.WriteInvalidRequest(w, r, "Update", "内容超过长度限制")
			return
		}
		if errors.Is(err, ErrTooManyMentions) {
			httpx.WriteInvalidRequest(w, r, "Update", "mentions 数量超过上限")
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			httpx.WriteAPIError(w, r, "Update", httpx.Forbidden(httpx.ReasonNotResourceOwner, "非瞬间作者，无权编辑"))
			return
		}
		if errors.Is(err, ErrTooManyImages) {
			httpx.WriteInvalidRequest(w, r, "Update", "图片数量超过上限")
			return
		}
		if errors.Is(err, ErrImagesNotOwned) {
			// image_file_ids 中媒体资源不存在或不属于当前用户 → 10300
			//（spec api_spec_moment.md moments/update 错误码）。
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeNotFound, "图片不存在或不属于当前用户")
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
			httpx.WriteAPIError(w, r, "Delete", httpx.Forbidden(httpx.ReasonNotResourceOwner, "权限不足"))
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
			httpx.WriteBusinessError(w, r, method, httpx.CodeNotFound, "瞬间不存在")
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

// FetchFollowing 关注动态（spec moment.md → moments/fetch_following，
// 原 feed/following 迁移）：我关注的人发布的公开瞬间，keyset 强一致分页。
func (h *Handler) FetchFollowing(w http.ResponseWriter, r *http.Request) {
	var req FetchFollowingMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "FetchFollowing", "参数错误")
		return
	}
	result, err := h.service.FetchFollowing(r.Context(), req)
	if err != nil {
		h.writeFetchError(w, r, "FetchFollowing", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

// FetchNearby 附近瞬间（spec moment.md → moments/fetch_nearby，
// 原 feed/nearby_moments 迁移）：distance 主序；双版本（匿名版无查看者字段）。
func (h *Handler) FetchNearby(w http.ResponseWriter, r *http.Request) {
	var req FetchNearbyMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "FetchNearby", "参数错误")
		return
	}
	result, err := h.service.FetchNearby(r.Context(), req)
	if err != nil {
		h.writeFetchError(w, r, "FetchNearby", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

// writeFetchError 统一映射发现流错误：非法请求/游标 → 10000，其余按服务端错误。
func (h *Handler) writeFetchError(w http.ResponseWriter, r *http.Request, method string, err error) {
	if errors.Is(err, ErrInvalidFetchRequest) || errors.Is(err, ErrInvalidCursor) {
		httpx.WriteInvalidRequest(w, r, method, "参数错误")
		return
	}
	httpx.WriteAPIError(w, r, method, err)
}

// isValidPrivacyLevel 校验 privacy_level 枚举；空串合法（缺省按 public 处理）。
func isValidPrivacyLevel(level string) bool {
	return level == "" || level == "public" || level == "private"
}
