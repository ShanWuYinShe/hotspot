package groupmessages

type GroupMessageItem struct {
	MessageID string `json:"message_id"`
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	SentAt    string `json:"sent_at"`
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
