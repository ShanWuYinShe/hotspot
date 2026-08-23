package auth

type ClientType string

const (
	ClientTypeWeb ClientType = "web"
	ClientTypeApp ClientType = "app"
)

type RegisterRequest struct {
	LoginID string `json:"login_id"`
	// VerificationCode 注册验证码（spec api_spec_public.md → register）：
	// login_id 为手机号/邮箱型时必填（所有权验证，经 send_register_code 发送），
	// 用户名型必须为 null（同传视为参数冲突）。
	VerificationCode *string    `json:"verification_code,omitempty"`
	Nickname         string     `json:"nickname"`
	Password         string     `json:"password"`
	ClientType       ClientType `json:"client_type"`
	DeviceID         string     `json:"device_id"` // App 端必填；Web 端由服务端从 device_id Cookie 读取（spec auth.md → register）
}

type LoginRequest struct {
	LoginID    string     `json:"login_id"`
	Password   string     `json:"password"`
	ClientType ClientType `json:"client_type"`
	DeviceID   string     `json:"device_id"` // App 端必填；Web 端由服务端从 device_id Cookie 读取（spec auth.md → login）
}

// GuestRequest：Nickname 可选（spec api_spec_public.md → guest），null 时服务端
// 生成默认昵称「游客·xxxx」（进入后引导修改，降低访客模式进入摩擦）。
type GuestRequest struct {
	Nickname   *string    `json:"nickname"`
	ClientType ClientType `json:"client_type"`
	DeviceID   string     `json:"device_id"` // App 端必填；Web 端由服务端从 device_id Cookie 读取（spec auth.md → guest）
}

// SendRegisterCodeRequest 发送注册验证码（spec api_spec_public.md →
// auth/send_register_code）：仅手机号/邮箱型 login_id 支持。
type SendRegisterCodeRequest struct {
	LoginID    string     `json:"login_id"`
	ClientType ClientType `json:"client_type"`
}

type RefreshTokenRequest struct {
	ClientType   ClientType `json:"client_type"`
	DeviceID     string     `json:"device_id"`
	RefreshToken string     `json:"refresh_token,omitempty"`
}

// SessionStatus 是 auth/session 的响应（spec auth.md → auth/session）。
// 该端点挂在 /api/v1 必须认证节点下，中间件已保证请求持有有效 Token；
// 因此响应只有两个必填字段，不再包含历史遗留的 is_valid。
type SessionStatus struct {
	UserID    string `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
}

type AuthTokenPairWeb struct {
	ClientType string `json:"client_type"` // 判别字段（spec auth.md → AuthTokenPair）
}

type AuthTokenPairApp struct {
	ClientType            string `json:"client_type"` // 判别字段
	UserID                string `json:"user_id"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
}

type SessionItem struct {
	SessionID    string  `json:"session_id"`
	DeviceID     string  `json:"device_id"`
	DeviceType   *string `json:"device_type"`
	DeviceName   *string `json:"device_name"`
	IPAddress    *string `json:"ip_address"`
	IPLocation   *string `json:"ip_location"`
	CreatedAt    string  `json:"created_at"`
	LastActiveAt string  `json:"last_active_at"`
	IsCurrent    bool    `json:"is_current"`
}

type SessionList struct {
	Items []SessionItem `json:"items"`
}

type RevokeSessionRequest struct {
	SessionID string `json:"session_id"`
}

type RequestPasswordResetRequest struct {
	LoginID    string     `json:"login_id"`
	ClientType ClientType `json:"client_type"`
}

type ResetPasswordRequest struct {
	LoginID          string     `json:"login_id"`
	VerificationCode string     `json:"verification_code"`
	NewPassword      string     `json:"new_password"`
	ClientType       ClientType `json:"client_type"`
	DeviceID         string     `json:"device_id"` // App 端必填；Web 端由服务端从 device_id Cookie 读取（spec auth.md → reset_password）
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type RevokeOtherSessionsResult struct {
	RevokedCount int `json:"revoked_count"`
}
