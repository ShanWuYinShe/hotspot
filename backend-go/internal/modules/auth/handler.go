package auth

import (
	"encoding/json"
	"net/http"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/realtime"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.CheckStatus(r.Context())))
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.Nickname == "" || req.Password == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/register", func() httpx.ApiResponse[AuthTokenPairApp] {
		result, err := h.service.Register(req)
		if err != nil {
			return httpx.ApiResponse[AuthTokenPairApp]{Code: 10400, Message: "资源已存在"}
		}
		if req.ClientType == ClientTypeWeb {
			platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour, result.CSRFToken)
			return httpx.ApiResponse[AuthTokenPairApp]{Code: 0}
		}
		return httpx.Success(AuthTokenPairApp{UserID: result.Principal.UserID, AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, AccessTokenExpiresAt: result.AccessExpires.UTC().Format(time.RFC3339), RefreshTokenExpiresAt: result.RefreshExpires.UTC().Format(time.RFC3339)})
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.Password == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.Login(req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10100, "密码错误"))
		return
	}
	h.writeAuthSuccess(w, req.ClientType, result)
}

func (h *Handler) Guest(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req GuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.Nickname == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/guest", func() httpx.ApiResponse[AuthTokenPairApp] {
		result, err := h.service.Guest(req)
		if err != nil {
			return httpx.ApiResponse[AuthTokenPairApp]{Code: 10600, Message: "服务暂时不可用"}
		}
		if req.ClientType == ClientTypeWeb {
			platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour, result.CSRFToken)
			return httpx.ApiResponse[AuthTokenPairApp]{Code: 0}
		}
		return httpx.Success(AuthTokenPairApp{UserID: result.Principal.UserID, AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, AccessTokenExpiresAt: result.AccessExpires.UTC().Format(time.RFC3339), RefreshTokenExpiresAt: result.RefreshExpires.UTC().Format(time.RFC3339)})
	})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.ClientType == ClientTypeWeb && req.RefreshToken != "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.ClientType == ClientTypeApp && req.RefreshToken == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	refreshCookie := ""
	if cookie, err := r.Cookie(platformauth.RefreshTokenCookieName); err == nil {
		refreshCookie = cookie.Value
	}
	result, err := h.service.Refresh(req, refreshCookie)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10100, "Refresh Token 无效或过期"))
		return
	}
	if req.ClientType == ClientTypeWeb {
		platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour, result.CSRFToken)
		payload := AuthTokenPairWeb{}
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
		return
	}
	payload := AuthTokenPairApp{
		UserID:                result.Principal.UserID,
		AccessToken:           result.AccessToken,
		RefreshToken:          result.RefreshToken,
		AccessTokenExpiresAt:  result.AccessExpires.UTC().Format(time.RFC3339),
		RefreshTokenExpiresAt: result.RefreshExpires.UTC().Format(time.RFC3339),
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RequestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.LoginID == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/request_password_reset", func() httpx.ApiResponse[struct{}] {
		h.service.RequestPasswordReset(req)
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.VerificationCode == "" || req.NewPassword == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/reset_password", func() httpx.ApiResponse[AuthTokenPairApp] {
		result, err := h.service.ResetPassword(req)
		if err != nil {
			return httpx.ApiResponse[AuthTokenPairApp]{Code: 10300, Message: "重置请求不存在或已过期"}
		}
		if req.ClientType == ClientTypeWeb {
			platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour, result.CSRFToken)
			return httpx.ApiResponse[AuthTokenPairApp]{Code: 0}
		}
		return httpx.Success(AuthTokenPairApp{UserID: result.Principal.UserID, AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, AccessTokenExpiresAt: result.AccessExpires.UTC().Format(time.RFC3339), RefreshTokenExpiresAt: result.RefreshExpires.UTC().Format(time.RFC3339)})
	})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OldPassword == "" || req.NewPassword == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/me/change_password", func() httpx.ApiResponse[struct{}] {
		if err := h.service.ChangePassword(r.Context(), req); err != nil {
			if apiErr, ok := err.(*httpx.APIError); ok {
				return httpx.ApiResponse[struct{}]{Code: apiErr.Code, Message: apiErr.Message}
			}
			return httpx.ApiResponse[struct{}]{Code: 10100, Message: "密码错误"}
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.ListSessions(r.Context())))
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	var req RevokeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.RevokeSession(r.Context(), req); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(apiErr.Code, apiErr.Message))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.RevokeAll(r.Context())))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	principal, ok := platformauth.PrincipalFromContext(r.Context())
	if ok {
		_ = h.service.auth.RevokeSession(principal.UserID, principal.SessionID, principal.SessionID)
		realtime.DefaultHub().CloseSession(principal.SessionID, realtime.Event{Type: "Error", Payload: realtime.ErrorPayload{Code: 20000, Message: "会话已失效", Severity: "fatal", Retryable: false}})
	}
	platformauth.ClearAuthCookies(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) writeAuthSuccess(w http.ResponseWriter, clientType ClientType, result platformauth.AuthResult) {
	if clientType == ClientTypeWeb {
		platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour, result.CSRFToken)
		payload := AuthTokenPairWeb{}
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
		return
	}
	payload := AuthTokenPairApp{
		UserID:                result.Principal.UserID,
		AccessToken:           result.AccessToken,
		RefreshToken:          result.RefreshToken,
		AccessTokenExpiresAt:  result.AccessExpires.UTC().Format(time.RFC3339),
		RefreshTokenExpiresAt: result.RefreshExpires.UTC().Format(time.RFC3339),
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}
