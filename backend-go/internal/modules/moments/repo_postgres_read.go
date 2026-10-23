package moments

import (
	"context"
	"errors"
	"fmt"
	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return MomentItem{}, false, persistence.ErrNotImplemented
	}
	// 单次往返取齐基础列 + tags + mentions（审查 P2-16）；此前 3 次往返。
	item, err := scanMomentRowWithAssociations(queryable.QueryRow(ctx, queryGetMoment, momentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MomentItem{}, false, nil
		}
		return MomentItem{}, false, fmt.Errorf("moments.Get scan: %w", err)
	}
	return item, true, nil
}

func (r *PostgresRepository) ListMine(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
		return PaginatedMoments{}, fmt.Errorf("moments.ListMine query: %w", err)
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		// ListMine 直接走 scanMomentRowParts 取全精度 createdAt：游标谓词是
		// (created_at, moment_id) < cursor，created_at 列以 time.Now() 全精度
		// 写入，若游标截断到秒（item.CreatedAt 的展示格式），同秒多条瞬间翻页
		// 时谓词恒假、行被静默丢失（先例见 scanFavoriteMomentRow 注释）。
		item, createdAtFull, _, err := scanMomentRowParts(rows, false)
		if err != nil {
			return PaginatedMoments{}, fmt.Errorf("moments.ListMine scan: %w", err)
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		cursors = append(cursors, encodeMomentMineCursor(createdAtFull.UTC().Format(time.RFC3339Nano), item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, fmt.Errorf("moments.ListMine tags: %w", err)
			}
			mentionsByMomentID, err := loadMomentMentions(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, fmt.Errorf("moments.ListMine mentions: %w", err)
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
				items[i].Mentions = mentionsByMomentID[items[i].MomentID]
			}
			return PaginatedMoments{Items: items[:limit], NextCursor: nextCursorOrNil(cursors[limit-1])}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, fmt.Errorf("moments.ListMine rows: %w", err)
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, fmt.Errorf("moments.ListMine tags: %w", err)
	}
	mentionsByMomentID, err := loadMomentMentions(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, fmt.Errorf("moments.ListMine mentions: %w", err)
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
		items[i].Mentions = mentionsByMomentID[items[i].MomentID]
	}
	return PaginatedMoments{Items: items}, nil
}

const momentMineCursorNamespace = "moments.mine"

// nextCursorOrNil 收口列表分页的翻页游标下发：spec 规定 next_cursor 恒定
// nullable，游标编码失败若下发空串，客户端透传回来会被当作首页重新解码，
// 存在整页重复拉取风险——失败时降级为 nil 并告警。
func nextCursorOrNil(cursorValue string) *string {
	if cursorValue == "" {
		observability.L().Warn("moments: cursor encode failed, next_cursor downgraded to null")
		return nil
	}
	return &cursorValue
}

func encodeMomentMineCursor(createdAt, momentID string) string {
	value, err := cursor.Encode(momentMineCursorNamespace, momentInteractionCursor{At: createdAt, ID: momentID})
	if err != nil {
		return ""
	}
	return value
}

func decodeMomentMineCursor(cursorValue string) (any, string, error) {
	if cursorValue == "" {
		return nil, "", nil
	}
	var decoded momentInteractionCursor
	if err := cursor.Decode(momentMineCursorNamespace, cursorValue, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	parsedAt, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return parsedAt.UTC(), decoded.ID, nil
}

func (r *PostgresRepository) LoadMyState(ctx context.Context, momentID, userID string) *MyMomentState {
	if userID == "" {
		return nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil
	}
	var favorited int
	isFavorited := false
	// ErrNoRows=未收藏是正常分支；其余错误（连接失败等）记录日志后按未收藏
	// 返回——签名 *MyMomentState 无 error 出口，吞掉真实错误会掩盖故障。
	err := queryable.QueryRow(ctx, queryCheckMomentFavorite, userID, momentID).Scan(&favorited)
	switch {
	case err == nil:
		isFavorited = true
	case errors.Is(err, pgx.ErrNoRows):
	default:
		observability.L().Error("moments.load_my_state favorite check failed", "error", err, "moment_id", momentID, "user_id", userID)
	}
	return &MyMomentState{IsFavorited: isFavorited}
}

func scanMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	item, _, _, err := scanMomentRowParts(scanner, false)
	return item, err
}

// scanSearchMomentRow 扫描 moments/search 查询行：基础列 + 尾部 is_favorited。
// 登录查看者时 buildSearchMomentsQuery 在 SELECT 尾部追加
// EXISTS(moment_favorites) 列（spec api_spec_types.md MomentItem 查看者布尔恒定），
// 匿名查询不产生该列、不走本函数。
func scanSearchMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	item, _, isFavorited, err := scanMomentRowParts(scanner, true)
	if err != nil {
		return MomentItem{}, err
	}
	item.IsFavorited = isFavorited
	return item, nil
}

// scanMomentRowParts 扫描 moment 基础列；额外返回全精度 createdAt（time.Time，
// 列写入精度），供时间 keyset 游标以 RFC3339Nano 编码——item.CreatedAt 是
// 截断到秒的展示格式，不得用于游标（秒截断丢同秒行，见 ListMine 注释）。
// withViewerState 时额外扫描尾部 is_favorited 布尔并原样返回（不写入 item，
// 由调用方决定用途）。
func scanMomentRowParts(scanner interface{ Scan(...any) error }, withViewerState bool) (MomentItem, time.Time, bool, error) {
	var item MomentItem
	var userID, nickname, locationName, privacyLevel string
	var createdAt time.Time
	var latitude, longitude float64
	var isFavorited bool
	dest := []any{&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &item.IsDeleted}
	if withViewerState {
		dest = append(dest, &isFavorited)
	}
	if err := scanner.Scan(dest...); err != nil {
		return MomentItem{}, time.Time{}, false, fmt.Errorf("moments.scanMomentRow scan: %w", err)
	}
	item = momentItemFromRow(item, userID, nickname, createdAt, latitude, longitude, locationName, privacyLevel)
	return item, createdAt.UTC(), isFavorited, nil
}

// momentItemFromRow 把已扫描的基础列填充为 MomentItem（scanMomentRowParts 与
// scanMomentRowWithAssociations 共用，避免两处展示字段口径漂移）。
func momentItemFromRow(item MomentItem, userID, nickname string, createdAt time.Time, latitude, longitude float64, locationName, privacyLevel string) MomentItem {
	// nickname 已通过 SQL 的 COALESCE(u.nickname, '') 兜底
	// 若 SQL 未提供 nickname（兼容性回退），降级为 userID
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
	return item
}

// scanMomentRowWithAssociations 扫描 queryGetMoment 的行：基础列 + tags +
// mentions（两个 LATERAL 的 ARRAY_AGG 列）。对应详情单条读取路径。
func scanMomentRowWithAssociations(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	var item MomentItem
	var userID, nickname, locationName, privacyLevel string
	var createdAt time.Time
	var latitude, longitude float64
	var tags, mentions []string
	dest := []any{&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &item.IsDeleted, &tags, &mentions}
	if err := scanner.Scan(dest...); err != nil {
		return MomentItem{}, fmt.Errorf("moments.scanMomentRowWithAssociations scan: %w", err)
	}
	item = momentItemFromRow(item, userID, nickname, createdAt, latitude, longitude, locationName, privacyLevel)
	item.Tags = tags
	item.Mentions = mentions
	return item, nil
}

func loadMomentTags(ctx context.Context, queryable persistence.Queryable, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	tagsByMomentID := make(map[string][]string, len(momentIDs))
	rows, err := queryable.Query(ctx, queryListMomentTagsByMomentIDs, momentIDs)
	if err != nil {
		return nil, fmt.Errorf("moments.loadMomentTags query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, tag string
		if err := rows.Scan(&momentID, &tag); err != nil {
			return nil, fmt.Errorf("moments.loadMomentTags scan: %w", err)
		}
		tagsByMomentID[momentID] = append(tagsByMomentID[momentID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moments.loadMomentTags rows: %w", err)
	}
	return tagsByMomentID, nil
}

func loadMomentMentions(ctx context.Context, queryable persistence.Queryable, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	mentionsByMomentID := make(map[string][]string, len(momentIDs))
	rows, err := queryable.Query(ctx, queryListMomentMentionsByMomentIDs, momentIDs)
	if err != nil {
		return nil, fmt.Errorf("moments.loadMomentMentions query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, mentionedUserID string
		if err := rows.Scan(&momentID, &mentionedUserID); err != nil {
			return nil, fmt.Errorf("moments.loadMomentMentions scan: %w", err)
		}
		mentionsByMomentID[momentID] = append(mentionsByMomentID[momentID], mentionedUserID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moments.loadMomentMentions rows: %w", err)
	}
	return mentionsByMomentID, nil
}

// scanFavoriteMomentRow 返回展现实体与全精度收藏时刻：FavoritedAt 对外展示
// 截断到秒（spec 时间格式），游标必须携带纳秒全精度——favorited_at 列以
// time.Now().UTC() 全精度写入，若游标截断到秒，同秒多条收藏翻页时会因
// keyset 谓词 (favorited_at < cursor) 恒假而静默丢失（秒截断丢数据）。
func scanFavoriteMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, time.Time, error) {
	var item MomentItem
	var userID, nickname, locationName, privacyLevel string
	var createdAt, favoritedAt time.Time
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &item.IsDeleted, &favoritedAt); err != nil {
		return MomentItem{}, time.Time{}, fmt.Errorf("moments.scanFavoriteMomentRow scan: %w", err)
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
	return item, favoritedAt.UTC(), nil
}
