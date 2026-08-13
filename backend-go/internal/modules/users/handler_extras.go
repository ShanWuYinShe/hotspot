package users

import (
	"net/http"
	"strings"

	"hotspot/backend-go/internal/platform/httpx"
)

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchUsersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.Keyword == "" && req.Tag == "" && req.Radius == nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	bundle, ok, err := h.service.GetProfileBundle(r.Context(), req.UserID, req.IncludeRecentMoments)
	if err != nil {
		h.writeServiceError(w, r, "OpenTheDoor", err)
		return
	}
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "用户不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bundle))
}

// BatchGet handles POST /api/v1/users/batch_get. Returns a user_id -> UserItem
// map; non-existent ids are omitted. Capped at 100 ids per spec.
func (h *Handler) BatchGet(w http.ResponseWriter, r *http.Request) {
	var req BatchGetUsersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || len(req.UserIDs) == 0 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if len(req.UserIDs) > 100 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.UpdateLocation(r.Context(), req); err != nil {
		h.writeServiceError(w, r, "UpdateLocation", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) {
	var req UpgradeAccountRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.LoginID == "" || req.Password == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if !isValidLoginID(req.LoginID) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// login_id 所有权验证（spec api_spec_user.md → upgrade）：手机号/邮箱需验证码
	// 服务校验所有权，接入前暂仅支持用户名型 login_id（防止访客抢占他人手机号/邮箱）。
	if strings.Contains(req.LoginID, "@") || isNumericLoginID(req.LoginID) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "手机号/邮箱升级需验证码验证，暂仅支持用户名"))
		return
	}
	// 升级成功返回完整账户数据（spec api_spec_user.md users/upgrade），
	// 客户端无需额外调 get_account 即可获取新的 user_type。
	profile, err := h.service.Upgrade(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "Upgrade", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(profile))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 组合约束（spec user.md → set_follow）：source='moment' 时 source_moment_id 必填；
	// 提供 source_moment_id 但 source != 'moment' 视为参数冲突
	if req.Active {
		if req.Source == FollowSourceMoment && req.SourceMomentID == "" {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "source 为 moment 时 source_moment_id 必填"))
			return
		}
		if req.SourceMomentID != "" && req.Source != FollowSourceMoment {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "source_moment_id 仅在 source=moment 时有意义"))
			return
		}
	} else if req.SourceMomentID != "" || req.Source != "" {
		// active=false 时来源字段无意义；宽容处理（忽略），与 spec 一致
		req.Source = ""
		req.SourceMomentID = ""
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	result, err := h.service.ListMoments(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, "ListMoments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
