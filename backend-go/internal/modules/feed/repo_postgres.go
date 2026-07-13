package feed

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) ListMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedMoments{}, err
	}
	candidates, err := r.loadMomentCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedMoments{}, err
	}
	return paginateMoments(candidates, query.Limit, query.Cursor, query.Source)
}

func (r *PostgresRepository) ListGroups(ctx context.Context, query groupsQuery) (PaginatedGroups, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedGroups{}, err
	}
	candidates, err := r.loadGroupCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedGroups{}, err
	}
	return paginateGroups(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListUsers(ctx context.Context, query usersQuery) (PaginatedUsers, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedUsers{}, err
	}
	candidates, err := r.loadUserCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedUsers{}, err
	}
	return paginateUsers(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListTrending(ctx context.Context, viewerUserID string) (TrendingBundle, error) {
	queryable, err := r.queryable()
	if err != nil {
		return TrendingBundle{}, err
	}
	tags, err := loadTrendingTags(ctx, queryable, viewerUserID)
	if err != nil {
		return TrendingBundle{}, err
	}
	hotMoments, err := loadTrendingMoments(ctx, queryable, viewerUserID)
	if err != nil {
		return TrendingBundle{}, err
	}
	return TrendingBundle{TrendingTags: tags, HotMoments: hotMoments}, nil
}

func (r *PostgresRepository) Dismiss(ctx context.Context, cmd dismissCommand) error {
	queryable, err := r.queryable()
	if err != nil {
		return err
	}
	if err := ensureDismissTargetExists(ctx, queryable, cmd.TargetType, cmd.TargetID); err != nil {
		return err
	}
	return queryable.Exec(ctx, queryFeedDismissInsert, cmd.ViewerUserID, cmd.TargetType, cmd.TargetID)
}

func (r *PostgresRepository) queryable() (persistence.QueryableDB, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	return queryable, nil
}

// FetchMomentImageRows loads ordered image rows for the given moment IDs.
func (r *PostgresRepository) FetchMomentImageRows(ctx context.Context, momentIDs []string) (map[string][]momentImageRow, error) {
	queryable, err := r.queryable()
	if err != nil {
		return map[string][]momentImageRow{}, nil
	}
	return FetchMomentImageRows(ctx, queryable, momentIDs)
}

func (r *PostgresRepository) loadUserCandidates(ctx context.Context, queryable persistence.QueryableDB, query usersQuery) ([]userCandidate, error) {
	bounds := geo.BoundsAround(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return nil, nil
	}
	rows, err := queryable.Query(ctx, queryFeedUsers, mins, maxs, query.ViewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius, "distance")
	items := make([]userCandidate, 0)
	for rows.Next() {
		var userID, nickname, avatarURL, bio, locationName string
		var latitude, longitude float64
		var isFollowing, isFollowedBy bool
		var mutualFriendsCount, commonGroupsCount int
		if err := rows.Scan(&userID, &nickname, &avatarURL, &bio, &locationName, &latitude, &longitude, &isFollowing, &isFollowedBy, &mutualFriendsCount, &commonGroupsCount); err != nil {
			return nil, err
		}
		if userID == query.ViewerUserID {
			continue
		}
			distance, bearing, ok := plan.Match(geo.Point{Latitude: latitude, Longitude: longitude})
			if !ok {
				continue
			}
			d := float64(distance)
			b := float64(bearing)
			item := UserItem{
				User: UserInfo{UserID: userID, Nickname: nickname, AvatarURL: avatarURL},
				Bio:  strings.TrimSpace(bio),
				Location: &RelativeLocation{
					Point:          GeoPoint{Latitude: latitude, Longitude: longitude},
					LocationName:   locationName,
					Distance:       &d,
					BearingDegrees: &b,
				},
			Relation:      &UserRelation{IsFollowing: isFollowing, IsFollowedBy: isFollowedBy, IsBlocked: false},
			SocialContext: &UserSocialContext{MutualFriendsCount: mutualFriendsCount, CommonGroupsCount: commonGroupsCount},
		}
		items = append(items, userCandidate{Item: item, Distance: distance})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool { return compareUserCandidates(items[i], items[j]) < 0 })
	return items, nil
}

func (r *PostgresRepository) loadGroupCandidates(ctx context.Context, queryable persistence.QueryableDB, query groupsQuery) ([]groupCandidate, error) {
	bounds := geo.BoundsAround(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return nil, nil
	}
	rows, err := queryable.Query(ctx, queryFeedGroups, mins, maxs, query.ViewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius, "distance")
	items := make([]groupCandidate, 0)
	for rows.Next() {
		var groupID, name, avatarURL, description, locationName string
		var latitude, longitude float64
		var hasPassword, isJoined bool
		var membersCount, friendsInGroup int
		if err := rows.Scan(&groupID, &name, &avatarURL, &description, &hasPassword, &locationName, &latitude, &longitude, &membersCount, &isJoined, &friendsInGroup); err != nil {
			return nil, err
		}
			distance, bearing, ok := plan.Match(geo.Point{Latitude: latitude, Longitude: longitude})
			if !ok {
				continue
			}
			d := float64(distance)
			b := float64(bearing)
			item := GroupItem{
				Group:       GroupInfo{GroupID: groupID, Name: name, AvatarURL: avatarURL, MembersCount: membersCount},
				Description: strings.TrimSpace(description),
				// has_password is @member-only per spec; don't surface it to viewers
				// who haven't joined (avoid tipping off crawlers/brute-forcers).
				HasPassword:    hasPassword && isJoined,
				Location:       RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName, Distance: &d, BearingDegrees: &b},
			MyState:        &MyGroupState{IsJoined: isJoined},
			FriendsInGroup: friendsInGroup,
		}
		items = append(items, groupCandidate{Item: item, Distance: distance})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool { return compareGroupCandidates(items[i], items[j]) < 0 })
	return items, nil
}

func (r *PostgresRepository) loadMomentCandidates(ctx context.Context, queryable persistence.QueryableDB, query momentsQuery) ([]momentCandidate, error) {
	bounds := geo.BoundsAround(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 && query.Source != "following" {
		return nil, nil
	}
	var querySQL string
	var args []any
	switch query.Source {
	case "following":
		querySQL = queryFeedMomentsFollowing
		args = []any{query.ViewerUserID}
	case "nearby":
		querySQL = queryFeedMomentsNearby
		args = []any{mins, maxs, query.ViewerUserID}
	default:
		querySQL = queryFeedMomentsAll
		args = []any{mins, maxs, query.ViewerUserID}
	}
	rows, err := queryable.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius, "relevance")
	items := make([]momentCandidate, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isLiked, isFavorited bool
		var isFollowing bool
		if query.Source == "following" {
			if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited); err != nil {
				return nil, err
			}
			isFollowing = true
		} else {
			if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited, &isFollowing); err != nil {
				return nil, err
			}
		}
		distance, bearing, ok := plan.Match(geo.Point{Latitude: latitude, Longitude: longitude})
		if !ok && query.Source != "following" {
			continue
		}
		if query.Source == "following" {
				distance = geo.DistanceMeters(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, geo.Point{Latitude: latitude, Longitude: longitude})
				bearing = geo.BearingDegrees(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, geo.Point{Latitude: latitude, Longitude: longitude})
			}
			d := float64(distance)
			b := float64(bearing)
			score := likesCount + commentsCount + sharesCount
		if isFollowing {
			score += 1000
		}
		if query.Source == "nearby" {
			score += max(0, query.Radius-distance)
		}
		item := MomentItem{
			MomentID:      momentID,
			Description:   description,
			CreatedAt:     createdAt.UTC().Format(time.RFC3339),
			LikesCount:    likesCount,
			CommentsCount: commentsCount,
			SharesCount:   sharesCount,
			User:          UserInfo{UserID: userID, Nickname: nickname, AvatarURL: avatarURL},
				Location:      RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName, Distance: &d, BearingDegrees: &b},
			MyState:       &MyMomentState{IsLiked: isLiked, IsFavorited: isFavorited},
		}
		reasonType := "trending"
		switch query.Source {
		case "following":
			reasonType = "following"
		case "nearby":
			reasonType = "nearby"
		case "all":
			if isFollowing {
				reasonType = "following"
			} else if distance <= query.Radius {
				reasonType = "nearby"
			}
		}
		item.Recommendation = &RecommendationContext{Type: reasonType}
		items = append(items, momentCandidate{Item: item, Score: score, Distance: distance, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareMomentCandidates(items[i], items[j], query.Source) < 0 })
	return items, nil
}

func ensureDismissTargetExists(ctx context.Context, queryable persistence.QueryableDB, targetType, targetID string) error {
	query := queryFeedDismissMomentExists
	switch targetType {
	case "moment":
		query = queryFeedDismissMomentExists
	case "user":
		query = queryFeedDismissUserExists
	case "group":
		query = queryFeedDismissGroupExists
	default:
		return ErrInvalidFeedRequest
	}
	var existingID string
	if err := queryable.QueryRow(ctx, query, targetID).Scan(&existingID); err != nil {
		if errors.Is(err, persistence.ErrNotImplemented) {
			return err
		}
		return ErrDismissTargetNotFound
	}
	if existingID == "" {
		return ErrDismissTargetNotFound
	}
	return nil
}

func loadTrendingTags(ctx context.Context, queryable persistence.QueryableDB, viewerUserID string) ([]TrendingTagItem, error) {
	rows, err := queryable.Query(ctx, queryFeedTrendingTags)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrendingTagItem, 0)
	for rows.Next() {
		var name string
		var usageCount int
		if err := rows.Scan(&name, &usageCount); err != nil {
			return nil, err
		}
		items = append(items, TrendingTagItem{Name: name, UsageCount: usageCount, Trend: "rising"})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].UsageCount != items[j].UsageCount {
			return items[i].UsageCount > items[j].UsageCount
		}
		return items[i].Name < items[j].Name
	})
	if len(items) == 0 {
		return items, nil
	}
	moments, err := loadTrendingMoments(ctx, queryable, viewerUserID)
	if err != nil {
		return nil, err
	}
	previewsByTag := make(map[string]*MomentItem, len(items))
	for _, moment := range moments {
		for _, tag := range moment.Tags {
			if previewsByTag[tag] == nil {
				copyMoment := moment
				previewsByTag[tag] = &copyMoment
			}
		}
	}
	for idx := range items {
		items[idx].HotMomentPreview = previewsByTag[items[idx].Name]
	}
	return items, nil
}

func loadTrendingMoments(ctx context.Context, queryable persistence.QueryableDB, viewerUserID string) ([]MomentItem, error) {
	rows, err := queryable.Query(ctx, queryFeedTrendingHotMoments, viewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]momentCandidate, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isLiked, isFavorited bool
		if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited); err != nil {
			return nil, err
		}
		item := MomentItem{
			MomentID:       momentID,
			Description:    description,
			CreatedAt:      createdAt.UTC().Format(time.RFC3339),
			LikesCount:     likesCount,
			CommentsCount:  commentsCount,
			SharesCount:    sharesCount,
			User:           UserInfo{UserID: userID, Nickname: nickname, AvatarURL: avatarURL},
			Location:       RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName},
			MyState:        &MyMomentState{IsLiked: isLiked, IsFavorited: isFavorited},
			Recommendation: &RecommendationContext{Type: "trending"},
		}
		items = append(items, momentCandidate{Item: item, Score: likesCount + commentsCount + sharesCount, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareMomentCandidates(items[i], items[j], "all") < 0 })
	out := make([]MomentItem, 0, len(items))
	for _, item := range items {
		out = append(out, item.Item)
	}
	return out, nil
}

func loadMomentTags(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	result := make(map[string][]string, len(momentIDs))
	rows, err := queryable.Query(ctx, queryFeedMomentTagsByMomentIDs, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, tag string
		if err := rows.Scan(&momentID, &tag); err != nil {
			return nil, err
		}
		result[momentID] = append(result[momentID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func paginateUsers(items []userCandidate, limit int, cursor string) (PaginatedUsers, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedUsers{}, err
	}
	start := findUserCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedUsers{Items: make([]UserItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeFeedCursor(itemCursor{Distance: items[end-1].Distance, ID: items[end-1].Item.User.UserID})
	}
	return resp, nil
}

func paginateGroups(items []groupCandidate, limit int, cursor string) (PaginatedGroups, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedGroups{}, err
	}
	start := findGroupCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedGroups{Items: make([]GroupItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeFeedCursor(itemCursor{Distance: items[end-1].Distance, ID: items[end-1].Item.Group.GroupID})
	}
	return resp, nil
}

func paginateMoments(items []momentCandidate, limit int, cursor string, source string) (PaginatedMoments, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	start := findMomentCursorStart(items, decoded, source)
	end := min(len(items), start+limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		last := items[end-1]
		resp.NextCursor = encodeFeedCursor(itemCursor{Score: last.Score, Distance: last.Distance, CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339), ID: last.Item.MomentID})
	}
	return resp, nil
}

func compareUserCandidates(left, right userCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Item.SocialContext != nil && right.Item.SocialContext != nil && left.Item.SocialContext.MutualFriendsCount != right.Item.SocialContext.MutualFriendsCount {
		return right.Item.SocialContext.MutualFriendsCount - left.Item.SocialContext.MutualFriendsCount
	}
	if left.Item.User.UserID > right.Item.User.UserID {
		return -1
	}
	if left.Item.User.UserID < right.Item.User.UserID {
		return 1
	}
	return 0
}

func compareGroupCandidates(left, right groupCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Item.FriendsInGroup != right.Item.FriendsInGroup {
		return right.Item.FriendsInGroup - left.Item.FriendsInGroup
	}
	if left.Item.Group.GroupID > right.Item.Group.GroupID {
		return -1
	}
	if left.Item.Group.GroupID < right.Item.Group.GroupID {
		return 1
	}
	return 0
}

func compareMomentCandidates(left, right momentCandidate, source string) int {
	if source == "following" {
		if !left.CreatedAt.Equal(right.CreatedAt) {
			if left.CreatedAt.After(right.CreatedAt) {
				return -1
			}
			return 1
		}
		if left.Item.MomentID > right.Item.MomentID {
			return -1
		}
		if left.Item.MomentID < right.Item.MomentID {
			return 1
		}
		return 0
	}
	if left.Score != right.Score {
		return right.Score - left.Score
	}
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.After(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.Item.MomentID > right.Item.MomentID {
		return -1
	}
	if left.Item.MomentID < right.Item.MomentID {
		return 1
	}
	return 0
}

func findUserCursorStart(items []userCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	for idx, item := range items {
		cmp := compareUserCandidates(item, userCandidate{Item: UserItem{User: UserInfo{UserID: cursor.ID}}, Distance: cursor.Distance})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findGroupCursorStart(items []groupCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	for idx, item := range items {
		cmp := compareGroupCandidates(item, groupCandidate{Item: GroupItem{Group: GroupInfo{GroupID: cursor.ID}}, Distance: cursor.Distance})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findMomentCursorStart(items []momentCandidate, cursor *itemCursor, source string) int {
	if cursor == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339, cursor.CreatedAt)
	for idx, item := range items {
		cmp := compareMomentCandidates(item, momentCandidate{Item: MomentItem{MomentID: cursor.ID}, Score: cursor.Score, Distance: cursor.Distance, CreatedAt: parsedAt.UTC()}, source)
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func encodeFeedCursor(cursor itemCursor) string {
	block, err := newFeedCursorBlock()
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	payload, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return ""
	}
	sealed := gcm.Seal(nonce, nonce, payload, nil)
	return base64.RawURLEncoding.EncodeToString(sealed)
}

func decodeFeedCursor(cursor string) (*itemCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	block, err := newFeedCursorBlock()
	if err != nil {
		return nil, ErrInvalidCursor
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || len(blob) < gcm.NonceSize() {
		return nil, ErrInvalidCursor
	}
	nonce, ciphertext := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	payload, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var decoded itemCursor
	if err := json.Unmarshal(payload, &decoded); err != nil || decoded.ID == "" {
		return nil, ErrInvalidCursor
	}
	if decoded.CreatedAt != "" {
		if _, err := time.Parse(time.RFC3339, decoded.CreatedAt); err != nil {
			return nil, ErrInvalidCursor
		}
	}
	return &decoded, nil
}

func newFeedCursorBlock() (cipher.Block, error) {
	secret := config.Load().AuthTokenSecret
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	key := sha256.Sum256([]byte(secret))
	return aes.NewCipher(key[:])
}

var _ Repository = (*PostgresRepository)(nil)
