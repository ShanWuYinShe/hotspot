package privatechats

import private_messages "hotspot/backend-go/internal/modules/privatemessages"

type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type LastMessage struct {
	MessageID string   `json:"message_id"`
	Content   string   `json:"content,omitempty"`
	Type      string   `json:"type"`
	ShareCode string   `json:"share_code,omitempty"`
	Sender    UserInfo `json:"sender"`
	SentAt    string   `json:"sent_at"`
}

type PrivateChatItem struct {
	OtherUser            UserInfo     `json:"other_user"`
	UnreadCount          int          `json:"unread_count"`
	IsMuted              bool         `json:"is_muted"`
	IsPinned             bool         `json:"is_pinned"`
	LastActivityAt       string       `json:"last_activity_at"`
	LastMessage          *LastMessage `json:"last_message,omitempty"`
	PresenceStatus       string       `json:"presence_status,omitempty"`
	CreatedAt            string       `json:"created_at,omitempty"`
	CanMessage           bool         `json:"can_message"`
	MessageBlockedReason string       `json:"message_blocked_reason,omitempty"`
}

type PaginatedPrivateChats struct {
	Items      []PrivateChatItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type OpenTheDoorRequest struct {
	TargetUserID string `json:"target_user_id"`
}

type PrivateChatOpenResult struct {
	Chat           PrivateChatItem                    `json:"chat"`
	RecentMessages private_messages.PaginatedMessages `json:"recent_messages"`
}
