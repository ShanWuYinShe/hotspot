package users

// ws_payloads.go — users 模块的 WebSocket 出站 payload 类型
//（spec api_spec_websocket.md UserProfileUpdatedPayload 镜像）。

type UserProfileUpdatedPayload struct {
	UserID    string `json:"user_id"`
	UpdatedAt string `json:"updated_at"`
}
