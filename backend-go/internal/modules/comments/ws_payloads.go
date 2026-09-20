package comments

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — comments 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 实现侧镜像）。struct 化后字段漂移在编译期暴露。

type MomentCommentedPayload struct {
	NotifyID      string              `json:"notify_id"`
	MomentID      string              `json:"moment_id"`
	MomentSummary types.MomentSummary `json:"moment_summary"`
	CommentID     string              `json:"comment_id"`
	Actor         types.UserInfo      `json:"actor"`
	Content       string              `json:"content"`
	CommentedAt   string              `json:"commented_at"`
}

type CommentReplyPayload struct {
	NotifyID              string              `json:"notify_id"`
	MomentID              string              `json:"moment_id"`
	MomentSummary         types.MomentSummary `json:"moment_summary"`
	CommentID             string              `json:"comment_id"`
	RepliedCommentPreview string              `json:"replied_comment_preview"`
	ReplyID               string              `json:"reply_id"`
	Actor                 types.UserInfo      `json:"actor"`
	Content               string              `json:"content"`
	RepliedAt             string              `json:"replied_at"`
}

type ContentMentionedPayload struct {
	NotifyID       string              `json:"notify_id"`
	TargetType     string              `json:"target_type"`
	MomentID       string              `json:"moment_id"`
	CommentID      string              `json:"comment_id"`
	MomentSummary  types.MomentSummary `json:"moment_summary"`
	CommentPreview string              `json:"comment_preview"`
	Actor          types.UserInfo      `json:"actor"`
	MentionedAt    string              `json:"mentioned_at"`
}
