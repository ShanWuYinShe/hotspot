package reports

import "context"

type Repository interface {
	Create(context.Context, string, CreateReportRequest) (ReportCreated, error)
	List(context.Context, string, ListReportsRequest) (PaginatedReports, error)
	FindExistingReport(context.Context, string, ReportTargetType, string) (string, bool, error)
	TargetOwner(context.Context, ReportTargetType, string) (string, bool, error)
}
