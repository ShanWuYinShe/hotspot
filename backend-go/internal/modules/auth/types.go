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

// AuthStatus 是 get_status 的响应（spec auth.md → get_status）。
// user_id / expires_at 为恒定 nullable：未通过认证（is_valid=false）或未知过期时为 null，
// 字段恒在（Go 侧 nil 指针序列化为 null），而非缺省。
type AuthStatus struct {
	IsValid   bool    `json:"is_valid"`
	UserID    *string `json:"user_id"`    // 恒定 nullable：is_valid=false 时为 null
	ExpiresAt *string `json:"expires_at"` // 恒定 nullable：is_valid=false 或过期未知时为 null
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
	SessionID    string `json:"session_id"`
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type,omitempty"`
	DeviceName   string `json:"device_name,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	IPLocation   string `json:"ip_location,omitempty"`
	CreatedAt    string `json:"created_at"`
	LastActiveAt string `json:"last_active_at"`
	IsCurrent    bool   `json:"is_current"`
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

type RevokeAllResult struct {
	RevokedCount int `json:"revoked_count"`
}
