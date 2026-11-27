package reactions

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — reactions 模块的 WebSocket 出站 payload 类型
//（spec api_spec_websocket.md ReactionRemovedPayload 镜像）。

// ReactionAddedPayload 表情反应事件负载（spec →
// ReactionAddedPayload）：notify_id 为恒定 nullable（null = 自反应不落通知），
// 无值时必须显式下发 null，不得省略键（契约审计 B08）。
type ReactionAddedPayload struct {
	NotifyID   *string        `json:"notify_id"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	Emoji      string         `json:"emoji"`
	Actor      types.UserInfo `json:"actor"`
	ReactedAt  string         `json:"reacted_at"`
}

type ReactionRemovedPayload struct {
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	Emoji      string         `json:"emoji"`
	Actor      types.UserInfo `json:"actor"`
	RemovedAt  string         `json:"removed_at"`
}
