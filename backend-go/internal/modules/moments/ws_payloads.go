package moments

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — moments 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 实现侧镜像）。struct 化后字段漂移在编译期暴露；
// omitempty 对齐原 map 的「条件写入键」行为，序列化结果不变。

// 瞬间被编辑（仅作者其他在线设备，多端同步）。
type MomentUpdatedPayload struct {
	MomentID  string `json:"moment_id"`
	UpdatedAt string `json:"updated_at"`
}

// 瞬间被软删除（作者其他设备 + 收藏者）。
type MomentDeletedPayload struct {
	MomentID  string `json:"moment_id"`
	DeletedAt string `json:"deleted_at"`
}

// 瞬间内容中被 @（spec ContentMentionedPayload）。
type ContentMentionedPayload struct {
	NotifyID      string              `json:"notify_id"`
	TargetType    string              `json:"target_type"`
	MomentID      string              `json:"moment_id"`
	MomentSummary types.MomentSummary `json:"moment_summary"`
	Actor         types.UserInfo      `json:"actor"`
	MentionedAt   string              `json:"mentioned_at"`
}
