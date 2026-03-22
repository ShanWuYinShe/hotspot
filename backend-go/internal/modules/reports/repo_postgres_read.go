package reports

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string) (PaginatedReports, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.List(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedReports{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListReports, userID)
	if err != nil {
		return PaginatedReports{}, err
	}
	defer rows.Close()
	items := []ReportItem{}
	for rows.Next() {
		var item ReportItem
		if err := rows.Scan(&item.ReportID, &item.TargetType, &item.TargetID, &item.Reason, &item.Description, &item.Status, &item.CreatedAt); err != nil {
			return PaginatedReports{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedReports{}, err
	}
	return PaginatedReports{Items: items}, nil
}
