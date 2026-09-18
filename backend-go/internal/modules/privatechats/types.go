package privatechats

import (
	"time"

	"hotspot/backend-go/internal/modules/privatemessages"
	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

type PaginationParams = types.PaginationParams

type chatCursor struct {
	// IsPinned 必须参与游标：排序为「置顶优先 → 活跃时间倒序」，若游标缺失
	// 置顶维度，页边界落在置顶区时下一页会把全部非置顶会话过滤掉（不可达）。
	IsPinned       bool   `json:"is_pinned"`
	LastActivityAt string `json:"last_activity_at"`
	UserID         string `json:"user_id"`
}

func encodeChatCursor(isPinned bool, lastActivityAt, userID string) string {
	value, err := platformcursor.Encode("private_chats.list", chatCursor{IsPinned: isPinned, LastActivityAt: lastActivityAt, UserID: userID})
	if err != nil {
		return ""
	}
	return value
}

func decodeChatCursor(value string) (bool, time.Time, string, error) {
	if value == "" {
		return false, time.Time{}, "", nil
	}
	var decoded chatCursor
	if err := platformcursor.Decode("private_chats.list", value, &decoded); err != nil || decoded.LastActivityAt == "" || decoded.UserID == "" {
		return false, time.Time{}, "", privatemessages.ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.LastActivityAt)
	if err != nil {
		return false, time.Time{}, "", privatemessages.ErrInvalidCursor
	}
	return decoded.IsPinned, parsed.UTC(), decoded.UserID, nil
}

type UserInfo = types.UserInfo

type LastMessage struct {
	MessageID string   `json:"message_id"`
	Content   *string  `json:"content"`
	Type      string   `json:"type"`
	ShareCode *string  `json:"share_code"`
	Sender    UserInfo `json:"sender"`
	SentAt    string   `json:"sent_at"`
}

type PrivateChatItem struct {
	OtherUser      UserInfo     `json:"other_user"`
	UnreadCount    int          `json:"unread_count"`
	IsMuted        bool         `json:"is_muted"`
	IsPinned       bool         `json:"is_pinned"`
	LastActivityAt string       `json:"last_activity_at"`
	LastMessage    *LastMessage `json:"last_message"`
	// presence_status 恒定 nullable（null = 对方 presence_visible=false 隐藏）
	PresenceStatus *string `json:"presence_status"`
	CreatedAt      *string `json:"created_at"`
	CanMessage     bool    `json:"can_message"`
	// blocked_reason 恒定 nullable：can_message=false 时返回 blocked_by_user /
	// privacy_setting / stranger_quota_exceeded，否则为 null
	ForbiddenReason *httpx.ForbiddenReason `json:"forbidden_reason"`
}

type PaginatedPrivateChats struct {
	Items      []PrivateChatItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
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
	Chat           PrivateChatItem                   `json:"chat"`
	RecentMessages privatemessages.PaginatedMessages `json:"recent_messages"`
}
