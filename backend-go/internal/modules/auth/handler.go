package auth

import (
	"errors"
	"net/http"
	"strconv"
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

// GetStatus 查询当前会话状态（spec auth.md → auth/session，必须认证节点）。
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	status, err := h.service.GetStatus(r.Context())
	if err != nil {
		httpx.WriteAPIError(w, r, "GetStatus", apiErrorFrom(err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(status))
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
	if req.LoginID == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// nickname/password 通用字段约束（spec api_spec_base.md「通用字段约束」）：
	// nickname 1-30 字符；password 8-128 字符且至少含字母和数字。
	if !isValidNickname(req.Nickname) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if !isValidPassword(req.Password) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// login_id 格式三分校验（spec api_spec_base.md「通用字段约束」）：
	// 长度 3-64；含 '@' 必须为合法邮箱；纯数字即手机号型；其余为用户名型。
	if !isValidAuthLoginID(req.LoginID) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 验证码规则（spec api_spec_public.md → register）：手机号/邮箱型必填
	//（缺失 10000，错误/过期 10401 由 service 判定）；用户名型必须为 null。
	if isPhoneOrEmailLoginID(req.LoginID) {
		if req.VerificationCode == nil || *req.VerificationCode == "" {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
	} else if req.VerificationCode != nil {
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
			platformauth.SetAuthCookies(w, result.AccessToken, h.service.AccessTTL(), result.RefreshToken, h.service.RefreshTTL())
			return httpx.Success(AuthTokenPairWeb{ClientType: "web"})
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
	// login_id 格式三分校验（与 register 同一规则，spec api_spec_base.md）。
	if !isValidAuthLoginID(req.LoginID) {
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
		writeAuthError(w, r, "Login", err)
		return
	}
	h.writeAuthSuccess(w, req.ClientType, result)
}

// writeAuthError 输出 service 层映射后的业务错误。APIError 携带
// RetryAfterSeconds 时设置 Retry-After 响应头（spec api_spec_errors.md →
// 10500「按 Retry-After 响应头等待后重试」；限流中间件自身触发时由中间件设置）。
func writeAuthError(w http.ResponseWriter, r *http.Request, op string, err error) {
	apiErr := apiErrorFrom(err)
	if apiErr.RetryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(apiErr.RetryAfterSeconds))
	}
	if apiErr.Code == httpx.CodeInternalUnavailable {
		observability.LogServerError(r, op, err)
	} else {
		observability.LogBusinessError(r, op, int(apiErr.Code), apiErr.Message)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(apiErr.Code, apiErr.Message))
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
	// Nickname 可选（spec api_spec_public.md → guest）：null 时由 service 生成
	// 默认昵称「游客·xxxx」；显式传入时须符合 1-30 字符约束
	//（spec api_spec_base.md「通用字段约束」）。
	if req.Nickname != nil && !isValidNickname(strings.TrimSpace(*req.Nickname)) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp {
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
			platformauth.SetAuthCookies(w, result.AccessToken, h.service.AccessTTL(), result.RefreshToken, h.service.RefreshTTL())
			return httpx.Success(AuthTokenPairWeb{ClientType: "web"})
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
		platformauth.SetAuthCookies(w, result.AccessToken, h.service.AccessTTL(), result.RefreshToken, h.service.RefreshTTL())
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

// SendRegisterCode 发送注册验证码（spec api_spec_public.md →
// auth/send_register_code）：仅手机号/邮箱型 login_id 支持；无论 login_id
// 是否已被注册均返回成功（防枚举）。频率由 auth_register_per_10minutes 限流。
func (h *Handler) SendRegisterCode(w http.ResponseWriter, r *http.Request) {
	var req SendRegisterCodeRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if req.LoginID == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 仅手机号型（纯数字）/邮箱型（合法邮箱）支持；用户名型 10000
	//（spec api_spec_public.md：用户名无第三方所有权，无需验证码）。
	if !isValidAuthLoginID(req.LoginID) || !isPhoneOrEmailLoginID(req.LoginID) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.SendRegisterCode(r.Context(), req); err != nil {
		writeAuthError(w, r, "SendRegisterCode", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
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
		writeAuthError(w, r, "RequestPasswordReset", err)
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
	if req.LoginID == "" || req.VerificationCode == "" || (req.ClientType != ClientTypeWeb && req.ClientType != ClientTypeApp) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// password 通用约束（spec api_spec_base.md「通用字段约束」：8-128 字符，
	// 至少含字母和数字），与 register 同一规则。
	if !isValidPassword(req.NewPassword) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// device_id：Web 端从 Cookie 读取，App 端请求体必填非 null
	//（spec api_spec_public.md → reset_password Request）；与 signin/register
	// 一致以空值 10000 拒绝，保证 (user_id, device_id) 单会话模型不被空设备标识破坏。
	deviceID := resolveDeviceID(r, req.DeviceID)
	if deviceID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 自然去重（spec api_spec_auth.md）：验证码为一次性凭证，首次成功后即失效，
	// 重发返回 httpx.CodeNotFound（验证码已用/过期），不会重复改密。无需 Idempotency-Key。
	if req.ClientType == ClientTypeWeb {
		result, err := h.service.ResetPassword(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
		if err != nil {
			writeAuthError(w, r, "ResetPassword", err)
			return
		}
		platformauth.SetAuthCookies(w, result.AccessToken, h.service.AccessTTL(), result.RefreshToken, h.service.RefreshTTL())
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(AuthTokenPairWeb{ClientType: "web"}))
		return
	}
	// App 端：device_id 来自请求体（spec auth.md → reset_password，自动登录会话纳入单设备单会话模型）
	result, err := h.service.ResetPassword(r.Context(), req, sessionMetaFromRequest(r, req.ClientType, deviceID))
	if err != nil {
		writeAuthError(w, r, "ResetPassword", err)
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
	// 新密码须符合通用约束（spec api_spec_base.md：8-128 字符，至少含字母和数字）。
	if req.OldPassword == "" || !isValidPassword(req.NewPassword) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	// 自然去重（spec api_spec_auth.md → change_password）：首次成功后旧密码立即失效，
	// 重发同一请求（仍带旧密码）返回 httpx.CodeInvalidOldCredential（10401 旧密码错误），不会重复改密。无需 Idempotency-Key。
	if err := h.service.ChangePassword(r.Context(), req); err != nil {
		writeAuthError(w, r, "ChangePassword", err)
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
		platformauth.SetAuthCookies(w, result.AccessToken, h.service.AccessTTL(), result.RefreshToken, h.service.RefreshTTL())
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(AuthTokenPairWeb{ClientType: "web"}))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, authSuccessApp(result))
}

func authSuccessApp(result platformauth.AuthResult) httpx.ApiResponse[AuthTokenPairApp] {
	return httpx.Success(AuthTokenPairApp{
		ClientType:            "app",
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
