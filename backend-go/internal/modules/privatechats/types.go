package privatechats

import (
	"time"

	private_messages "hotspot/backend-go/internal/modules/privatemessages"
	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

type PaginationParams = types.PaginationParams

type chatCursor struct {
	LastActivityAt string `json:"last_activity_at"`
	UserID         string `json:"user_id"`
}

func encodeChatCursor(lastActivityAt, userID string) string {
	value, err := platformcursor.Encode("private_chats.list", chatCursor{LastActivityAt: lastActivityAt, UserID: userID})
	if err != nil {
		return ""
	}
	return value
}

func decodeChatCursor(value string) (time.Time, string, error) {
	if value == "" {
		return time.Time{}, "", nil
	}
	var decoded chatCursor
	if err := platformcursor.Decode("private_chats.list", value, &decoded); err != nil || decoded.LastActivityAt == "" || decoded.UserID == "" {
		return time.Time{}, "", private_messages.ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.LastActivityAt)
	if err != nil {
		return time.Time{}, "", private_messages.ErrInvalidCursor
	}
	return parsed.UTC(), decoded.UserID, nil
}

type UserInfo = types.UserInfo

type LastMessage struct {
	MessageID string   `json:"message_id"`
	Content   string   `json:"content,omitempty"`
	Type      string   `json:"type"`
	ShareCode string   `json:"share_code,omitempty"`
	Sender    UserInfo `json:"sender"`
	SentAt    string   `json:"sent_at"`
}

type PrivateChatItem struct {
	OtherUser      UserInfo     `json:"other_user"`
	UnreadCount    int          `json:"unread_count"`
	IsMuted        bool         `json:"is_muted"`
	IsPinned       bool         `json:"is_pinned"`
	LastActivityAt string       `json:"last_activity_at"`
	LastMessage    *LastMessage `json:"last_message,omitempty"`
	PresenceStatus string       `json:"presence_status,omitempty"`
	CreatedAt      string       `json:"created_at,omitempty"`
	CanMessage     bool         `json:"can_message"`
	BlockReason    string       `json:"message_blocked_reason,omitempty"`
}

type PaginatedPrivateChats struct {
	Items      []PrivateChatItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type MyPrivateChatsRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type OpenTheDoorRequest struct {
	TargetUserID string `json:"target_user_id"`
}

type UpdatePrivateChatSettingsRequest struct {
	TargetUserID string `json:"target_user_id"`
	IsMuted      *bool  `json:"is_muted,omitempty"`
	IsPinned     *bool  `json:"is_pinned,omitempty"`
}

type DeletePrivateChatRequest struct {
	TargetUserID string `json:"target_user_id"`
}

type PrivateChatOpenResult struct {
	Chat           PrivateChatItem                    `json:"chat"`
	RecentMessages private_messages.PaginatedMessages `json:"recent_messages"`
}
