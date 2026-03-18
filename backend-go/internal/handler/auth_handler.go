package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"backend-go/internal/api"
	apierr "backend-go/internal/errors"
	"backend-go/internal/service"
	"backend-go/internal/types"
)

type AuthHandler struct {
	service service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

func RegisterAuthRoutes(r chi.Router, h *AuthHandler) {
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/guest", h.Guest)
		r.Post("/refresh_token", h.RefreshToken)
		r.Post("/check_status", h.CheckStatus)
		r.Post("/request_password_reset", h.RequestPasswordReset)
		r.Post("/reset_password", h.ResetPassword)
		r.Post("/me/change_password", h.ChangePassword)
		r.Post("/me/sessions/list", h.ListSessions)
		r.Post("/me/sessions/revoke", h.RevokeSession)
		r.Post("/me/sessions/revoke_all", h.RevokeAllSessions)
	})
}

// helper 将内部 AuthTokens 按 client_type 转换为对外类型。
func writeAuthTokens(w http.ResponseWriter, r *http.Request, clientType types.ClientType, tokens *service.AuthTokens) {
	if clientType == types.ClientTypeWeb {
		// Web：token 通过 Cookie 下发，body 为空对象
		// 具体 Cookie 属性（Secure/SameSite/Path）由上层部署根据 spec/api_spec_auth.md 配置
		http.SetCookie(w, &http.Cookie{Name: "access_token", Value: tokens.AccessToken, Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: tokens.RefreshToken, Path: "/api/v1/auth/refresh_token"})
		api.WriteSuccess(w, r, types.AuthTokenPairWeb{})
		return
	}
	// App：token 在 body 中返回
	resp := types.AuthTokenPairApp{
		UserID:               tokens.UserID,
		AccessToken:          tokens.AccessToken,
		RefreshToken:         tokens.RefreshToken,
		AccessTokenExpiresAt: tokens.AccessTokenExpiresAt.UTC().Format(time.RFC3339Nano),
		RefreshTokenExpiresAt: tokens.RefreshTokenExpiresAt.UTC().Format(time.RFC3339Nano),
	}
	api.WriteSuccess(w, r, resp)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	tokens, err := h.service.Register(r.Context(), req)
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	writeAuthTokens(w, r, req.ClientType, tokens)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req types.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	tokens, err := h.service.Login(r.Context(), req)
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	writeAuthTokens(w, r, req.ClientType, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// TODO: 解析当前用户与会话信息，标记黑名单并撤销 session
	if err := h.service.Logout(r.Context()); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *AuthHandler) Guest(w http.ResponseWriter, r *http.Request) {
	var req types.GuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	tokens, err := h.service.Guest(r.Context(), req)
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	writeAuthTokens(w, r, req.ClientType, tokens)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req types.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}

	var tokenStr string
	if req.ClientType == types.ClientTypeWeb {
		// Web：不能在 body 中携带 refresh_token，必须从 Cookie 中获取
		if req.RefreshToken != nil && *req.RefreshToken != "" {
			e := apierr.ErrInvalidRequest("refresh_token_must_be_empty_for_web")
			api.WriteError(w, r, e.Code, e.Message)
			return
		}
		cookie, err := r.Cookie("refresh_token")
		if err != nil || cookie.Value == "" {
			api.WriteError(w, r, 10100, "auth_failed")
			return
		}
		tokenStr = cookie.Value
	}

	tokens, err := h.service.RefreshToken(r.Context(), req, tokenStr)
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	writeAuthTokens(w, r, req.ClientType, tokens)
}

func (h *AuthHandler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	// 尝试从 Header 和 Cookie 中读取 access_token
	var tokenStr string
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		tokenStr = strings.TrimSpace(authHeader[7:])
	} else if cookie, err := r.Cookie("access_token"); err == nil {
		tokenStr = cookie.Value
	}

	status, err := h.service.CheckStatus(r.Context(), tokenStr)
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, status)
}

func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req types.RequestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	if err := h.service.RequestPasswordReset(r.Context(), req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req types.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	tokens, err := h.service.ResetPassword(r.Context(), req)
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	writeAuthTokens(w, r, req.ClientType, tokens)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req types.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	// TODO: 从上下文获取 userID
	if err := h.service.ChangePassword(r.Context(), "", req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	// TODO: 从上下文获取 userID
	items, err := h.service.ListSessions(r.Context(), "")
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{
		Items []types.SessionItem `json:"items"`
	}{Items: items})
}

func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	var req types.RevokeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		e := apierr.ErrInvalidRequest("invalid_request_body")
		api.WriteError(w, r, e.Code, e.Message)
		return
	}
	// TODO: 从上下文获取 userID
	if err := h.service.RevokeSession(r.Context(), "", req); err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, struct{}{})
}

func (h *AuthHandler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	// TODO: 从上下文获取 userID
	res, err := h.service.RevokeAllSessions(r.Context(), "")
	if err != nil {
		if ae, ok := err.(*apierr.APIError); ok {
			api.WriteError(w, r, ae.Code, ae.Message)
			return
		}
		api.WriteError(w, r, apierr.ErrInternal("").Code, apierr.ErrInternal("").Message)
		return
	}
	api.WriteSuccess(w, r, res)
}
