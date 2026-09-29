package users

import (
	"context"
	"errors"
	"fmt"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return MyProfile{}, false, persistence.ErrNotImplemented
	}
	profile, err := scanUserProfileRow(queryable.QueryRow(ctx, queryGetUserProfile, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyProfile{}, false, nil
		}
		return MyProfile{}, false, err
	}
	stats, err := countUserStats(ctx, queryable, userID)
	if err != nil {
		return MyProfile{}, false, err
	}
	profile.Stats = stats
	// 画像标签独立表：与批量/搜索路径一致在此回填，否则 get_account 等单查
	// 永远返回空 tags（线上实测：保存成功，编辑页重载即丢失）。
	tags, err := loadUserProfileTags(ctx, queryable, userID)
	if err != nil {
		return MyProfile{}, false, err
	}
	profile.Tags = tags
	return profile, true, nil
}

// GetProfiles 批量取画像（一次往返；不含 Stats，调用方按需另取）。
func (r *PostgresRepository) GetProfiles(ctx context.Context, userIDs []string) (map[string]MyProfile, error) {
	result := make(map[string]MyProfile, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryGetUserProfiles, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scanned := make([]MyProfile, 0, len(userIDs))
	for rows.Next() {
		profile, err := scanUserProfileRow(rows)
		if err != nil {
			return nil, err
		}
		scanned = append(scanned, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 画像标签独立表：单查(GetProfile)与搜索路径一致在此回填。
	ids := make([]string, 0, len(scanned))
	for _, profile := range scanned {
		ids = append(ids, profile.UserID)
	}
	tagsByUserID, err := loadUserProfileTagsBatch(ctx, queryable, ids)
	if err != nil {
		return nil, err
	}
	for _, profile := range scanned {
		profile.Tags = tagsByUserID[profile.UserID]
		result[profile.UserID] = profile
	}
	return result, nil
}

func (r *PostgresRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return UserSettings{}, false, persistence.ErrNotImplemented
	}
	var settings UserSettings
	var allowPrivateChat string
	if err := queryable.QueryRow(ctx, queryGetUserSettings, userID).Scan(&settings.PrivacyLevel, &settings.LocationVisible, &settings.PresenceVisible, &settings.SocialListsVisible, &settings.GroupsVisible, &settings.ReactedMomentsVisible, &settings.StatsVisible, &settings.AllowFollow, &allowPrivateChat, &settings.Searchable, &settings.PhoneSearchable, &settings.NicknameSearchable, &settings.LocationSearchable, &settings.NearbyMomentNotify); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserSettings{}, false, nil
		}
		return UserSettings{}, false, err
	}
	settings.AllowPrivateChat = ChatPerm(allowPrivateChat)
	return settings, true, nil
}

func (r *PostgresRepository) GetSettingsBatch(ctx context.Context, userIDs []string) (map[string]UserSettings, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	result := make(map[string]UserSettings, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := queryable.Query(ctx, queryGetUserSettingsBatch, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var settings UserSettings
		var allowPrivateChat string
		if err := rows.Scan(&userID, &settings.PrivacyLevel, &settings.LocationVisible, &settings.PresenceVisible, &settings.SocialListsVisible, &settings.GroupsVisible, &settings.ReactedMomentsVisible, &settings.StatsVisible, &settings.AllowFollow, &allowPrivateChat, &settings.Searchable, &settings.PhoneSearchable, &settings.NicknameSearchable, &settings.LocationSearchable, &settings.NearbyMomentNotify); err != nil {
			return nil, err
		}
		settings.AllowPrivateChat = ChatPerm(allowPrivateChat)
		result[userID] = settings
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) (PaginatedProfiles, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedProfiles{}, persistence.ErrNotImplemented
	}
	// radius 语义（spec user.md → users/search）：显式提供才按半径过滤；缺省不过滤（全球匹配）
	radiusFilter := req.Radius != nil && *req.Radius > 0
	radius := normalizeSearchRadius(req.Radius)
	refPt := geo.Point{Latitude: req.ReferencePoint.Latitude, Longitude: req.ReferencePoint.Longitude}
	var bounds geo.Bounds
	if radiusFilter {
		bounds = geo.BoundsAround(refPt, radius)
	}
	sql, args := buildSearchUsersQuery(bounds, req.Keyword, strings.ToLower(strings.TrimSpace(req.Tag)), radiusFilter)
	if sql == "" {
		return PaginatedProfiles{}, nil
	}
	rows, err := queryable.Query(ctx, sql, args...)
	if err != nil {
		return PaginatedProfiles{}, err
	}
	defer rows.Close()
	candidates := make([]searchProfileCandidate, 0)
	sortBy := normalizeSearchSort(req.SortBy)
	plan := geo.NewSearchPlan(refPt, radius, sortBy)
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	tag := strings.ToLower(strings.TrimSpace(req.Tag))
	// Scan every candidate row first, then load profile tags for the whole
	// page in a single query (avoids an N+1 of loadUserProfileTags calls
	// inside the loop).
	scanned := make([]MyProfile, 0)
	for rows.Next() {
		profile, err := scanUserProfileRow(rows)
		if err != nil {
			return PaginatedProfiles{}, err
		}
		scanned = append(scanned, profile)
	}
	if err := rows.Err(); err != nil {
		return PaginatedProfiles{}, err
	}
	if len(scanned) == 0 {
		return PaginatedProfiles{}, nil
	}
	userIDs := make([]string, 0, len(scanned))
	for _, profile := range scanned {
		userIDs = append(userIDs, profile.UserID)
	}
	tagsByUserID, err := loadUserProfileTagsBatch(ctx, queryable, userIDs)
	if err != nil {
		return PaginatedProfiles{}, err
	}
	for _, profile := range scanned {
		profile.Tags = tagsByUserID[profile.UserID]
		// SQL 已下推 tag EXISTS；此处的 Go 层比对保留为双保险（大小写归一）。
		if tag != "" && !containsFold(profile.Tags, tag) {
			continue
		}
		distance, bearing, within := plan.Match(geo.Point{Latitude: profile.Location.Point.Latitude, Longitude: profile.Location.Point.Longitude})
		if radiusFilter && !within {
			continue
		}
		candidates = append(candidates, searchProfileCandidate{Profile: profile, Distance: distance, BearingDegrees: bearing, SortBy: sortBy, RelevanceScore: searchRelevanceScore(profile, keyword, tag)})
	}
	sortSearchCandidates(candidates, normalizeSearchSort(req.SortBy))
	return paginateSearchCandidates(candidates, req.Pagination), nil
}

// searchCandidateCap 限制单次搜索加载的候选行数：此前「只传 tag」时 SQL 仅剩
// searchable+public 谓词，等于全表扫描 users⋈user_settings 全量物化，用户量
// 上来后单请求即可打爆内存。
const searchCandidateCap = 500

func buildSearchUsersQuery(bounds geo.Bounds, keyword, tag string, radiusFilter bool) (string, []any) {
	args := make([]any, 0, 4)
	var builder strings.Builder
	builder.WriteString(`SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), u.user_type, COALESCE(u.bio, ''), COALESCE(u.gender, ''), COALESCE(u.email, ''), COALESCE(u.phone, ''), u.location_name, u.latitude, u.longitude, u.presence_status, u.last_online, u.created_at, u.is_deleted FROM users u JOIN user_settings us ON us.user_id = u.user_id WHERE us.searchable = TRUE AND u.is_deleted = FALSE AND us.privacy_level = 'public'`)
	if radiusFilter {
		mins, maxs := geo.CoveringArrays(bounds)
		if len(mins) == 0 {
			return "", nil
		}
		// location_searchable 控制位置/半径维度（spec user.md → searchable 系列开关）
		args = append(args, mins, maxs)
		builder.WriteString(` AND us.location_searchable = TRUE AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi) WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)`)
	}
	if keyword != "" {
		args = append(args, keyword)
		kwArg := len(args)
		// keyword 为手机号格式（纯数字）时，对 phone_searchable=true 的用户做
		// 登录手机号精确匹配（spec user.md → users/search keyword 说明）。手机号
		// 真值在 auth 侧 auth_accounts.login_id（users.phone 由 account 回填但
		// 以 login_id 为权威），经子查询跨表精确比对，与现有可见性过滤一致。
		phoneArg := 0
		if isPhoneFormatKeyword(keyword) {
			args = append(args, keyword)
			phoneArg = len(args)
		}
		builder.WriteString(fmt.Sprintf(" AND ($%d = '' OR (us.nickname_searchable = TRUE AND u.nickname ILIKE '%%' || $%d || '%%') OR u.bio ILIKE '%%' || $%d || '%%'", kwArg, kwArg, kwArg))
		if phoneArg > 0 {
			builder.WriteString(fmt.Sprintf(" OR (us.phone_searchable = TRUE AND EXISTS (SELECT 1 FROM auth_accounts aa WHERE aa.user_id = u.user_id AND aa.login_id = $%d))", phoneArg))
		}
		builder.WriteString(")")
	}
	// tag 过滤下推 SQL（EXISTS user_profile_tags）：此前在 Go 层对全量候选逐个
	// 比对，tag-only 搜索退化为全表加载。
	if tag != "" {
		args = append(args, tag)
		builder.WriteString(fmt.Sprintf(` AND EXISTS (SELECT 1 FROM user_profile_tags upt WHERE upt.user_id = u.user_id AND lower(upt.tag_name) = lower($%d))`, len(args)))
	}
	args = append(args, searchCandidateCap)
	builder.WriteString(fmt.Sprintf(` ORDER BY u.created_at DESC, u.user_id DESC LIMIT $%d`, len(args)))
	return builder.String(), args
}

// isPhoneFormatKeyword 判定搜索关键字是否手机号格式（纯数字，spec
// api_spec_base.md login_id：手机号型=纯数字）。空串由调用方排除。
func isPhoneFormatKeyword(keyword string) bool {
	if keyword == "" {
		return false
	}
	for _, c := range keyword {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func scanUserProfileRow(scanner interface{ Scan(...any) error }) (MyProfile, error) {
	var profile MyProfile
	var avatarURL, bio, gender, email, phone string
	var locationName string
	var latitude, longitude float64
	var presenceStatus, userType string
	var lastOnline, createdAt time.Time
	if err := scanner.Scan(&profile.UserID, &profile.Nickname, &avatarURL, &userType, &bio, &gender, &email, &phone, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &createdAt, &profile.IsDeleted); err != nil {
		return MyProfile{}, err
	}
	profile.AvatarURL = avatarURL
	profile.UserType = UserType(userType)
	profile.Bio = bio
	profile.Gender = Gender(gender)
	profile.Email = email
	profile.Phone = phone
	profile.Location = Location{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	profile.PresenceStatus = UserPresenceStatus(presenceStatus)
	profile.LastOnline = lastOnline.UTC().Format(time.RFC3339)
	profile.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return profile, nil
}

func countUserStats(ctx context.Context, queryable persistence.Queryable, userID string) (UserStats, error) {
	var stats UserStats
	if err := queryable.QueryRow(ctx, queryCountUserStats, userID).Scan(&stats.MomentsCount, &stats.FriendsCount, &stats.FollowersCount, &stats.FollowingCount, &stats.GroupsCount); err != nil {
		return UserStats{}, err
	}
	return stats, nil
}

func loadUserProfileTags(ctx context.Context, queryable persistence.Queryable, userID string) ([]string, error) {
	rows, err := queryable.Query(ctx, queryListUserProfileTags, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

// loadUserProfileTagsBatch fetches profile tags for many user IDs in one
// query. It returns a userID -> []tags map; user IDs with no tags are absent
// from the map (callers should treat absence as an empty slice).
func loadUserProfileTagsBatch(ctx context.Context, queryable persistence.Queryable, userIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := queryable.Query(ctx, queryListUserProfileTagsBatch, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, tag string
		if err := rows.Scan(&userID, &tag); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
