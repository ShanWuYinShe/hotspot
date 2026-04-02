package private_messages

type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type MessageMedia struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
}

type MessageReplyTo struct {
	MessageID string   `json:"message_id"`
	Type      string   `json:"type"`
	Content   string   `json:"content,omitempty"`
	ShareCode string   `json:"share_code,omitempty"`
	Sender    UserInfo `json:"sender"`
	SentAt    string   `json:"sent_at"`
}

type PrivateMessageItem struct {
	MessageID  string          `json:"message_id"`
	Type       string          `json:"type"`
	Content    string          `json:"content,omitempty"`
	Media      *MessageMedia   `json:"media,omitempty"`
	ShareCode  string          `json:"share_code,omitempty"`
	Sender     UserInfo        `json:"sender"`
	ReplyTo    *MessageReplyTo `json:"reply_to,omitempty"`
	SentAt     string          `json:"sent_at"`
	EditedAt   string          `json:"edited_at,omitempty"`
	RecalledAt string          `json:"recalled_at,omitempty"`
	IsRead     bool            `json:"is_read"`
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
	TargetUserID string            `json:"target_user_id"`
	Pagination   *PaginationParams `json:"pagination,omitempty"`
}

type UpdatePrivateMessageRequest struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

type RecallPrivateMessageRequest struct {
	MessageID string `json:"message_id"`
}

type MarkPrivateMessagesReadRequest struct {
	TargetUserID  string `json:"target_user_id"`
	UpToMessageID string `json:"up_to_message_id,omitempty"`
}

type MessageSent struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}
