package notify

import (
	"errors"
	"strings"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

const (
	defaultNotifyListLimit = 20
	maxNotifyListLimit     = types.MaxPageSize
)

var ErrInvalidNotifyRequest = errors.New("notifys: invalid request")

type NotifyCategory = string

const (
	NotifyCategorySocial      NotifyCategory = "social"
	NotifyCategoryInteraction NotifyCategory = "interaction"
	NotifyCategoryGroup       NotifyCategory = "group"
	NotifyCategorySystem      NotifyCategory = "system"
	NotifyCategoryNearby      NotifyCategory = "nearby"
)

// NotifyItem is the notify/list item. The primary key is exposed as notify_id
// (spec api_spec_notify.md): it is shared with the realtime WS event payload of
// the same business event, so clients can deduplicate across the two paths.
type NotifyItem struct {
	NotifyID  string         `json:"notify_id"`
	Type      string         `json:"type"`
	IsRead    bool           `json:"is_read"`
	CreatedAt string         `json:"created_at"`
	Payload   map[string]any `json:"payload"`
}

type PaginatedNotify struct {
	Items      []NotifyItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type UnreadCountByType struct {
	Social      int `json:"social"`
	Interaction int `json:"interaction"`
	Group       int `json:"group"`
	Report      int `json:"report"`
	Nearby      int `json:"nearby"`
	System      int `json:"system"`
}

type UnreadCountResult struct {
	Total  int               `json:"total"`
	ByType UnreadCountByType `json:"by_type"`
}

type PaginationParams = types.PaginationParams

// ListNotifyRequest 是 notify/list 的请求类型（spec api_spec_notify.md）。
type ListNotifyRequest struct {
	Type       string            `json:"type,omitempty"`
	IsRead     *bool             `json:"is_read,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// NotifyListRequest 是内部沿用类型（与 ListNotifyRequest 同构）。
type NotifyListRequest = ListNotifyRequest

// MarkReadScope enumera the two mark-read scopes (spec api_spec_notify.md).
// scope is mandatory and must be explicit — there is no default-to-all semantics.
type MarkReadScope string

const (
	MarkReadScopeIDs MarkReadScope = "ids"
	MarkReadScopeAll MarkReadScope = "all"
)

// MarkReadRequest marks notifys as read (spec api_spec_notify.md):
// scope='ids' requires notify_ids (1-50); scope='all' marks everything read and
// ignores notify_ids. Unknown or foreign ids are silently ignored by the repo.
type MarkReadRequest struct {
	Scope     string   `json:"scope"`
	NotifyIDs []string `json:"notify_ids,omitempty"`
}

const (
	maxMarkReadIDs = 50
)

type notifyListQuery struct {
	Type   string
	IsRead *bool
	Limit  int
	Cursor *notifyCursor
}

type notifyCursor struct {
	CreatedAt string `json:"created_at"`
	NotifyID  string `json:"notify_id"`
}

func normalizeNotifyType(value string) string {
	value = strings.TrimSpace(value)
	// 与 spec/api_spec_types.md NotifyType union 保持一致（15 种）。
	switch value {
	case "", "new_follower", "mutual_follow", "moment_commented", "comment_reply", "mentioned", "group_mentioned", "reaction_added", "group_invite_received", "group_application_received", "group_application_accepted", "group_application_rejected", "report_resolved", "report_dismissed", "system_announcement", "nearby_new_moment":
		return value
	default:
		return ""
	}
}

func normalizeNotifyListQuery(req NotifyListRequest) (notifyListQuery, error) {
	normalizedType := normalizeNotifyType(req.Type)
	if strings.TrimSpace(req.Type) != "" && normalizedType == "" {
		return notifyListQuery{}, ErrInvalidNotifyRequest
	}
	limit := defaultNotifyListLimit
	cursorValue := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > maxNotifyListLimit {
			limit = maxNotifyListLimit
		}
		cursorValue = strings.TrimSpace(req.Pagination.Cursor)
	}
	cursor, err := decodeNotifyCursor(cursorValue)
	if err != nil {
		return notifyListQuery{}, err
	}
	return notifyListQuery{Type: normalizedType, IsRead: req.IsRead, Limit: limit, Cursor: cursor}, nil
}

// notifyCursorNamespace 是通知列表分页游标的加密命名空间。
const notifyCursorNamespace = "notify.list"

func encodeNotifyCursor(c notifyCursor) string {
	value, err := cursor.Encode(notifyCursorNamespace, c)
	if err != nil {
		return ""
	}
	return value
}

func decodeNotifyCursor(value string) (*notifyCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var c notifyCursor
	if err := cursor.Decode(notifyCursorNamespace, value, &c); err != nil {
		return nil, ErrInvalidNotifyRequest
	}
	if c.CreatedAt == "" || c.NotifyID == "" {
		return nil, ErrInvalidNotifyRequest
	}
	return &c, nil
}
