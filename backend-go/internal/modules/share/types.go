package share

type CreateShareRequest struct {
	ShareType        string         `json:"share_type"`
	Payload          map[string]any `json:"payload"`
	ExpiresInSeconds int            `json:"expires_in_seconds,omitempty"`
}

type ShareCreated struct {
	ShareID   string `json:"share_id"`
	ShareCode string `json:"share_code"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type GetShareRequest struct {
	ShareCode string `json:"share_code"`
}

type ShareDetail struct {
	Share   map[string]any `json:"share"`
	Payload map[string]any `json:"payload"`
	Preview map[string]any `json:"preview,omitempty"`
}

type ShareListItem struct {
	ShareID   string         `json:"share_id"`
	ShareCode string         `json:"share_code"`
	ShareType string         `json:"share_type"`
	CreatedAt string         `json:"created_at"`
	ExpiresAt string         `json:"expires_at,omitempty"`
	IsRevoked bool           `json:"is_revoked"`
	Preview   map[string]any `json:"preview,omitempty"`
}

type PaginatedShares struct {
	Items []ShareListItem `json:"items"`
}

type RevokeShareRequest struct {
	ShareID string `json:"share_id"`
}
