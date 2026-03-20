package groups

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) Get(ctx context.Context, groupID string) (GroupItem, bool) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.Get(ctx, groupID)
	}
	var item GroupItem
	var locationName, joinMethod string
	var latitude, longitude float64
	if err := queryable.QueryRow(ctx, queryGetGroup, groupID).Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers); err != nil {
		return r.fallback.Get(ctx, groupID)
	}
	item.Group.MembersCount = 1
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.JoinMethod = GroupJoinMethod(joinMethod)
	return item, true
}

func (r *PostgresRepository) Search(ctx context.Context, keyword string) PaginatedGroups {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.Search(ctx, keyword)
	}
	rows, err := queryable.Query(ctx, querySearchGroups, keyword)
	if err != nil {
		return r.fallback.Search(ctx, keyword)
	}
	defer rows.Close()
	items := []GroupItem{}
	for rows.Next() {
		var item GroupItem
		var locationName, joinMethod string
		var latitude, longitude float64
		if err := rows.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers); err != nil {
			return r.fallback.Search(ctx, keyword)
		}
		item.Group.MembersCount = 1
		item.Location = Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
		item.JoinMethod = GroupJoinMethod(joinMethod)
		items = append(items, item)
	}
	if len(items) == 0 {
		return r.fallback.Search(ctx, keyword)
	}
	return PaginatedGroups{Items: items}
}
