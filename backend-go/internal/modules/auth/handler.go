package auth

import (
	"errors"
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.Nickname == "" || req.Password == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	deviceID := resolveDeviceID(r, req.DeviceID)
	if deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.Password == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	deviceID := resolveDeviceID(r, req.DeviceID)
	if deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	result, err := h.service.Login(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
	if err != nil {
		apiErr := apiErrorFrom(err)
		if apiErr.Code == httpx.CodeInternalUnavailable {
			observability.LogServerError(r, "Login", err)
		} else {
			observability.LogBusinessError(r, "Login", int(apiErr.Code), apiErr.Message)
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.Nickname == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	deviceID := resolveDeviceID(r, req.DeviceID)
	if deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if req.ClientType == ClientTypeWeb && req.RefreshToken != "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if req.ClientType == ClientTypeApp && req.RefreshToken == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	deviceID := resolveDeviceID(r, req.DeviceID)
	if req.ClientType == ClientTypeApp && deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	refreshCookie := ""
	if cookie, err := r.Cookie(platformauth.RefreshTokenCookieName); err == nil {
		refreshCookie = cookie.Value
	}
	result, err := h.service.Refresh(r.Context(), req, deviceID, refreshCookie)
	if err != nil {
		httpx.WriteBusinessError(w, r, "RefreshToken", httpx.CodeUnauthorized, "Refresh Token 无效或过期")
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 幂等语义（spec api_spec_auth.md）：本接口靠 IP 限流保护，无需
	// Idempotency-Key。重复请求重新触发 service（防账号枚举 + 限流），不缓存。
	if err := h.service.RequestPasswordReset(r.Context(), req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, errorResponse[struct{}](apiErrorFrom(err)))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || req.VerificationCode == "" || req.NewPassword == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 自然去重（spec api_spec_auth.md）：验证码为一次性凭证，首次成功后即失效，
	// 重发返回 httpx.CodeNotFound（验证码已用/过期），不会重复改密。无需 Idempotency-Key。
	if req.ClientType == ClientTypeWeb {
		result, err := h.service.ResetPassword(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, resolveDeviceID(r, "")))
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, errorResponse[AuthTokenPairWeb](apiErrorFrom(err)))
			return
		}
		platformauth.SetAuthCookies(w, result.AccessToken, 15*time.Minute, result.RefreshToken, 30*24*time.Hour)
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(AuthTokenPairWeb{}))
		return
	}
	result, err := h.service.ResetPassword(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, resolveDeviceID(r, "")))
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, errorResponse[AuthTokenPairApp](apiErrorFrom(err)))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, authSuccessApp(result))
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 自然去重（spec api_spec_auth.md）：首次成功后旧密码立即失效，
	// 重发同一请求（仍带旧密码）返回 httpx.CodeUnauthorized，不会重复改密。无需 Idempotency-Key。
	if err := h.service.ChangePassword(r.Context(), req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, errorResponse[struct{}](apiErrorFrom(err)))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
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
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.SessionID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.RevokeSession(r.Context(), req); err != nil {
		// service 以 %w 包装底层错误，必须用 errors.As 解包 APIError
		//（type assertion 会因包装而失效，导致业务错误码退化为 httpx.CodeInternalUnavailable）。
		var apiErr *httpx.APIError
		if errors.As(err, &apiErr) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(apiErr.Code, apiErr.Message))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
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
		// Best-effort revocation: even if it fails the tokens will expire, but a
		// persistent failure (e.g. DB down) means revoked tokens stay valid until
		// then — log it so operators can detect a degraded revocation path.
		if err := h.service.auth.RevokeSession(r.Context(), principal.UserID, principal.SessionID, principal.SessionID); err != nil {
			observability.L().Warn("logout: session revoke failed", "user_id", principal.UserID, "session_id", principal.SessionID, "error", err)
		}
		realtime.DefaultHub().CloseSession(principal.SessionID, sessionRevokedEvent("revoked_by_user"))
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
		apiErr = &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
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
