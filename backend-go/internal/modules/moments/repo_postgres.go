package moments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/textutil"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return MomentItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	momentID, err := idgen.WithPrefix("moment_")
	if err != nil {
		return MomentItem{}, err
	}
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	// moment + tags + mentions + images 五步写包进同一事务：任一步失败
	// 整体回滚，避免「moment 已落库但接口报错」导致客户端重试产生重复。
	var allowedTags []string
	normalizedMentions := normalizeMomentMentions(req.Mentions)
	nickname := userID
	if err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := queryable(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryInsertMoment, momentID, userID, req.Description, createdAt, 0, 0, 0, req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, privacy, geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})); err != nil {
			return fmt.Errorf("moments.Create: insert moment: %w", err)
		}
		tags := normalizeMomentTags(req.Tags)
		// blocked tag 校验收口在 insertMomentTags 内（与 Update 同一口径）：
		// 含被屏蔽标签 → ErrBlockedTag（10400）整体回滚；顺带消除此前
		// filterAllowedMomentTags 在本事务内的重复执行。
		allowed, err := insertMomentTags(txCtx, txQueryable, momentID, tags, now)
		if err != nil {
			return fmt.Errorf("moments.Create: insert tags: %w", err)
		}
		allowedTags = allowed
		if err := replaceMomentMentions(txCtx, txQueryable, momentID, normalizedMentions, now); err != nil {
			return fmt.Errorf("moments.Create: insert mentions: %w", err)
		}
		if len(req.ImageFileIDs) > 0 {
			if err := insertMomentImages(txCtx, txQueryable, momentID, req.ImageFileIDs, now); err != nil {
				return fmt.Errorf("moments.Create: insert images: %w", err)
			}
		}
		n, err := lookupUserNickname(txCtx, txQueryable, userID)
		if err != nil {
			return err
		}
		if n != "" {
			nickname = n
		}
		return nil
	}); err != nil {
		return MomentItem{}, err
	}
	return MomentItem{
		MomentID:      momentID,
		Description:   req.Description,
		Tags:          allowedTags,
		Mentions:      normalizedMentions,
		CreatedAt:     createdAt,
		LikesCount:    0,
		CommentsCount: 0,
		SharesCount:   0,
		User:          UserInfo{UserID: userID, Nickname: nickname},
		Location:      req.Location,
		PrivacyLevel:  privacy,
		IsFavorited:   false,
	}, nil
}

func lookupUserNickname(ctx context.Context, queryable persistence.Queryable, userID string) (string, error) {
	var nickname string
	err := queryable.QueryRow(ctx, queryGetUserNickname, userID).Scan(&nickname)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return nickname, nil
}

func (r *PostgresRepository) IncrementCommentCount(ctx context.Context, momentID string) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryIncrementMomentCommentCount, momentID)
}

func (r *PostgresRepository) DecrementCommentCount(ctx context.Context, momentID string) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDecrementMomentCommentCount, momentID)
}

// AdjustReactionCount 按 delta 增减 reaction 计数冗余列（SQL 内 GREATEST 钳 0）。
func (r *PostgresRepository) AdjustReactionCount(ctx context.Context, momentID string, delta int) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryAdjustMomentReactionCount, delta, momentID)
}

func (r *PostgresRepository) IncrementSharesCount(ctx context.Context, momentID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryIncrementMomentShareCount, momentID)
}

func (r *PostgresRepository) Search(ctx context.Context, req SearchMomentsRequest, viewerID string) (PaginatedMoments, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
	}
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "relevance"
	}
	searchCursor, err := decodeMomentSearchCursor(paginationCursor(req.Pagination))
	if err != nil {
		return PaginatedMoments{}, err
	}
	// 衰减时钟在此取一次并作为 SQL 参数下传：排序与游标编码必须使用**同一个**
	// 时钟值。此前 SQL 用数据库 now()、Go 用进程时钟，两者跨 15 分钟桶边界
	// （或存在时钟偏差）时游标分值与实际排序分值不一致，popular 翻页会漏行/重行。
	decayClock := SearchDecayClock(time.Now().UTC())
	sql, args := buildSearchMomentsQuery(req.Keyword, req.Tag, sortBy, limit+1, searchCursor, viewerID, decayClock)
	if sql == "" {
		return PaginatedMoments{}, nil
	}
	rows, err := queryable.Query(ctx, sql, args...)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var item MomentItem
		if viewerID != "" {
			// 登录查看者：SQL 尾列 EXISTS(moment_favorites) → is_favorited
			//（spec api_spec_types.md MomentItem 查看者布尔恒定）。
			item, err = scanSearchMomentRow(rows)
		} else {
			item, err = scanMomentRow(rows)
		}
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
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
	}
	// SQL 已按最终排序返回 keyset 页；多取的第 limit+1 行仅用于探测是否有下一页。
	var nextCursor *string
	if len(items) > limit {
		keyword := ""
		if req.Keyword != nil {
			keyword = *req.Keyword
		}
		tag := ""
		if req.Tag != nil {
			tag = *req.Tag
		}
		// popular 模式：游标分值按 Go 侧同公式衰减计算（与 searchScoreSQL 逐项
		// 对应），时钟量化到同一衰减桶（SearchDecayClock）。游标编码本页最后一项
		// （排他 keyset：下一页谓词严格 <，从页尾项的后继开始）——编码下页首项
		// 会让严格 < 把它自身排除造成丢项（量化时钟前被实时衰减漂移偶然遮掩）。
		c := encodeMomentSearchCursor(sortBy, items[limit-1], keyword, tag, decayClock)
		nextCursor = &c
		items = items[:limit]
	}
	return PaginatedMoments{Items: items, NextCursor: nextCursor}, nil
}

func paginationCursor(p *PaginationParams) string {
	if p == nil {
		return ""
	}
	return p.Cursor
}

// buildSearchMomentsQuery：全球 keyword/tag 搜索（spec api_spec_moment.md moments/search：
// 搜索为全局匹配、不承担附近发现职责，无半径/位置条件；附近发现入口为
// moments/fetch_nearby）。
//
// 三种排序共用同一 keyset 元组 (score, created_at, moment_id)：
//   - newest:  score 恒为 0，元组退化为时间倒序
//   - popular: score = 互动热度 / 时间衰减因子（见 searchScoreSQL）
//   - relevance: score = keyword 命中 + tag 命中
//
// viewerID 非空时 SELECT 追加 EXISTS(moment_favorites) 尾列承载查看者 is_favorited
// （spec api_spec_types.md MomentItem；与 fetch_nearby/fetch_following 同模式）。
//
// SQL 直接输出最终序，游标按最终序继续，翻页不重不漏。
func buildSearchMomentsQuery(keyword *string, tag *string, sortBy string, limit int, after *momentSearchCursor, viewerID string, decayClock time.Time) (string, []any) {
	args := make([]any, 0, 8)
	var builder strings.Builder
	builder.WriteString(`SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted`)
	if viewerID != "" {
		// 查看者收藏态：参数化 EXISTS 子查询（$N 占位符），匿名时不追加该列。
		args = append(args, viewerID)
		builder.WriteString(fmt.Sprintf(", EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $%d)", len(args)))
	}
	builder.WriteString(` FROM moments m LEFT JOIN users u ON u.user_id = m.user_id WHERE m.privacy_level = 'public' AND m.is_deleted = FALSE`)
	if viewerID != "" {
		// 双向拉黑过滤（对齐 fetch_following/fetch_nearby 口径）：搜索是发现面，
		// 被拉黑/拉黑对方作者的瞬间不应出现在结果里。
		args = append(args, viewerID)
		v := len(args)
		builder.WriteString(fmt.Sprintf(
			" AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = $%d AND blk.target_user_id = m.user_id AND blk.is_blocked = TRUE)"+
				" AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = m.user_id AND blk.target_user_id = $%d AND blk.is_blocked = TRUE)",
			v, v))
	}
	kwArg := 0
	tagArg := 0
	if keyword != nil && *keyword != "" {
		// 转义 LIKE 通配符：用户输入的 %/_ 被当通配符会改变匹配语义
		//（与内存实现的精确子串语义不一致）。
		args = append(args, escapeLikePattern(*keyword))
		kwArg = len(args)
		builder.WriteString(fmt.Sprintf(" AND m.description ILIKE '%%' || $%d || '%%' ESCAPE '\\'", len(args)))
	}
	if tag != nil && *tag != "" {
		args = append(args, *tag)
		tagArg = len(args)
		builder.WriteString(fmt.Sprintf(" AND m.moment_id IN (SELECT moment_id FROM moment_tags WHERE tag_name = $%d)", len(args)))
	}

	// popular 的衰减时钟作为参数绑定（而非 SQL 内 now()）：保证同一次查询的
	// ORDER BY 与 keyset 谓词使用同一时钟，且与 Go 侧游标分值完全同源。
	clockArg := 0
	if normalizeMomentSearchSort(sortBy) == "popular" {
		args = append(args, decayClock.Unix())
		clockArg = len(args)
	}
	scoreExpr := searchScoreSQL(sortBy, kwArg, tagArg, clockArg)
	if after != nil {
		args = append(args, after.Score, after.At, after.ID)
		n := len(args)
		builder.WriteString(fmt.Sprintf(
			" AND (%s, m.created_at, m.moment_id) < ($%d::float8, $%d::timestamptz, $%d)",
			scoreExpr, n-2, n-1, n,
		))
	}
	builder.WriteString(fmt.Sprintf(" ORDER BY (%s) DESC, m.created_at DESC, m.moment_id DESC", scoreExpr))
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))
	return builder.String(), args
}

// searchScoreSQL returns the SQL expression for the selected sort score. It
// intentionally mirrors momentSearchScore so memory and Postgres paginate with
// the same tuple semantics.
func searchScoreSQL(sortBy string, kwArg, tagArg, clockArg int) string {
	switch normalizeMomentSearchSort(sortBy) {
	case "popular":
		// 热度排序含时间衰减因子（spec api_spec_moment.md moments/search 排序：
		// 「按互动热度排序（含时间衰减因子；具体衰减算法为服务端实现细节）」）。
		// 实现：score = (reaction 计数冗余列 + comments_count) / POWER(发布小时龄 + 2, 1.5)，
		// 小时龄钳制非负（防未来时间戳导致负底数幂报错）。
		//
		// 计数用 likes_count 冗余列（语义已复活为 reactions 总数，reactions 模块
		// set/unreact 成功后经 Service.AdjustReactionCount 增量维护，启动期
		// ensureReactionCountColumn 差异回填自愈），替代此前的每行 reactions
		// 聚合子查询——子查询让 cursor 谓词与 ORDER BY 各算一遍 O(候选集×聚合)。
		//
		// 衰减时钟由调用方（Search）量化到 15 分钟桶后经 $clockArg 传入（epoch
		// 秒），SQL 不再使用 now()：ORDER BY 与 keyset 谓词、Go 游标编码三者严格
		// 使用同一时钟值。跨桶翻页（15 分钟以上）仍可能有弱一致漂移，但幅度远
		// 小于实时衰减；同一次查询内（含游标谓词）绝对一致。
		// clockArg>0 由 buildSearchMomentsQuery 保证。
		return fmt.Sprintf("((m.likes_count + m.comments_count)::float8 / POWER(GREATEST((FLOOR($%d::float8 / 900.0) * 900.0 - EXTRACT(EPOCH FROM m.created_at)) / 3600.0, 0.0) + 2.0, 1.5))", clockArg)
	case "newest":
		// 0::float8：裸整数常量会被 Postgres 当作 ORDER BY 位置引用（42P10），
		// 必须带类型成为表达式。
		return "0::float8"
	default:
		parts := make([]string, 0, 2)
		if kwArg > 0 {
			parts = append(parts, fmt.Sprintf("CASE WHEN m.description ILIKE '%%' || $%d || '%%' THEN 1 ELSE 0 END", kwArg))
		}
		if tagArg > 0 {
			// tag 已由 IN 子查询过滤，命中即计 1 分。
			parts = append(parts, "1")
		}
		if len(parts) == 0 {
			// 无 keyword/tag 的 relevance 退化为常量分，理由同 newest 的 0::float8。
			return "0::float8"
		}
		return strings.Join(parts, " + ")
	}
}

func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdateMomentRequest) error {
	var locationName *string
	var lat, lng *float64
	var s2CellID *int64
	if req.Location != nil {
		locationName = &req.Location.LocationName
		// 坐标以 float64 原值参数化（与 Create 同口径）：此前先 %f 转字符串
		// 引入 ~0.11m 量化截断，落库坐标与按原始 float 计算的 s2_cell_id
		// 错位（编辑位置后索引与坐标不一致）。
		latValue, lngValue := req.Location.Point.Latitude, req.Location.Point.Longitude
		lat, lng = &latValue, &lngValue
		id := geo.LeafCellID(geo.Point{Latitude: latValue, Longitude: lngValue})
		s2CellID = &id
	}
	// update + tags/mentions/images 替换包进同一事务：中途失败不会留下
	// 「标签已删但未插回」的半更新状态。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		q, ok := queryable(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		// queryUpdateMoment 带 owner + 未软删条件并 RETURNING：非作者/已软删
		// （含 service 预检后的并发软删窗口）0 行命中 → ErrMomentNotFound（10300），
		// 不再把检查-使用窗口内的已删瞬间静默更新成功。
		var updatedID string
		if err := q.QueryRow(txCtx, queryUpdateMoment, req.MomentID, userID, req.Description, req.PrivacyLevel, locationName, lat, lng, s2CellID).Scan(&updatedID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMomentNotFound
			}
			return fmt.Errorf("moments.Update: update moment: %w", err)
		}
		if req.Tags != nil {
			if err := q.Exec(txCtx, queryDeleteMomentTagsByMomentID, req.MomentID); err != nil {
				return fmt.Errorf("moments.Update: delete tags: %w", err)
			}
			// blocked tag 校验收口在 insertMomentTags（与 Create 同口径）：
			// 含被屏蔽标签 → ErrBlockedTag → 整个事务回滚（10400）。
			if _, err := insertMomentTags(txCtx, q, req.MomentID, req.Tags, time.Now().UTC()); err != nil {
				return fmt.Errorf("moments.Update: insert tags: %w", err)
			}
		}
		if req.Mentions != nil {
			if err := replaceMomentMentions(txCtx, q, req.MomentID, req.Mentions, time.Now().UTC()); err != nil {
				return fmt.Errorf("moments.Update: replace mentions: %w", err)
			}
		}
		if req.ImageFileIDs != nil {
			if err := replaceMomentImages(txCtx, q, req.MomentID, req.ImageFileIDs, time.Now().UTC()); err != nil {
				return fmt.Errorf("moments.Update: replace images: %w", err)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) Delete(ctx context.Context, momentID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var deletedID string
	if err := queryable.QueryRow(ctx, queryDeleteMoment, momentID).Scan(&deletedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMomentNotFound
		}
		return fmt.Errorf("moments.Delete: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Favorite(ctx context.Context, userID string, momentID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryInsertMomentFavorite, userID, momentID, time.Now().UTC())
}

func (r *PostgresRepository) Unfavorite(ctx context.Context, userID string, momentID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteMomentFavorite, userID, momentID)
}

func (r *PostgresRepository) NotificationContext(ctx context.Context, momentID string) (MomentItem, bool, error) {
	return r.Get(ctx, momentID)
}

func (r *PostgresRepository) ListFavorites(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = normalizeMomentInteractionLimit(limit)
	cursorAt, cursorMomentID, err := decodeMomentFavoriteCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	rows, err := queryable.Query(ctx, queryListFavoriteMoments, userID, cursorAt, cursorMomentID, limit+1)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, favoritedAt, err := scanFavoriteMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		item.IsFavorited = true
		item.FavoritedAt = favoritedAt.UTC().Format(time.RFC3339) // spec moment.md → FavoritedMomentItem.favorited_at（展示截断到秒）
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		// 游标携带纳秒全精度（秒截断会丢同秒收藏，见 scanFavoriteMomentRow 注释）；
		// 解码侧 time.Parse(RFC3339Nano) 兼容秒/纳秒两种历史格式，老游标仍有效。
		cursors = append(cursors, encodeMomentFavoriteCursor(favoritedAt.UTC().Format(time.RFC3339Nano), item.MomentID))
		if len(items) == limit+1 {
			// 显式关闭 rows 再发嵌套查询（同 ListMine：tx/单连接下未关闭 rows 会 conn busy）。
			rows.Close()
			tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			return PaginatedMoments{Items: items[:limit], NextCursor: nextCursorOrNil(cursors[limit-1])}, nil
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

func (r *PostgresRepository) ListFavoriteUserIDs(ctx context.Context, momentID string) ([]string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListMomentFavoriteUserIDs, momentID)
	if err != nil {
		return nil, fmt.Errorf("moments.ListFavoriteUserIDs query: %w", err)
	}
	defer rows.Close()
	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("moments.ListFavoriteUserIDs scan: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moments.ListFavoriteUserIDs rows: %w", err)
	}
	return userIDs, nil
}

type momentInteractionCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

func normalizeMomentInteractionLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > types.MaxPageSize:
		return types.MaxPageSize
	default:
		return limit
	}
}

// momentFavoriteCursorNamespace 是收藏列表分页游标的加密命名空间。
const momentFavoriteCursorNamespace = "moments.favorites"

func encodeMomentFavoriteCursor(favoritedAt, momentID string) string {
	value, err := cursor.Encode(momentFavoriteCursorNamespace, momentInteractionCursor{At: favoritedAt, ID: momentID})
	if err != nil {
		return ""
	}
	return value
}

func decodeMomentFavoriteCursor(cursorValue string) (any, string, error) {
	if cursorValue == "" {
		return nil, "", nil
	}
	var decoded momentInteractionCursor
	if err := cursor.Decode(momentFavoriteCursorNamespace, cursorValue, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	favoritedAt, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return favoritedAt.UTC(), decoded.ID, nil
}

// queryable resolves the Queryable to use for this request: an in-flight
// transaction bound to ctx when present, otherwise the shared pool. Returns
// ok=false when persistence is not configured (in-memory mode).
func queryable(ctx context.Context, factory *persistence.Factory) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, factory)
}

// escapeLikePattern 转义 LIKE/ILIKE 的 \、%、_ 通配符（配 ESCAPE '\'）。
// 实现已上收 platform/textutil.EscapeLikePattern（users/groups 搜索同用）。
func escapeLikePattern(input string) string {
	return textutil.EscapeLikePattern(input)
}
