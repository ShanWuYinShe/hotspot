package reactions

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — reactions 模块的 WebSocket 出站 payload 类型
//（spec api_spec_websocket.md ReactionRemovedPayload 镜像）。

type ReactionRemovedPayload struct {
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	Emoji      string         `json:"emoji"`
	Actor      types.UserInfo `json:"actor"`
	RemovedAt  string         `json:"removed_at"`
}
