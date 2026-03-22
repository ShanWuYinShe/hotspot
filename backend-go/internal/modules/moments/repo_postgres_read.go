package moments

import (
	"context"
	"errors"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Get(ctx, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MomentItem{}, false, persistence.ErrNotImplemented
	}
	item, err := scanMomentRow(queryable.QueryRow(ctx, queryGetMoment, momentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MomentItem{}, false, nil
		}
		return MomentItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) ListMine(ctx context.Context, userID string) (PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMine(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListMyMoments, userID)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := []MomentItem{}
	for rows.Next() {
		item, err := scanMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	return PaginatedMoments{Items: items}, nil
}

func (r *PostgresRepository) Exists(ctx context.Context, momentID string) (bool, error) {
	_, ok, err := r.Get(ctx, momentID)
	return ok, err
}

func scanMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	var item MomentItem
	var userID, createdAt, locationName, privacyLevel string
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel); err != nil {
		return MomentItem{}, err
	}
	item.User = UserInfo{UserID: userID, Nickname: userID}
	item.CreatedAt = createdAt
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, nil
}
