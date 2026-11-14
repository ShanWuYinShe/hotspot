package users

import (
	"context"
	"fmt"
	"strings"

	"hotspot/backend-go/internal/platform/persistence"
)

// searchProfilesRanked 是大用户量密集关键词的搜索路径：匹配（PGroonga）+
// relevance 评分 + 排序 + keyset 分页全部在一条 SQL 内完成，返回页即最终页，
// 不再受 searchCandidateCap=500 的「最新候选」截断。游标仍复用既有
// searchCursor（opaque JSON），新增 relevance_score 字段。
func (r *PostgresRepository) searchProfilesRanked(ctx context.Context, queryable persistence.Queryable, req SearchUsersRequest, keyword string) (PaginatedProfiles, error) {
	sortBy := normalizeSearchSort(req.SortBy)
	limit, rawCursor := normalizePagination(req.Pagination)
	cursor := searchCursor{}
	hasCursor := false
	if rawCursor != "" {
		decoded, err := decodeSearchCursor(rawCursor)
		if err != nil {
			return PaginatedProfiles{InvalidCursor: true}, nil
		}
		if normalizeSearchSort(decoded.SortBy) != sortBy {
			return PaginatedProfiles{InvalidCursor: true}, nil
		}
		if sortBy == "relevance" && decoded.RelevanceScore <= 0 {
			// 旧候选路径产生的游标缺 score，无法定位新全序；明确拒绝而非错页。
			return PaginatedProfiles{InvalidCursor: true}, nil
		}
		if sortBy == "newest" && decoded.CreatedAt == "" {
			return PaginatedProfiles{InvalidCursor: true}, nil
		}
		cursor = decoded
		hasCursor = true
	}
	tag := strings.ToLower(strings.TrimSpace(req.Tag))
	rawKeyword := strings.TrimSpace(req.Keyword)
	sql, args := buildRankedSearchUsersQuery(rawKeyword, tag, sortBy, cursor, hasCursor, limit+1)
	rows, err := queryable.Query(ctx, sql, args...)
	if err != nil {
		return PaginatedProfiles{}, err
	}
	defer rows.Close()
	scanned := make([]MyProfile, 0, limit+1)
	settingsByUserID := make(map[string]UserSettings, limit+1)
	for rows.Next() {
		profile, settings, scanErr := scanSearchProfileRow(rows, true)
		if scanErr != nil {
			return PaginatedProfiles{}, scanErr
		}
		settingsByUserID[profile.UserID] = settings
		scanned = append(scanned, profile)
	}
	if err := rows.Err(); err != nil {
		return PaginatedProfiles{}, err
	}
	hasNext := len(scanned) > limit
	if hasNext {
		scanned = scanned[:limit]
	}
	result := PaginatedProfiles{Items: make([]MyProfile, 0, len(scanned)), Settings: settingsByUserID}
	result.Items = append(result.Items, scanned...)
	if hasNext && len(scanned) > 0 {
		last := scanned[len(scanned)-1]
		result.NextCursor = encodeSearchCursor(searchProfileCandidate{Profile: last, SortBy: sortBy, RelevanceScore: searchRelevanceScore(last, keyword, tag)})
	}
	return result, nil
}

func buildRankedSearchUsersQuery(rawKeyword, tag, sortBy string, cursor searchCursor, hasCursor bool, limit int) (string, []any) {
	args := make([]any, 0, 8)
	// 内层只排 (user_id, time, score) 小行：密集关键词命中 10 万行时，排序宽度
	// 从 80+ 字节/行降到 ~25 字节/行，避免 external merge sort 撑爆 work_mem；
	// 外层只 JOIN 页内 ≤MaxPageSize 行的完整画像 + 标签。
	var inner strings.Builder
	inner.WriteString("SELECT u.user_id, u.last_online, u.created_at, ")
	args = append(args, rawKeyword)
	kwArg := len(args)
	scoreExpr := rankedRelevanceScoreSQL(kwArg, tag != "")
	inner.WriteString("(" + scoreExpr + ") AS search_score FROM users u JOIN user_settings us ON us.user_id = u.user_id WHERE us.searchable = TRUE AND u.is_deleted = FALSE AND us.privacy_level = 'public'")
	inner.WriteString(fmt.Sprintf(" AND (u.nickname &@ $%d OR u.bio &@ $%d", kwArg, kwArg))
	// 手机号格式 keyword 需同时按登录手机号精确匹配（spec user.md → users/search
	// keyword 说明）：手机号真值在 auth_accounts.login_id（users.phone 只是回填），
	// 与候选路径 buildSearchUsersQuery 同口径，受 us.phone_searchable 控制。
	// 手机号命中本身不计相关性分（评分/排序口径不变，仅决定是否进入候选集）。
	if isPhoneFormatKeyword(rawKeyword) {
		args = append(args, rawKeyword)
		inner.WriteString(fmt.Sprintf(" OR (us.phone_searchable = TRUE AND EXISTS (SELECT 1 FROM auth_accounts aa WHERE aa.user_id = u.user_id AND aa.login_id = $%d))", len(args)))
	}
	inner.WriteString(")")
	if tag != "" {
		args = append(args, tag)
		inner.WriteString(fmt.Sprintf(" AND EXISTS (SELECT 1 FROM user_profile_tags upt WHERE upt.user_id = u.user_id AND lower(upt.tag_name) = lower($%d))", len(args)))
	}
	if hasCursor {
		switch sortBy {
		case "newest":
			args = append(args, cursor.CreatedAt, cursor.UserID)
			createdArg, idArg := len(args)-1, len(args)
			inner.WriteString(fmt.Sprintf(" AND (date_trunc('second', u.created_at) < $%d::timestamptz OR (date_trunc('second', u.created_at) = $%d::timestamptz AND u.user_id > $%d))", createdArg, createdArg, idArg))
		default:
			args = append(args, cursor.RelevanceScore, cursor.LastOnline, cursor.UserID)
			scoreArg, lastArg, idArg := len(args)-2, len(args)-1, len(args)
			inner.WriteString(fmt.Sprintf(" AND (((%s) < $%d) OR ((%s) = $%d AND date_trunc('second', u.last_online) < $%d::timestamptz) OR ((%s) = $%d AND date_trunc('second', u.last_online) = $%d::timestamptz AND u.user_id > $%d))",
				scoreExpr, scoreArg, scoreExpr, scoreArg, lastArg, scoreExpr, scoreArg, lastArg, idArg))
		}
	}
	switch sortBy {
	case "newest":
		inner.WriteString(" ORDER BY date_trunc('second', u.created_at) DESC, u.user_id ASC")
	default:
		inner.WriteString(" ORDER BY search_score DESC, date_trunc('second', u.last_online) DESC, u.user_id ASC")
	}
	args = append(args, limit)
	inner.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))

	var outer strings.Builder
	outer.WriteString("SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), u.user_type, COALESCE(u.bio, ''), COALESCE(u.gender, ''), COALESCE(u.email, ''), COALESCE(u.phone, ''), u.location_name, u.latitude, u.longitude, u.presence_status, u.last_online, u.created_at, u.is_deleted, COALESCE((SELECT array_agg(up.tag_name ORDER BY up.tag_name) FROM user_profile_tags up WHERE up.user_id = u.user_id), ARRAY[]::text[]), us.presence_visible, us.location_visible, us.stats_visible FROM (")
	outer.WriteString(inner.String())
	outer.WriteString(") ranked JOIN users u ON u.user_id = ranked.user_id JOIN user_settings us ON us.user_id = ranked.user_id ORDER BY ")
	switch sortBy {
	case "newest":
		outer.WriteString("date_trunc('second', ranked.created_at) DESC, ranked.user_id ASC")
	default:
		outer.WriteString("ranked.search_score DESC, date_trunc('second', ranked.last_online) DESC, ranked.user_id ASC")
	}
	return outer.String(), args
}

// rankedRelevanceScoreSQL 与 Go 侧 searchRelevanceScore 严格同分：昵称精确 6、
// 前缀 4、包含 3、bio 包含 2、tag 过滤命中固定 3；SQL 用 lower + position，
// 避免 ILIKE 通配符转义，同时与列上表达式口径一致。
func rankedRelevanceScoreSQL(keywordArg int, hasTag bool) string {
	score := fmt.Sprintf("(CASE WHEN lower(u.nickname) = lower($%d) THEN 6 WHEN left(lower(u.nickname), char_length($%d)) = lower($%d) THEN 4 WHEN position(lower($%d) in lower(u.nickname)) > 0 THEN 3 ELSE 0 END + CASE WHEN position(lower($%d) in lower(COALESCE(u.bio, ''))) > 0 THEN 2 ELSE 0 END)",
		keywordArg, keywordArg, keywordArg, keywordArg, keywordArg)
	if hasTag {
		return "(" + score + " + 3)"
	}
	return score
}
