package moments

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PersistenceRepository) Get(ctx context.Context, momentID string) (MomentItem, bool) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.memory.Get(ctx, momentID)
	}
	var item MomentItem
	var userID, createdAt, locationName, privacyLevel string
	var latitude, longitude float64
	if err := queryable.QueryRow(ctx, queryGetMoment, momentID).Scan(&item.MomentID, &userID, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel); err != nil {
		return r.memory.Get(ctx, momentID)
	}
	item.User = UserInfo{UserID: userID, Nickname: userID}
	item.CreatedAt = createdAt
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, true
}

func (r *PersistenceRepository) ListMine(ctx context.Context, userID string) PaginatedMoments {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.memory.ListMine(ctx, userID)
	}
	rows, err := queryable.Query(ctx, queryListMyMoments, userID)
	if err != nil {
		return r.memory.ListMine(ctx, userID)
	}
	defer rows.Close()
	items := []MomentItem{}
	for rows.Next() {
		var item MomentItem
		var rowUserID, createdAt, locationName, privacyLevel string
		var latitude, longitude float64
		if err := rows.Scan(&item.MomentID, &rowUserID, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel); err != nil {
			return r.memory.ListMine(ctx, userID)
		}
		item.User = UserInfo{UserID: rowUserID, Nickname: rowUserID}
		item.CreatedAt = createdAt
		item.Location = Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
		item.PrivacyLevel = privacyLevel
		items = append(items, item)
	}
	if len(items) == 0 {
		return r.memory.ListMine(ctx, userID)
	}
	return PaginatedMoments{Items: items}
}
