package moments

import (
	"context"
	"errors"
	"fmt"
	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/persistence"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return MomentItem{}, false, persistence.ErrNotImplemented
	}
	item, err := scanMomentRow(queryable.QueryRow(ctx, queryGetMoment, momentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MomentItem{}, false, nil
		}
		return MomentItem{}, false, fmt.Errorf("moments.Get scan: %w", err)
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, []string{momentID})
	if err != nil {
		return MomentItem{}, false, fmt.Errorf("moments.Get tags: %w", err)
	}
	item.Tags = tagsByMomentID[momentID]
	mentionsByMomentID, err := loadMomentMentions(ctx, queryable, []string{momentID})
	if err != nil {
		return MomentItem{}, false, fmt.Errorf("moments.Get mentions: %w", err)
	}
	item.Mentions = mentionsByMomentID[momentID]
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
		item, err := scanMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, fmt.Errorf("moments.ListMine scan: %w", err)
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		cursors = append(cursors, encodeMomentMineCursor(item.CreatedAt, item.MomentID))
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
			c := cursors[limit-1]
			return PaginatedMoments{Items: items[:limit], NextCursor: &c}, nil
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
	if err := queryable.QueryRow(ctx, queryCheckMomentFavorite, userID, momentID).Scan(&favorited); err == nil {
		isFavorited = true
	}
	return &MyMomentState{IsFavorited: isFavorited}
}

func (r *PostgresRepository) Exists(ctx context.Context, momentID string) (bool, error) {
	_, ok, err := r.Get(ctx, momentID)
	return ok, err
}

func scanMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	item, _, err := scanMomentRowParts(scanner, false)
	return item, err
}

// scanSearchMomentRow 扫描 moments/search 查询行：基础列 + 尾部 is_favorited。
// 登录查看者时 buildSearchMomentsQuery 在 SELECT 尾部追加
// EXISTS(moment_favorites) 列（spec api_spec_types.md MomentItem 查看者布尔恒定），
// 匿名查询不产生该列、不走本函数。
func scanSearchMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, error) {
	item, isFavorited, err := scanMomentRowParts(scanner, true)
	if err != nil {
		return MomentItem{}, err
	}
	item.IsFavorited = isFavorited
	return item, nil
}

// scanMomentRowParts 扫描 moment 基础列；withViewerState 时额外扫描尾部
// is_favorited 布尔并原样返回（不写入 item，由调用方决定用途）。
func scanMomentRowParts(scanner interface{ Scan(...any) error }, withViewerState bool) (MomentItem, bool, error) {
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
		return MomentItem{}, false, fmt.Errorf("moments.scanMomentRow scan: %w", err)
	}
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
	return item, isFavorited, nil
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

func scanFavoriteMomentRow(scanner interface{ Scan(...any) error }) (MomentItem, string, error) {
	var item MomentItem
	var userID, nickname, locationName, privacyLevel string
	var createdAt, favoritedAt time.Time
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &item.IsDeleted, &favoritedAt); err != nil {
		return MomentItem{}, "", fmt.Errorf("moments.scanFavoriteMomentRow scan: %w", err)
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
	return item, favoritedAt.UTC().Format(time.RFC3339), nil
}
