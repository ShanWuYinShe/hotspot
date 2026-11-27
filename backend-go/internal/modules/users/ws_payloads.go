package users

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — users 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 各事件负载的实现侧镜像）。struct 化后字段漂移在
// 编译期暴露；spec 定义的「恒定 nullable」字段必须恒在（无值下发 null），
// 故不加 omitempty。

// UserProfileUpdatedPayload 是用户资料变更事件负载（spec
// api_spec_websocket.md → UserProfileUpdatedPayload）：user 为更新后的
// 用户基础信息，客户端据此刷新本地 UserInfo 缓存。
type UserProfileUpdatedPayload struct {
	User      types.UserInfo `json:"user"`
	UpdatedAt string         `json:"updated_at"`
}

// NewFollowerPayload 是新关注事件负载（spec → NewFollowerPayload）。
// source/source_moment_id 为恒定 nullable：null = 自然关注 / 非瞬间来源。
type NewFollowerPayload struct {
	NotifyID       string         `json:"notify_id"`
	Actor          types.UserInfo `json:"actor"`
	FollowedAt     string         `json:"followed_at"`
	Source         *string        `json:"source"`
	SourceMomentID *string        `json:"source_moment_id"`
}
