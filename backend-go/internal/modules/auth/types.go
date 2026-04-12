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
}

type LoginRequest struct {
	LoginID    string     `json:"login_id"`
	Password   string     `json:"password"`
	ClientType ClientType `json:"client_type"`
}

type GuestRequest struct {
	Nickname   string     `json:"nickname"`
	ClientType ClientType `json:"client_type"`
}

type RefreshTokenRequest struct {
	ClientType   ClientType `json:"client_type"`
	RefreshToken string     `json:"refresh_token,omitempty"`
}

type AuthStatus struct {
	IsValid   bool   `json:"is_valid"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type AuthTokenPairWeb struct{}

type AuthTokenPairApp struct {
	UserID                string `json:"user_id"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
}

type SessionItem struct {
	SessionID    string `json:"session_id"`
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
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type RevokeAllResult struct {
	RevokedCount int `json:"revoked_count"`
}
