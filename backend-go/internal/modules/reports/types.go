package reports

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

	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewing ReportStatus = "reviewing"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type CreateReportRequest struct {
	TargetType  ReportTargetType `json:"target_type"`
	TargetID    string           `json:"target_id"`
	Reason      ReportReason     `json:"reason"`
	Description string           `json:"description,omitempty"`
}

type MyReportsRequest struct {
	Status     ReportStatus      `json:"status,omitempty"`
	TargetType ReportTargetType  `json:"target_type,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type ReportCreated struct {
	ReportID string `json:"report_id"`
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
}

type PaginatedReports struct {
	Items      []ReportItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}
