package reports

import "context"

type Repository interface {
	Create(context.Context, string, CreateReportRequest) ReportCreated
	List(context.Context, string) PaginatedReports
}
