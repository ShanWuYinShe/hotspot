package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend-go/internal/api"
	apierr "backend-go/internal/errors"
	"backend-go/internal/service"
	"backend-go/internal/types"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

func RegisterUserRoutes(r chi.Router, h *UserHandler) {
	r.Route("/api/v1/users", func(r chi.Router) {
		// 本 Task 仅实现 users/me/* 相关端点，其他列表/搜索端点将于后续 Task 中补齐。
		r.Post("/me/profile/get", h.GetMyProfile)
		r.Post("/me/profile/update", h.UpdateMyProfile)
		r.Post("/me/location/update", h.UpdateMyLocation)
		r.Post("/me/upgrade", h.UpgradeAccount)
		r.Post("/me/settings/get", h.GetMySettings)
		r.Post("/me/settings/update", h.UpdateMySettings)
		r.Post("/me/delete", h.DeleteAccount)
	})
}

func (h *UserHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.GetMyProfile(r.Context())
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, profile)
}

func (h *UserHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	if err := h.service.UpdateMyProfile(r.Context(), req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *UserHandler) UpdateMyLocation(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	if err := h.service.UpdateMyLocation(r.Context(), req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *UserHandler) UpgradeAccount(w http.ResponseWriter, r *http.Request) {
	var req types.UpgradeAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	if err := h.service.UpgradeAccount(r.Context(), req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *UserHandler) GetMySettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetMySettings(r.Context())
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, settings)
}

func (h *UserHandler) UpdateMySettings(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	if err := h.service.UpdateMySettings(r.Context(), req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req types.DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	if err := h.service.DeleteAccount(r.Context(), req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}
