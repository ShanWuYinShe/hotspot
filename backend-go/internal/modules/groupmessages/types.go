package groupmessages

type GroupMessageItem struct {
	MessageID   string                   `json:"message_id"`
	Type        string                   `json:"type"`
	Content     string                   `json:"content,omitempty"`
	Media       map[string]any           `json:"media,omitempty"`
	ShareCode   string                   `json:"share_code,omitempty"`
	Sender      map[string]any           `json:"sender"` // UserInfo
	ReplyTo     map[string]any           `json:"reply_to,omitempty"`
	Mentions    []string                 `json:"mentions,omitempty"`
	RecalledBy  map[string]any           `json:"recalled_by,omitempty"`
	SentAt      string                   `json:"sent_at"`
	EditedAt    string                   `json:"edited_at,omitempty"`
	RecalledAt  string                   `json:"recalled_at,omitempty"`
	ReadCount   int                      `json:"read_count,omitempty"`
}
type MessageSent struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}
type SendGroupMessageRequest struct {
	GroupID          string   `json:"group_id"`
	Type             string   `json:"type"`
	Content          string   `json:"content,omitempty"`
	FileID           string   `json:"file_id,omitempty"`
	ShareCode        string   `json:"share_code,omitempty"`
	ReplyToMessageID string   `json:"reply_to_message_id,omitempty"`
	Mentions         []string `json:"mentions,omitempty"`
}
type GetGroupMessagesHistoryRequest struct {
	GroupID string `json:"group_id"`
}
type PaginatedGroupMessages struct {
	Items []GroupMessageItem `json:"items"`
}

// 新增请求类型
type UpdateGroupMessageRequest struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

type RecallGroupMessageRequest struct {
	MessageID string `json:"message_id"`
}

type MarkGroupMessagesReadRequest struct {
	GroupID       string `json:"group_id"`
	UpToMessageID string `json:"up_to_message_id,omitempty"`
}

// MarkAllRead 使用空对象 {}
