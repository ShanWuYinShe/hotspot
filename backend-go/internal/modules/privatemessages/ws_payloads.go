package privatemessages

// ws_payloads.go — privatemessages 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 实现侧镜像）。omitempty 对齐原 map 的「条件写入键」。

type PrivateMessageSentPayload struct {
	MessageID    string `json:"message_id"`
	TargetUserID string `json:"target_user_id"`
	// Sender 承载 resolveUserInfo 的返回（条件键 map：avatar_url 为空时省略键，
	// 与 types.UserInfo 的恒定 nullable 语义不同，保持原行为故用 any）。
	Sender    any    `json:"sender"`
	Type      string `json:"type"`
	SentAt    string `json:"sent_at"`
	Content   string `json:"content,omitempty"`
	ShareCode string `json:"share_code,omitempty"`
	// ReplyTo / Media 为 buildReplyTo / buildMedia 的动态对象（可能缺省）。
	ReplyTo any `json:"reply_to,omitempty"`
	Media   any `json:"media,omitempty"`
}

type PrivateMessageEditedPayload struct {
	MessageID    string `json:"message_id"`
	TargetUserID string `json:"target_user_id"`
	Content      string `json:"content"`
	EditedAt     string `json:"edited_at"`
}

type PrivateMessageRecalledPayload struct {
	MessageID    string `json:"message_id"`
	TargetUserID string `json:"target_user_id"`
	// RecalledBy 承载 resolveUserInfo 的返回（条件键 map，同 Sender）。
	RecalledBy any    `json:"recalled_by"`
	RecalledAt string `json:"recalled_at"`
}

type PrivateMessageReadPayload struct {
	TargetUserID  string `json:"target_user_id"`
	UpToMessageID string `json:"up_to_message_id"`
	ReaderID      string `json:"reader_id"`
	ReadAt        string `json:"read_at"`
}
