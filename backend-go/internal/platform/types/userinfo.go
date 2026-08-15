package types

// UserInfo is the shared user-profile shape used across all modules. Modules
// alias this instead of redefining it, so a field rename or addition doesn't
// drift across 7+ copies.
type UserInfo struct {
	UserID    string  `json:"user_id"`
	Nickname  string  `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
}

// StrPtr returns a pointer to s, or nil when s is empty. It is the standard
// way to build spec-compliant `string | null` fields from internal string data.
func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
