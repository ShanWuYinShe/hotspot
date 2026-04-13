package moments

import (
	"context"
	"errors"
	"sort"

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
	tagsByMomentID, err := loadMomentTags(ctx, queryable, []string{momentID})
	if err != nil {
		return MomentItem{}, false, err
	}
	item.Tags = tagsByMomentID[momentID]
	mentionsByMomentID, err := loadMomentMentions(ctx, queryable, []string{momentID})
	if err != nil {
		return MomentItem{}, false, err
	}
	item.Mentions = mentionsByMomentID[momentID]
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
	momentIDs := make([]string, 0)
	for rows.Next() {
		item, err := scanMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	mentionsByMomentID, err := loadMomentMentions(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
		items[i].Mentions = mentionsByMomentID[items[i].MomentID]
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

func loadMomentTags(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	tagsByMomentID := make(map[string][]string, len(momentIDs))
	for _, momentID := range momentIDs {
		rows, err := queryable.Query(ctx, queryListMomentTagsByMomentID, momentID)
		if err != nil {
			return nil, err
		}
		tags := make([]string, 0)
		for rows.Next() {
			var tag string
			if err := rows.Scan(&tag); err != nil {
				rows.Close()
				return nil, err
			}
			tags = append(tags, tag)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		sort.Strings(tags)
		tagsByMomentID[momentID] = tags
	}
	return tagsByMomentID, nil
}

func loadMomentMentions(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	mentionsByMomentID := make(map[string][]string, len(momentIDs))
	for _, momentID := range momentIDs {
		rows, err := queryable.Query(ctx, queryListMomentMentionsByMomentID, momentID)
		if err != nil {
			return nil, err
		}
		mentions := make([]string, 0)
		for rows.Next() {
			var mentionedUserID string
			if err := rows.Scan(&mentionedUserID); err != nil {
				rows.Close()
				return nil, err
			}
			mentions = append(mentions, mentionedUserID)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		sort.Strings(mentions)
		mentionsByMomentID[momentID] = mentions
	}
	return mentionsByMomentID, nil
}

func scanFavoriteMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, string, error) {
	var item MomentItem
	var userID, createdAt, locationName, privacyLevel, favoritedAt string
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &favoritedAt); err != nil {
		return MomentItem{}, "", err
	}
	item.User = UserInfo{UserID: userID, Nickname: userID}
	item.CreatedAt = createdAt
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, favoritedAt, nil
}
