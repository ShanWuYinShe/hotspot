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
	Description *string          `json:"description"`
}

// MessageMedia 是消息附件/媒体信息（spec api_spec_types.md MessageMedia =
// MediaAsset 基础上补充文件元数据）。恒定 nullable 语义：thumbnail_url null =
// 无缩略图，width/height/mime_type null = 未知，file_name/file_size/duration
// null = 不适用（duration 仅音视频非 null）。
type MessageMedia struct {
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnail_url"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
	MimeType     *string `json:"mime_type"`
	FileName     *string `json:"file_name"`
	FileSize     *int    `json:"file_size"`
	Duration     *int    `json:"duration"`
}

// ReportSnapshot 是消息举报的服务端快照（spec api_spec_types.md
// ReportSnapshot）：受理时基于消息真值生成并冻结（客户端不提交快照，
// 防伪造/篡改证据链），管理员审核基于快照判断。snapshot_media 为完整
// MessageMedia（spec api_spec_types.md ReportSnapshot.snapshot_media），
// 文本类型为 null。
type ReportSnapshot struct {
	ChatType        string        `json:"chat_type"`
	TargetChatID    string        `json:"target_chat_id"`
	MessageType     string        `json:"message_type"`
	SnapshotContent *string       `json:"snapshot_content"`
	SnapshotMedia   *MessageMedia `json:"snapshot_media"`
	SentAt          string        `json:"sent_at"`
}

type ListReportsRequest struct {
	Status     ReportStatus      `json:"status,omitempty"`
	TargetType ReportTargetType  `json:"target_type,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type ReportCreated struct {
	ReportID    string `json:"report_id"`
	IsDuplicate bool   `json:"is_duplicate"`
}

// ReportAction 是举报处理动作（spec api_spec_types.md ReportAction）。
type ReportAction string

const (
	ReportActionResolve ReportAction = "resolve"
	ReportActionDismiss ReportAction = "dismiss"
)

// ReportResolution 是举报处理结果摘要（spec api_spec_types.md
// ReportItem.resolution，@owner-only）：仅举报人自己可见（reports/list
// 返回），不暴露处理管理员身份。
type ReportResolution struct {
	Action     ReportAction `json:"action"`
	ResolvedAt string       `json:"resolved_at"`
	Note       *string      `json:"note"`
}

type ReportItem struct {
	ReportID    string           `json:"report_id"`
	TargetType  ReportTargetType `json:"target_type"`
	TargetID    string           `json:"target_id"`
	Reason      ReportReason     `json:"reason"`
	Description *string          `json:"description"`
	Status      ReportStatus     `json:"status"`
	CreatedAt   string           `json:"created_at"`
	HandledAt   *string          `json:"handled_at"`
	// MessageSnapshot 恒定 nullable：null = 非消息举报（spec api_spec_types.md
	// ReportItem.message_snapshot），供举报人回看与管理员审核。
	MessageSnapshot *ReportSnapshot `json:"message_snapshot"`
	// Resolution 恒定 nullable：null = 尚未处理（spec api_spec_types.md）。
	Resolution *ReportResolution `json:"resolution"`
}

type PaginatedReports struct {
	Items      []ReportItem `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}
