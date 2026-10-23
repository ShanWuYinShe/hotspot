package moments

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// ── users 模块「按作者聚合的瞬间列表」读能力（SQL 与 scan helper 原样迁自
// users/repo_postgres_lists.go；归位本模块后 users 经 Service 接口消费，
// 不再直查 moments/reactions 表）──

var momentsByUserCursorCodec = cursor.NewCodec("moments-by-user")

type momentsByUserCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

func encodeMomentsByUserCursor(at, id string) string {
	token, err := momentsByUserCursorCodec.Encode(momentsByUserCursor{At: at, ID: id})
	if err != nil {
		return ""
	}
	return token
}

func decodeMomentsByUserCursor(raw string) (time.Time, string, error) {
	if raw == "" {
		return time.Time{}, "", nil
	}
	var decoded momentsByUserCursor
	if err := momentsByUserCursorCodec.Decode(raw, &decoded); err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	if decoded.At == "" || decoded.ID == "" {
		return time.Time{}, "", ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return parsed.UTC(), decoded.ID, nil
}

// ListMomentsByAuthor returns a user's non-deleted moments in created-order
// desc, keyset-paginated. includePrivate=false restricts to public moments
// (public profile view); true is reserved for the author's own view.
func (r *PostgresRepository) ListMomentsByAuthor(ctx context.Context, userID string, limit int, cursor string, includePrivate bool) (PaginatedMoments, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = types.NormalizeLimit(limit)
	cursorAt, cursorMomentID, err := decodeMomentsByUserCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, ErrInvalidCursor
	}
	var cursorAtValue any
	if !cursorAt.IsZero() {
		cursorAtValue = cursorAt
	}
	rows, err := queryable.Query(ctx, queryListMomentsByAuthor, userID, includePrivate, cursorAtValue, cursorMomentID, limit+1)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, err := scanUserMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		parsed, _ := time.Parse(time.RFC3339Nano, item.CreatedAt)
		cursors = append(cursors, encodeMomentsByUserCursor(parsed.UTC().Format(time.RFC3339), item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			c := cursors[limit-1]
			return PaginatedMoments{Items: items[:limit], NextCursor: &c}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
	}
	return PaginatedMoments{Items: items}, nil
}

// ListReactedByUser returns a user's reacted moments in reacted-order desc,
// keyset-paginated. viewerUserID is empty for anonymous callers; non-empty it
// unlocks the author's own private moments in the result (spec api_spec_moment.md
// → moments/list_reacted).
func (r *PostgresRepository) ListReactedByUser(ctx context.Context, userID, viewerUserID string, limit int, cursor string) (PaginatedMoments, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = types.NormalizeLimit(limit)
	cursorReactedAt, cursorMomentID, err := decodeMomentsByUserCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, ErrInvalidCursor
	}
	var cursorReactedAtValue any
	if !cursorReactedAt.IsZero() {
		cursorReactedAtValue = cursorReactedAt
	}
	rows, err := queryable.Query(ctx, queryListReactedByUser, userID, viewerUserID, cursorReactedAtValue, cursorMomentID, limit+1)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		// is_favorited 由 SQL 尾列 EXISTS(moment_favorites) 按查看者实查（spec
		// api_spec_types.md MomentItem：当前用户是否收藏）；favorited_at 仅
		// fetch_favorites 的 FavoritedMomentItem 语义，本端点不填充。
		item, reactedAt, err := scanUserReactedMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		// 游标携带 reacted_at 全精度（RFC3339Nano）：列由 time.Now().UTC() 全精度
		// 写入且 upsert 切换 emoji 会刷新，秒截断会让同秒行在翻页中静默丢失
		// （AGENTS 分页纪律，fetch_favorites 修复先例）。
		cursors = append(cursors, encodeMomentsByUserCursor(reactedAt.UTC().Format(time.RFC3339Nano), item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			c := cursors[limit-1]
			return PaginatedMoments{Items: items[:limit], NextCursor: &c}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
	}
	return PaginatedMoments{Items: items}, nil
}

func scanUserMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	var item MomentItem
	var userID, nickname, locationName, privacyLevel string
	var latitude, longitude float64
	var createdAt time.Time
	if err := scanner.Scan(&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel); err != nil {
		return MomentItem{}, err
	}
	if nickname == "" {
		nickname = userID
	}
	item.User = UserInfo{UserID: userID, Nickname: nickname}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.Location = &Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, nil
}

// scanUserReactedMomentRow 扫描 queryListReactedByUser 行（含尾列 is_favorited），
// 返回全精度 reactedAt（time.Time，游标编码用，不得用秒精度展示格式）。
func scanUserReactedMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, time.Time, error) {
	var item MomentItem
	var userID, nickname, locationName, privacyLevel string
	var latitude, longitude float64
	var createdAt, reactedAt time.Time
	if err := scanner.Scan(&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &reactedAt, &item.IsFavorited); err != nil {
		return MomentItem{}, time.Time{}, err
	}
	if nickname == "" {
		nickname = userID
	}
	item.User = UserInfo{UserID: userID, Nickname: nickname}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.Location = &Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, reactedAt.UTC(), nil
}
