package share

import (
	"errors"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

const (
	ShareTypeUser     = "user"
	ShareTypeGroup    = "group"
	ShareTypeMoment   = "moment"
	ShareTypeLocation = "location"

	defaultShareListLimit = 20
	maxShareListLimit     = types.MaxPageSize
	maxShareExpiresInSec  = 30 * 24 * 60 * 60
	shareCodeAlphabet     = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	shareCodeLength       = 12
)

var (
	ErrInvalidShareRequest = errors.New("share: invalid request")
	ErrShareNotFound       = errors.New("share: not found")
	ErrSharePermission     = errors.New("share: permission denied")
)

type CreateShareRequest struct {
	ShareType        string         `json:"share_type"`
	Payload          map[string]any `json:"payload"`
	ExpiresInSeconds int            `json:"expires_in_seconds,omitempty"`
}

type ShareCreated struct {
	ShareID   string  `json:"share_id"`
	ShareCode string  `json:"share_code"`
	ExpiresAt *string `json:"expires_at"`
}

type GetShareRequest struct {
	ShareCode string `json:"share_code"`
}

type ShareUserInfo struct {
	UserID    string  `json:"user_id"`
	Nickname  string  `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
}

type SharePreview struct {
	Title    string  `json:"title"`
	Subtitle *string `json:"subtitle"`
	CoverURL *string `json:"cover_url"`
}

type ShareBase struct {
	ShareID   string        `json:"share_id"`
	ShareCode string        `json:"share_code"`
	ShareType string        `json:"share_type"`
	Creator   ShareUserInfo `json:"creator"`
	CreatedAt string        `json:"created_at"`
	ExpiresAt *string       `json:"expires_at"`
}

type ShareDetail struct {
	Share   ShareBase      `json:"share"`
	Payload map[string]any `json:"payload"`
	Preview *SharePreview  `json:"preview"`
}

type PaginationParams = types.PaginationParams

type ListSharesRequest struct {
	ShareType  string            `json:"share_type,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type ShareListItem struct {
	ShareID   string        `json:"share_id"`
	ShareCode string        `json:"share_code"`
	ShareType string        `json:"share_type"`
	CreatedAt string        `json:"created_at"`
	ExpiresAt *string       `json:"expires_at"`
	IsRevoked bool          `json:"is_revoked"`
	Preview   *SharePreview `json:"preview"`
}

type PaginatedShares struct {
	Items      []ShareListItem `json:"items"`
	NextCursor *string         `json:"next_cursor"`
}

// RevokeShareRequest revokes a share (spec api_spec_share.md): exactly one of
// ShareID or ShareCode must be provided — the creator may only have kept the
// copied link code, so revoke must be addressable by code as well.
type RevokeShareRequest struct {
	ShareID   string `json:"share_id,omitempty"`
	ShareCode string `json:"share_code,omitempty"`
}

type createShareCommand struct {
	CreatorUserID string
	ShareType     string
	Payload       map[string]any
	ExpiresAt     *time.Time
}

type getShareQuery struct {
	ShareCode string
}

type listSharesQuery struct {
	CreatorUserID string
	ShareType     string
	Limit         int
	Cursor        string
}

type revokeShareCommand struct {
	CreatorUserID string
	ShareID       string
}

type shareCursor struct {
	CreatedAt string `json:"created_at"`
	ShareID   string `json:"share_id"`
}

type shareRecord struct {
	ShareID       string
	ShareCode     string
	ShareType     string
	CreatorUserID string
	Payload       map[string]any
	CreatedAt     string
	ExpiresAt     string
	IsRevoked     bool
}

func normalizeShareType(value string) string {
	switch strings.TrimSpace(value) {
	case ShareTypeUser, ShareTypeGroup, ShareTypeMoment, ShareTypeLocation:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeShareCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func validShareCode(value string) bool {
	if len(value) != shareCodeLength {
		return false
	}
	for _, r := range value {
		if !strings.ContainsRune(shareCodeAlphabet, r) {
			return false
		}
	}
	return true
}

func normalizeSharePagination(p *PaginationParams) (int, string) {
	if p == nil {
		return defaultShareListLimit, ""
	}
	limit := p.Limit
	if limit <= 0 {
		limit = defaultShareListLimit
	}
	if limit > maxShareListLimit {
		limit = maxShareListLimit
	}
	return limit, strings.TrimSpace(p.Cursor)
}

// shareCursorCodec 是分页游标的统一加密封装（platform/cursor，AES-GCM +
// 命名空间）。审查 P2-12：此前本模块是全仓唯一用裸 base64(JSON) 的游标——
// 客户端可解码甚至手工伪造 (created_at, share_id) 游标，违反 AGENTS 分页
// 纪律「cursor 不透明」。命名空间隔离保证与其它模块游标不可互换。
var shareCursorCodec = cursor.NewCodec("share")

func encodeShareCursor(cursor shareCursor) (string, error) {
	return shareCursorCodec.Encode(cursor)
}

func decodeShareCursor(raw string) (*shareCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var parsed shareCursor
	if err := shareCursorCodec.Decode(strings.TrimSpace(raw), &parsed); err != nil {
		return nil, ErrInvalidShareRequest
	}
	if parsed.CreatedAt == "" || parsed.ShareID == "" {
		return nil, ErrInvalidShareRequest
	}
	return &parsed, nil
}
