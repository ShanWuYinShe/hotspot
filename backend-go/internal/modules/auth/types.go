package auth

type ClientType string

const (
	ClientTypeWeb ClientType = "web"
	ClientTypeApp ClientType = "app"
)

type RegisterRequest struct {
	LoginID    string     `json:"login_id"`
	Nickname   string     `json:"nickname"`
	Password   string     `json:"password"`
	ClientType ClientType `json:"client_type"`
	DeviceID   string     `json:"device_id"` // App 端必填；Web 端由服务端从 device_id Cookie 读取（spec auth.md → register）
}

type LoginRequest struct {
	LoginID    string     `json:"login_id"`
	Password   string     `json:"password"`
	ClientType ClientType `json:"client_type"`
	DeviceID   string     `json:"device_id"` // App 端必填；Web 端由服务端从 device_id Cookie 读取（spec auth.md → login）
}

type GuestRequest struct {
	Nickname   string     `json:"nickname"`
	ClientType ClientType `json:"client_type"`
	DeviceID   string     `json:"device_id"` // App 端必填；Web 端由服务端从 device_id Cookie 读取（spec auth.md → guest）
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
