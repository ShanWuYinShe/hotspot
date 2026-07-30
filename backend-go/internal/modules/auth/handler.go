package auth

import (
	"net/http"
	"strings"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/requestmeta"
)

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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.Nickname == "" || req.Password == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	deviceID := resolveDeviceID(r, req.DeviceID)
	if deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.ClientType == ClientTypeWeb {
		idempotency.ReplayOrStore(w, r, "auth/register", func() httpx.ApiResponse[AuthTokenPairWeb] {
			result, err := h.service.Register(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
			if err != nil {
				return errorResponse[AuthTokenPairWeb](apiErrorFrom(err))
			}
			platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour)
			return httpx.Success(AuthTokenPairWeb{})
		})
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/register", func() httpx.ApiResponse[AuthTokenPairApp] {
		result, err := h.service.Register(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
		if err != nil {
			return errorResponse[AuthTokenPairApp](apiErrorFrom(err))
		}
		return authSuccessApp(result)
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.Password == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	deviceID := resolveDeviceID(r, req.DeviceID)
	if deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.Login(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
	if err != nil {
		apiErr := apiErrorFrom(err)
		if apiErr.Code == 10600 {
			logServerError(r, "Login", err)
		} else {
			logBusinessError(r, "Login", apiErr.Code, apiErr.Message)
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(apiErr.Code, apiErr.Message))
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.Nickname == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	deviceID := resolveDeviceID(r, req.DeviceID)
	if deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.ClientType == ClientTypeWeb {
		idempotency.ReplayOrStore(w, r, "auth/guest", func() httpx.ApiResponse[AuthTokenPairWeb] {
			result, err := h.service.Guest(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
			if err != nil {
				return errorResponse[AuthTokenPairWeb](apiErrorFrom(err))
			}
			platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour)
			return httpx.Success(AuthTokenPairWeb{})
		})
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/guest", func() httpx.ApiResponse[AuthTokenPairApp] {
		result, err := h.service.Guest(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
		if err != nil {
			return errorResponse[AuthTokenPairApp](apiErrorFrom(err))
		}
		return authSuccessApp(result)
	})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	deviceID := resolveDeviceID(r, req.DeviceID)
	if req.ClientType == ClientTypeApp && deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	refreshCookie := ""
	if cookie, err := r.Cookie(platformauth.RefreshTokenCookieName); err == nil {
		refreshCookie = cookie.Value
	}
	result, err := h.service.Refresh(r.Context(), req, deviceID, refreshCookie)
	if err != nil {
		logBusinessError(r, "RefreshToken", 10100, "Refresh Token 无效或过期")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10100, "Refresh Token 无效或过期"))
		return
	}
	if req.ClientType == ClientTypeWeb {
		platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour)
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
	var req RequestPasswordResetRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/request_password_reset", func() httpx.ApiResponse[struct{}] {
		if err := h.service.RequestPasswordReset(r.Context(), req); err != nil {
			return errorResponse[struct{}](apiErrorFrom(err))
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ResetPasswordRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.VerificationCode == "" || req.NewPassword == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if req.ClientType == ClientTypeWeb {
		idempotency.ReplayOrStore(w, r, "auth/reset_password", func() httpx.ApiResponse[AuthTokenPairWeb] {
			result, err := h.service.ResetPassword(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, resolveDeviceID(r, "")))
			if err != nil {
				return errorResponse[AuthTokenPairWeb](apiErrorFrom(err))
			}
			platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour)
			return httpx.Success(AuthTokenPairWeb{})
		})
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/reset_password", func() httpx.ApiResponse[AuthTokenPairApp] {
		result, err := h.service.ResetPassword(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, resolveDeviceID(r, "")))
		if err != nil {
			return errorResponse[AuthTokenPairApp](apiErrorFrom(err))
		}
		return authSuccessApp(result)
	})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ChangePasswordRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "auth/change_password", func() httpx.ApiResponse[struct{}] {
		if err := h.service.ChangePassword(r.Context(), req); err != nil {
			return errorResponse[struct{}](apiErrorFrom(err))
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	list := h.service.ListSessions(r.Context())
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	var req RevokeSessionRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.SessionID == "" {
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
		_ = h.service.auth.RevokeSession(r.Context(), principal.UserID, principal.SessionID, principal.SessionID)
		realtime.DefaultHub().CloseSession(principal.SessionID, sessionRevokedEvent())
	}
	platformauth.ClearAuthCookies(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) writeAuthSuccess(w http.ResponseWriter, clientType ClientType, result platformauth.AuthResult) {
	if clientType == ClientTypeWeb {
		platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour)
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(AuthTokenPairWeb{}))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, authSuccessApp(result))
}

func authSuccessApp(result platformauth.AuthResult) httpx.ApiResponse[AuthTokenPairApp] {
	return httpx.Success(AuthTokenPairApp{
		UserID:                result.Principal.UserID,
		AccessToken:           result.AccessToken,
		RefreshToken:          result.RefreshToken,
		AccessTokenExpiresAt:  result.AccessExpires.UTC().Format(time.RFC3339),
		RefreshTokenExpiresAt: result.RefreshExpires.UTC().Format(time.RFC3339),
	})
}

func errorResponse[T any](apiErr *httpx.APIError) httpx.ApiResponse[T] {
	if apiErr == nil {
		apiErr = &httpx.APIError{Code: 10600, Message: "服务暂时不可用"}
	}
	return httpx.ApiResponse[T]{Code: apiErr.Code, Message: apiErr.Message}
}

func sessionMetaFromRequest(r *http.Request, clientType ClientType, deviceID string) platformauth.SessionMeta {
	meta := platformauth.SessionMeta{
		DeviceID:   deviceID,
		DeviceType: string(clientType),
		IPAddress:  requestmeta.ClientIP(r),
		IPLocation: requestmeta.ClientCountry(r),
	}
	if r == nil {
		return meta
	}
	ua := strings.TrimSpace(r.Header.Get("User-Agent"))
	if len(ua) > 120 {
		ua = ua[:120]
	}
	meta.DeviceName = ua
	return meta
}

func resolveDeviceID(r *http.Request, bodyDeviceID string) string {
	if bodyDeviceID != "" {
		return bodyDeviceID
	}
	if c, err := r.Cookie("device_id"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}
