package reports

import "context"

type Repository interface {
	Create(context.Context, string, CreateReportRequest) (ReportCreated, error)
	List(context.Context, string) (PaginatedReports, error)
}
