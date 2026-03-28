package users

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"time"

	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.GetProfile(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MyProfile{}, false, persistence.ErrNotImplemented
	}
	var profile MyProfile
	var avatarURL, bio, gender, email, phone string
	var locationName string
	var latitude, longitude float64
	var presenceStatus, userType, lastOnline, createdAt string
	if err := queryable.QueryRow(ctx, queryGetUserProfile, userID).Scan(&profile.UserID, &profile.Nickname, &avatarURL, &userType, &bio, &gender, &email, &phone, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyProfile{}, false, nil
		}
		return MyProfile{}, false, err
	}
	profile.AvatarURL = avatarURL
	profile.UserType = UserType(userType)
	profile.Bio = bio
	profile.Gender = Gender(gender)
	profile.Email = email
	profile.Phone = phone
	profile.Location = Location{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	profile.PresenceStatus = UserPresenceStatus(presenceStatus)
	profile.LastOnline = lastOnline
	profile.CreatedAt = createdAt
	return profile, true, nil
}

func (r *PostgresRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.GetSettings(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return UserSettings{}, false, persistence.ErrNotImplemented
	}
	var settings UserSettings
	var allowPrivateChat string
	if err := queryable.QueryRow(ctx, queryGetUserSettings, userID).Scan(&settings.PrivacyLevel, &settings.LocationVisible, &settings.PresenceVisible, &settings.SocialListsVisible, &settings.GroupsVisible, &settings.LikedMomentsVisible, &settings.StatsVisible, &settings.AllowFollow, &allowPrivateChat, &settings.Searchable, &settings.PhoneSearchable, &settings.NicknameSearchable, &settings.LocationSearchable); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserSettings{}, false, nil
		}
		return UserSettings{}, false, err
	}
	settings.AllowPrivateChat = PrivateChatPermission(allowPrivateChat)
	return settings, true, nil
}

func (r *PostgresRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) ([]MyProfile, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.SearchProfiles(ctx, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, querySearchUserProfiles, req.Keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MyProfile{}
	for rows.Next() {
		var profile MyProfile
		var avatarURL, bio, gender, email, phone string
		var locationName string
		var latitude, longitude float64
		var presenceStatus, userType, lastOnline, createdAt string
		if err := rows.Scan(&profile.UserID, &profile.Nickname, &avatarURL, &userType, &bio, &gender, &email, &phone, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &createdAt); err != nil {
			return nil, err
		}
		profile.AvatarURL = avatarURL
		profile.UserType = UserType(userType)
		profile.Bio = bio
		profile.Gender = Gender(gender)
		profile.Email = email
		profile.Phone = phone
		profile.Location = Location{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
		profile.PresenceStatus = UserPresenceStatus(presenceStatus)
		profile.LastOnline = lastOnline
		profile.CreatedAt = createdAt
		items = append(items, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) ListBlocked(ctx context.Context, userID string) ([]BlockedUserItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListBlocked(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListBlockedUsers, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BlockedUserItem{}
	for rows.Next() {
		var targetUserID, remark string
		if err := rows.Scan(&targetUserID, &remark); err != nil {
			return nil, err
		}
		items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: targetUserID}, Remark: remark})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) Relationship(ctx context.Context, userID, targetUserID string) (Relationship, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Relationship(ctx, userID, targetUserID)
	}
	following, blocked, _, err := r.getRelationFields(ctx, userID, targetUserID)
	if err != nil {
		return RelationshipNone, err
	}
	if blocked {
		return RelationshipBlocked, nil
	}
	if following {
		return RelationshipFollowing, nil
	}
	return RelationshipNone, nil
}

func (r *PostgresRepository) getRelationFields(ctx context.Context, userID, targetUserID string) (bool, bool, string, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, false, "", persistence.ErrNotImplemented
	}
	var following, blocked bool
	var remark string
	if err := queryable.QueryRow(ctx, queryGetUserRelation, userID, targetUserID).Scan(&following, &blocked, &remark); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, "", nil
		}
		return false, false, "", err
	}
	return following, blocked, remark, nil
}

func listLikedMomentsPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string) (momenttypes.PaginatedMoments, error) {
	if factory == nil || factory.DB == nil || factory.Config.PostgresDSN == "" {
		return momenttypes.PaginatedMoments{}, persistence.ErrNotImplemented
	}
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return momenttypes.PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorLikedAt, cursorMomentID, err := decodeUserMomentCursor(cursor)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	rows, err := queryable.Query(ctx, queryListLikedMoments, userID, cursorLikedAt, cursorMomentID, limit+1)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]momenttypes.MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, likedAt, err := scanUserLikedMomentRow(rows)
		if err != nil {
			return momenttypes.PaginatedMoments{}, err
		}
		item.MyState = &momenttypes.MyMomentState{IsLiked: true}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		cursors = append(cursors, encodeUserMomentCursor(likedAt, item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadUserMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return momenttypes.PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			return momenttypes.PaginatedMoments{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadUserMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
	}
	return momenttypes.PaginatedMoments{Items: items}, nil
}

func scanUserLikedMomentRow(scanner interface{ Scan(...any) error }) (momenttypes.MomentItem, string, error) {
	var item momenttypes.MomentItem
	var userID, createdAt, locationName, privacyLevel, likedAt string
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &likedAt); err != nil {
		return momenttypes.MomentItem{}, "", err
	}
	item.User = momenttypes.UserInfo{UserID: userID, Nickname: userID}
	item.CreatedAt = createdAt
	item.Location = momenttypes.Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, likedAt, nil
}

type userMomentCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

func normalizeUserMomentLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 100:
		return 100
	default:
		return limit
	}
}

func encodeUserMomentCursor(at, id string) string {
	blob, err := json.Marshal(userMomentCursor{At: at, ID: id})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func decodeUserMomentCursor(cursor string) (any, string, error) {
	if cursor == "" {
		return nil, "", nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, "", persistence.ErrNotImplemented
	}
	var decoded userMomentCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	parsed, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", persistence.ErrNotImplemented
	}
	return parsed.UTC(), decoded.ID, nil
}

func loadUserMomentTags(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	tagsByMomentID := make(map[string][]string, len(momentIDs))
	for _, momentID := range momentIDs {
		rows, err := queryable.Query(ctx, momenttypesQueryListMomentTagsByMomentID, momentID)
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
