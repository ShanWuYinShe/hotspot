package reports

import (
	"context"
	"encoding/json"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) List(ctx context.Context, userID string, req MyReportsRequest) (PaginatedReports, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedReports{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedReports{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListReports, userID, string(req.Status), string(req.TargetType))
	if err != nil {
		return PaginatedReports{}, err
	}
	defer rows.Close()
	items := []ReportItem{}
	for rows.Next() {
		var item ReportItem
		var snapshotJSON *string
		var handledAt *time.Time
		var handleNote string
		if err := rows.Scan(&item.ReportID, &item.TargetType, &item.TargetID, &item.Reason, &item.Description, &item.Status, &item.CreatedAt, &snapshotJSON, &handledAt, &handleNote); err != nil {
			return PaginatedReports{}, err
		}
		if snapshotJSON != nil && *snapshotJSON != "" {
			var snapshot ReportSnapshot
			if err := json.Unmarshal([]byte(*snapshotJSON), &snapshot); err != nil {
				return PaginatedReports{}, err
			}
			item.MessageSnapshot = &snapshot
		}
		// resolution 组装（@owner-only，spec api_spec_types.md）：action 从
		// 终态 status 推断，不暴露处理管理员身份。
		if handledAt != nil && (item.Status == ReportStatusResolved || item.Status == ReportStatusDismissed) {
			action := ReportActionDismiss
			if item.Status == ReportStatusResolved {
				action = ReportActionResolve
			}
			item.Resolution = &ReportResolution{
				Action:     action,
				ResolvedAt: handledAt.UTC().Format(time.RFC3339),
				Note:       handleNote,
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedReports{}, err
	}
	return PaginatedReports{Items: items}, nil
}
