package private_messages

type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type PrivateMessageItem struct {
	MessageID  string   `json:"message_id"`
	Type       string   `json:"type"`
	Content    string   `json:"content,omitempty"`
	ShareCode  string   `json:"share_code,omitempty"`
	Sender     UserInfo `json:"sender"`
	SentAt     string   `json:"sent_at"`
	EditedAt   string   `json:"edited_at,omitempty"`
	RecalledAt string   `json:"recalled_at,omitempty"`
	IsRead     bool     `json:"is_read"`
}

type PaginatedMessages struct {
	Items      []PrivateMessageItem `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type SendPrivateMessageRequest struct {
	TargetUserID     string `json:"target_user_id"`
	Type             string `json:"type"`
	Content          string `json:"content,omitempty"`
	FileID           string `json:"file_id,omitempty"`
	ShareCode        string `json:"share_code,omitempty"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
}

type GetHistoryRequest struct {
	TargetUserID string `json:"target_user_id"`
}

type MessageSent struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}
