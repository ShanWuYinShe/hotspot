package reports

import "context"

type Repository interface {
	// Create 落库举报；messageSnapshot 为服务端生成的消息快照
	//（target_type='message' 时非 nil），随举报一并冻结存储。
	Create(context.Context, string, CreateReportRequest, *ReportSnapshot) (ReportCreated, error)
	List(context.Context, string, ListReportsRequest) (PaginatedReports, error)
	FindExistingReport(context.Context, string, ReportTargetType, string) (string, bool, error)
	TargetOwner(context.Context, ReportTargetType, string) (string, bool, error)
}
