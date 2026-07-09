package types

// UserInfo is the shared user-profile shape used across all modules. Modules
// alias this instead of redefining it, so a field rename or addition doesn't
// drift across 7+ copies.
type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}
