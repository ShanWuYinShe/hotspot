package reports

import "context"

type Repository interface {
	Create(context.Context, string, CreateReportRequest) (ReportCreated, error)
	List(context.Context, string, MyReportsRequest) (PaginatedReports, error)
	ExistsOpenReport(context.Context, ReportTargetType, string) (bool, error)
	TargetOwner(context.Context, ReportTargetType, string) (string, bool, error)
}
