package privatemessages

// ws_payloads.go — privatemessages 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 实现侧镜像）。

// PrivateMessageSentPayload 是私聊新消息事件负载（spec →
// PrivateMessageNewPayload）。content/share_code/reply_to/media 为 spec 明确定义
// 的「恒定 nullable」字段：无值时必须显式下发 null，不能用 omitempty 省略键
// （否则纯文本帧缺 media/reply_to，客户端只能靠 undefined 容错，契约审计 B07）。
type PrivateMessageSentPayload struct {
	MessageID    string `json:"message_id"`
	TargetUserID string `json:"target_user_id"`
	// Sender 承载 resolveUserInfo 的返回（条件键 map：avatar_url 为空时省略键，
	// 与 types.UserInfo 的恒定 nullable 语义不同，保持原行为故用 any）。
	Sender    any     `json:"sender"`
	Type      string  `json:"type"`
	Content   *string `json:"content"`
	ShareCode *string `json:"share_code"`
	// ReplyTo / Media 为 buildReplyTo / buildMedia 的动态对象，nil → null。
	ReplyTo any    `json:"reply_to"`
	Media   any    `json:"media"`
	SentAt  string `json:"sent_at"`
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

// PrivateMessageReadPayload 已读回执（spec → PrivateMessageReadPayload）：
// 接收方天然知道是私聊对端读了消息，spec 明确不要冗余 target_user_id/reader_id，
// 只保留 up_to_message_id + read_at（契约审计 B14）。
type PrivateMessageReadPayload struct {
	UpToMessageID string `json:"up_to_message_id"`
	ReadAt        string `json:"read_at"`
}
