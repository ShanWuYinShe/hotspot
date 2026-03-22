package groups

import (
	"context"
	"errors"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Get(ctx, groupID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupItem{}, false, persistence.ErrNotImplemented
	}
	var item GroupItem
	var locationName, joinMethod string
	var latitude, longitude float64
	if err := queryable.QueryRow(ctx, queryGetGroup, groupID).Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupItem{}, false, nil
		}
		return GroupItem{}, false, err
	}
	item.Group.MembersCount = 1
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.JoinMethod = GroupJoinMethod(joinMethod)
	return item, true, nil
}

func (r *PostgresRepository) Search(ctx context.Context, keyword string) (PaginatedGroups, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Search(ctx, keyword)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedGroups{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, querySearchGroups, keyword)
	if err != nil {
		return PaginatedGroups{}, err
	}
	defer rows.Close()
	items := []GroupItem{}
	for rows.Next() {
		var item GroupItem
		var locationName, joinMethod string
		var latitude, longitude float64
		if err := rows.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers); err != nil {
			return PaginatedGroups{}, err
		}
		item.Group.MembersCount = 1
		item.Location = Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
		item.JoinMethod = GroupJoinMethod(joinMethod)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedGroups{}, err
	}
	return PaginatedGroups{Items: items}, nil
}
