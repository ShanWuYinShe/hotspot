package users

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.Search(r.Context(), req)))
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req UserOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, ok := h.service.OpenTheDoor(r.Context(), req.UserID)
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "用户不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Location.LocationName == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	h.service.UpdateLocation(r.Context(), req)
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) {
	var req UpgradeAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.LoginID == "" || req.Password == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	h.service.Upgrade(r.Context(), req)
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Confirmation != "DELETE_MY_ACCOUNT" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	h.service.DeleteAccount(r.Context())
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) {
	h.toggleUser(r, w, func(userID string) { h.service.Follow(r.Context(), userID) })
}
func (h *Handler) Unfollow(w http.ResponseWriter, r *http.Request) {
	h.toggleUser(r, w, func(userID string) { h.service.Unfollow(r.Context(), userID) })
}
func (h *Handler) Block(w http.ResponseWriter, r *http.Request) {
	h.toggleUser(r, w, func(userID string) { h.service.Block(r.Context(), userID) })
}
func (h *Handler) Unblock(w http.ResponseWriter, r *http.Request) {
	h.toggleUser(r, w, func(userID string) { h.service.Unblock(r.Context(), userID) })
}
func (h *Handler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.ListBlocked(r.Context())))
}
func (h *Handler) SetRemark(w http.ResponseWriter, r *http.Request) {
	var req SetRemarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	h.service.SetRemark(r.Context(), req.UserID, req.Remark)
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) toggleUser(r *http.Request, w http.ResponseWriter, fn func(string)) {
	var req ToggleUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	fn(req.UserID)
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
