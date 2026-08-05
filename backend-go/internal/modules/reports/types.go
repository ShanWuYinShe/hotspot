package reports

import "hotspot/backend-go/internal/platform/types"

type ReportTargetType string

type ReportReason string

type ReportStatus string

const (
	ReportTargetTypeUser    ReportTargetType = "user"
	ReportTargetTypeMoment  ReportTargetType = "moment"
	ReportTargetTypeComment ReportTargetType = "comment"
	ReportTargetTypeMessage ReportTargetType = "message"
	ReportTargetTypeGroup   ReportTargetType = "group"

	ReportReasonSpam           ReportReason = "spam"
	ReportReasonHarassment     ReportReason = "harassment"
	ReportReasonViolence       ReportReason = "violence"
	ReportReasonHateSpeech     ReportReason = "hate_speech"
	ReportReasonSexualContent  ReportReason = "sexual_content"
	ReportReasonMisinformation ReportReason = "misinformation"
	ReportReasonImpersonation  ReportReason = "impersonation"
	ReportReasonSelfHarm       ReportReason = "self_harm"
	ReportReasonOther          ReportReason = "other"

	// ReportStatus 收敛为三态（spec api_spec_types.md）：reviewing/escalated 已删除
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

type PaginationParams = types.PaginationParams

type CreateReportRequest struct {
	TargetType  ReportTargetType `json:"target_type"`
	TargetID    string           `json:"target_id"`
	Reason      ReportReason     `json:"reason"`
	Description string           `json:"description,omitempty"`
	// MessageSnapshot 为消息举报的内容快照（spec api_spec_types.md
	// ReportSnapshot）：target_type='message' 时必填，服务端原样存储，
	// 管理员基于快照审核，不直接查询私聊/群聊消息库（隐私边界）。
	MessageSnapshot *ReportSnapshot `json:"message_snapshot,omitempty"`
}

// ReportSnapshot 是消息举报时的消息内容快照（spec api_spec_types.md
// ReportSnapshot）：举报人提交，服务端原样存储，管理员审核基于快照判断。
type ReportSnapshot struct {
	ChatType        string         `json:"chat_type"`
	TargetChatID    string         `json:"target_chat_id"`
	MessageType     string         `json:"message_type"`
	SnapshotContent string         `json:"snapshot_content,omitempty"`
	SnapshotMedia   map[string]any `json:"snapshot_media,omitempty"`
	SentAt          string         `json:"sent_at"`
}

type MyReportsRequest struct {
	Status     ReportStatus      `json:"status,omitempty"`
	TargetType ReportTargetType  `json:"target_type,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type ReportCreated struct {
	ReportID    string `json:"report_id"`
	IsDuplicate bool   `json:"is_duplicate"`
}

type ReportItem struct {
	ReportID    string           `json:"report_id"`
	TargetType  ReportTargetType `json:"target_type"`
	TargetID    string           `json:"target_id"`
	Reason      ReportReason     `json:"reason"`
	Description string           `json:"description,omitempty"`
	Status      ReportStatus     `json:"status"`
	CreatedAt   string           `json:"created_at"`
	HandledAt   string           `json:"handled_at,omitempty"`
	// MessageSnapshot 仅 target_type='message' 时返回（spec api_spec_types.md
	// ReportItem.message_snapshot），供举报人回看与管理员审核。
	MessageSnapshot *ReportSnapshot `json:"message_snapshot,omitempty"`
}

type PaginatedReports struct {
	Items      []ReportItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}
