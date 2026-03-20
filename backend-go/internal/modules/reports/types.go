package reports

type CreateReportRequest struct {
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Reason      string `json:"reason"`
	Description string `json:"description,omitempty"`
}

type ReportCreated struct {
	ReportID string `json:"report_id"`
}

type ReportItem struct {
	ReportID    string `json:"report_id"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Reason      string `json:"reason"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type PaginatedReports struct {
	Items      []ReportItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}
