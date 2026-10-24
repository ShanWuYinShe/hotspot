package reports

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateReportRequest, messageSnapshot *ReportSnapshot) (ReportCreated, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return ReportCreated{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	reportID, err := idgen.WithPrefix("report_")
	if err != nil {
		return ReportCreated{}, err
	}
	var description any
	if req.Description != nil {
		description = *req.Description
	}
	var snapshotJSON any
	if messageSnapshot != nil {
		raw, err := json.Marshal(messageSnapshot)
		if err != nil {
			return ReportCreated{}, err
		}
		snapshotJSON = string(raw)
	}
	if err := queryable.Exec(ctx, queryInsertReport, reportID, userID, string(req.TargetType), req.TargetID, string(req.Reason), description, string(ReportStatusPending), now.Format(time.RFC3339), snapshotJSON); err != nil {
		return ReportCreated{}, err
	}
	return ReportCreated{ReportID: reportID}, nil
}

func (r *PostgresRepository) FindExistingReport(ctx context.Context, userID string, targetType ReportTargetType, targetID string) (string, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	var reportID string
	if err := queryable.QueryRow(ctx, queryFindExistingReport, string(targetType), targetID, userID).Scan(&reportID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return reportID, true, nil
}

func (r *PostgresRepository) TargetOwner(ctx context.Context, targetType ReportTargetType, targetID string) (string, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	query := queryReportTargetOwnerUser
	switch targetType {
	case ReportTargetTypeUser:
		query = queryReportTargetOwnerUser
	case ReportTargetTypeMoment:
		query = queryReportTargetOwnerMoment
	case ReportTargetTypeComment:
		query = queryReportTargetOwnerComment
	case ReportTargetTypeMessage:
		query = queryReportTargetOwnerMessage
	case ReportTargetTypeGroup:
		query = queryReportTargetOwnerGroup
	default:
		return "", false, nil
	}
	var ownerUserID string
	if err := queryable.QueryRow(ctx, query, targetID).Scan(&ownerUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	return ownerUserID, ownerUserID != "", nil
}
