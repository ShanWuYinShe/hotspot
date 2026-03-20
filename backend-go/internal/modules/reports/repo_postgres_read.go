package reports

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string) PaginatedReports {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.List(ctx, userID)
	}
	rows, err := queryable.Query(ctx, queryListReports, userID)
	if err != nil {
		return r.fallback.List(ctx, userID)
	}
	defer rows.Close()
	items := []ReportItem{}
	for rows.Next() {
		var item ReportItem
		if err := rows.Scan(&item.ReportID, &item.TargetType, &item.TargetID, &item.Reason, &item.Description, &item.Status, &item.CreatedAt); err != nil {
			return r.fallback.List(ctx, userID)
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return r.fallback.List(ctx, userID)
	}
	return PaginatedReports{Items: items}
}
