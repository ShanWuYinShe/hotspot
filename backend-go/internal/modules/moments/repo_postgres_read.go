package moments

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/persistence"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MomentItem{}, false, persistence.ErrNotImplemented
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

func (r *PostgresRepository) ListMine(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = normalizeMomentInteractionLimit(limit)
	cursorAt, cursorMomentID, err := decodeMomentMineCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	rows, err := queryable.Query(ctx, queryListMyMoments, userID, cursorAt, cursorMomentID, limit+1)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, err := scanMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		cursors = append(cursors, encodeMomentMineCursor(item.CreatedAt, item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, err
			}
			mentionsByMomentID, err := loadMomentMentions(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
				items[i].Mentions = mentionsByMomentID[items[i].MomentID]
			}
			return PaginatedMoments{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
		}
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

func encodeMomentMineCursor(createdAt, momentID string) string {
	blob, err := json.Marshal(momentInteractionCursor{At: createdAt, ID: momentID})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func decodeMomentMineCursor(cursor string) (any, string, error) {
	if cursor == "" {
		return nil, "", nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	var decoded momentInteractionCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	parsedAt, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return parsedAt.UTC(), decoded.ID, nil
}

func (r *PostgresRepository) LoadMyState(ctx context.Context, momentID, userID string) *MyMomentState {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" || userID == "" {
		return nil
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil
	}
	var liked int
	isLiked := false
	if err := queryable.QueryRow(ctx, queryGetMomentLike, momentID, userID).Scan(&liked); err == nil {
		isLiked = true
	}
	var favorited int
	isFavorited := false
	if err := queryable.QueryRow(ctx, queryCheckMomentFavorite, userID, momentID).Scan(&favorited); err == nil {
		isFavorited = true
	}
	return &MyMomentState{IsLiked: isLiked, IsFavorited: isFavorited}
}

func (r *PostgresRepository) Exists(ctx context.Context, momentID string) (bool, error) {
	_, ok, err := r.Get(ctx, momentID)
	return ok, err
}

func scanMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	var item MomentItem
	var userID, locationName, privacyLevel string
	var createdAt time.Time
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel); err != nil {
		return MomentItem{}, err
	}
	item.User = UserInfo{UserID: userID, Nickname: userID}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
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
	rows, err := queryable.Query(ctx, queryListMomentTagsByMomentIDs, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, tag string
		if err := rows.Scan(&momentID, &tag); err != nil {
			return nil, err
		}
		tagsByMomentID[momentID] = append(tagsByMomentID[momentID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tagsByMomentID, nil
}

func loadMomentMentions(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	mentionsByMomentID := make(map[string][]string, len(momentIDs))
	rows, err := queryable.Query(ctx, queryListMomentMentionsByMomentIDs, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, mentionedUserID string
		if err := rows.Scan(&momentID, &mentionedUserID); err != nil {
			return nil, err
		}
		mentionsByMomentID[momentID] = append(mentionsByMomentID[momentID], mentionedUserID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mentionsByMomentID, nil
}

func scanFavoriteMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, string, error) {
	var item MomentItem
	var userID, locationName, privacyLevel string
	var createdAt, favoritedAt time.Time
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &favoritedAt); err != nil {
		return MomentItem{}, "", err
	}
	item.User = UserInfo{UserID: userID, Nickname: userID}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, favoritedAt.UTC().Format(time.RFC3339), nil
}
