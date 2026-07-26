package users

import (
	"net/http"
	"strings"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchUsersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.Keyword == "" && req.Tag == "" && req.Radius == nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	// reference_point 可选:缺失时由 service 按回退链解析(见 api_spec_types.md)
	list, err := h.service.Search(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "Search", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req UserOpenTheDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, ok, err := h.service.OpenTheDoor(r.Context(), req.UserID)
	if err != nil {
		h.writeServiceError(w, r, "OpenTheDoor", err)
		return
	}
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "用户不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	var req UserProfileBundleRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	bundle, ok, err := h.service.GetProfileBundle(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "GetProfile", err)
		return
	}
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "用户不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bundle))
}

// BatchGet handles POST /api/v1/users/batch_get. Returns a user_id -> UserItem
// map; non-existent ids are omitted. Capped at 100 ids per spec.
func (h *Handler) BatchGet(w http.ResponseWriter, r *http.Request) {
	var req BatchGetUsersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || len(req.UserIDs) == 0 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if len(req.UserIDs) > 100 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	users, err := h.service.BatchGet(r.Context(), req.UserIDs)
	if err != nil {
		h.writeServiceError(w, r, "BatchGet", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(map[string]any{"users": users}))
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var req UpdateLocationRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Location.LocationName == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UpdateLocation(r.Context(), req); err != nil {
		h.writeServiceError(w, r, "UpdateLocation", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RequestEmailChangeRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.NewEmail == "" || req.Password == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "users/email/request_change", func() httpx.ApiResponse[VerificationSent] {
		result, err := h.service.RequestEmailChange(r.Context(), req)
		if err != nil {
			return errorResponse[VerificationSent](err)
		}
		return httpx.Success(result)
	})
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ConfirmEmailChangeRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.NewEmail == "" || req.VerificationCode == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "users/email/confirm_change", func() httpx.ApiResponse[struct{}] {
		if err := h.service.ConfirmEmailChange(r.Context(), req); err != nil {
			return errorResponse[struct{}](err)
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) RequestPhoneChange(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RequestPhoneChangeRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.NewPhone == "" || req.Password == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "users/phone/request_change", func() httpx.ApiResponse[VerificationSent] {
		result, err := h.service.RequestPhoneChange(r.Context(), req)
		if err != nil {
			return errorResponse[VerificationSent](err)
		}
		return httpx.Success(result)
	})
}

func (h *Handler) ConfirmPhoneChange(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ConfirmPhoneChangeRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.NewPhone == "" || req.VerificationCode == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "users/phone/confirm_change", func() httpx.ApiResponse[struct{}] {
		if err := h.service.ConfirmPhoneChange(r.Context(), req); err != nil {
			return errorResponse[struct{}](err)
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) {
	var req UpgradeAccountRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.LoginID == "" || req.Password == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if !isValidLoginID(req.LoginID) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Upgrade(r.Context(), req); err != nil {
		h.writeServiceError(w, r, "Upgrade", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func isValidLoginID(loginID string) bool {
	if strings.Contains(loginID, "@") {
		parts := strings.SplitN(loginID, "@", 2)
		return len(parts[0]) > 0 && len(parts[1]) > 0 && strings.Contains(parts[1], ".")
	}
	for _, c := range loginID {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(loginID) >= 5
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req DeleteAccountRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.Confirmation != "DELETE_MY_ACCOUNT" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.DeleteAccount(r.Context(), req); err != nil {
		h.writeServiceError(w, r, "DeleteAccount", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

// Follow is the unified two-state follow endpoint (api_spec_user.md). active=true
// follows the target user; active=false unfollows them. Idempotent: a no-op when
// active matches the current state.
func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) {
	var req FollowRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.Active {
		if err := h.service.Follow(r.Context(), req.UserID, req.Source, req.SourceMomentID); err != nil {
			h.writeServiceError(w, r, "Follow", err)
			return
		}
	} else {
		if err := h.service.Unfollow(r.Context(), req.UserID); err != nil {
			h.writeServiceError(w, r, "Unfollow", err)
			return
		}
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

// Block is the unified two-state block endpoint (api_spec_user.md). active=true
// blocks the target user; active=false unblocks them. Idempotent.
func (h *Handler) Block(w http.ResponseWriter, r *http.Request) {
	var req BlockRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.Active {
		if err := h.service.Block(r.Context(), req.UserID, req.Reason); err != nil {
			h.writeServiceError(w, r, "Block", err)
			return
		}
	} else {
		if err := h.service.Unblock(r.Context(), req.UserID); err != nil {
			h.writeServiceError(w, r, "Unblock", err)
			return
		}
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
func (h *Handler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	var req MyBlockedRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	blocked, err := h.service.ListBlocked(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListBlocked", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(blocked))
}
func (h *Handler) SetRemark(w http.ResponseWriter, r *http.Request) {
	var req SetRemarkRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.SetRemark(r.Context(), req.UserID, req.Remark); err != nil {
		h.writeServiceError(w, r, "SetRemark", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) ListReactedMoments(w http.ResponseWriter, r *http.Request) {
	var req UserReactedMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.ListReactedMoments(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListReactedMoments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	var req ListFriendsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	result, err := h.service.ListFriends(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListFriends", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) ListFollowing(w http.ResponseWriter, r *http.Request) {
	var req ListFollowingRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	result, err := h.service.ListFollowing(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListFollowing", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) ListFollowers(w http.ResponseWriter, r *http.Request) {
	var req ListFollowersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	result, err := h.service.ListFollowers(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListFollowers", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	var req UserGroupsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	result, err := h.service.ListGroups(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListGroups", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) ListMoments(w http.ResponseWriter, r *http.Request) {
	var req UserMomentsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	result, err := h.service.ListMoments(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListMoments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
