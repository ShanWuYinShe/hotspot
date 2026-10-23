package moments

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type rowValues struct {
	values []any
	err    error
}

func (r rowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int:
			switch v := r.values[i].(type) {
			case int:
				*d = v
			case string:
				*d = 1
			}
		case *bool:
			*d = r.values[i].(bool)
		case *float64:
			*d = r.values[i].(float64)
		case *time.Time:
			switch v := r.values[i].(type) {
			case time.Time:
				*d = v
			case string:
				parsed, err := time.Parse(time.RFC3339Nano, v)
				if err != nil {
					return err
				}
				*d = parsed
			}
		case *[]string:
			// Get 合并查询的 tags/mentions 是 text[] 列（stub 里存 []string / nil）。
			switch v := r.values[i].(type) {
			case []string:
				*d = v
			case nil:
				*d = nil
			}
		}
	}
	return nil
}

type rowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *rowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *rowsValues) Scan(dest ...any) error {
	row := rowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *rowsValues) Close() error { return nil }
func (r *rowsValues) Err() error   { return r.err }

type backendStub struct {
	moments      map[string]MomentItem
	blockedTags  map[string]struct{}
	favorites    map[string]map[string]string
	execErrs     map[string]error
	queryErr     error
	queryRowErrs map[string]error
}

func newBackendStub() *backendStub {
	return &backendStub{moments: make(map[string]MomentItem), blockedTags: make(map[string]struct{}), favorites: make(map[string]map[string]string), execErrs: make(map[string]error), queryRowErrs: make(map[string]error)}
}

func (b *backendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	// 标签写入已批量化为动态多值 INSERT（审查 P2-15），无法按常量精确匹配；
	// 前缀识别后 args = [$1 momentID, $2 createdAt, tag...]。
	if strings.HasPrefix(query, "INSERT INTO moment_tags (moment_id, tag_name, created_at) VALUES ") {
		momentID := args[0].(string)
		item := b.moments[momentID]
		for _, raw := range args[2:] {
			item.Tags = append(item.Tags, raw.(string))
		}
		sort.Strings(item.Tags)
		b.moments[momentID] = item
		return nil
	}
	switch query {
	case queryInsertMoment:
		item := MomentItem{
			MomentID:      args[0].(string),
			Description:   args[2].(string),
			CreatedAt:     args[3].(string),
			LikesCount:    args[4].(int),
			CommentsCount: args[5].(int),
			SharesCount:   args[6].(int),
			User:          UserInfo{UserID: args[1].(string), Nickname: args[1].(string)},
			PrivacyLevel:  args[10].(string),
			IsFavorited:   false,
		}
		item.Location = &Location{}
		item.Location.LocationName = args[7].(string)
		item.Location.Point.Latitude = args[8].(float64)
		item.Location.Point.Longitude = args[9].(float64)
		b.moments[item.MomentID] = item
	case queryInsertMomentTagAssignment:
		momentID := args[0].(string)
		item := b.moments[momentID]
		tag := args[1].(string)
		item.Tags = append(item.Tags, tag)
		sort.Strings(item.Tags)
		b.moments[momentID] = item
	case queryInsertMomentMention:
		momentID := args[0].(string)
		item := b.moments[momentID]
		mentionedUserID := args[1].(string)
		item.Mentions = append(item.Mentions, mentionedUserID)
		sort.Strings(item.Mentions)
		b.moments[momentID] = item
	case queryDeleteMomentMentionsByMomentID:
		momentID := args[0].(string)
		item := b.moments[momentID]
		item.Mentions = nil
		b.moments[momentID] = item
	case queryIncrementMomentCommentCount:
		momentID := args[0].(string)
		item := b.moments[momentID]
		item.CommentsCount++
		b.moments[momentID] = item
	case queryIncrementMomentShareCount:
		momentID := args[0].(string)
		item := b.moments[momentID]
		item.SharesCount++
		b.moments[momentID] = item
	case queryInsertMomentFavorite:
		userID := args[0].(string)
		momentID := args[1].(string)
		favoritedAt := args[2].(time.Time).UTC().Format(time.RFC3339)
		if b.favorites[userID] == nil {
			b.favorites[userID] = make(map[string]string)
		}
		if _, exists := b.favorites[userID][momentID]; !exists {
			b.favorites[userID][momentID] = favoritedAt
		}
	case queryDeleteMomentFavorite:
		userID := args[0].(string)
		momentID := args[1].(string)
		if b.favorites[userID] != nil {
			delete(b.favorites[userID], momentID)
		}
	}
	return nil
}

func (b *backendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if err := b.queryRowErrs[query]; err != nil {
		return rowValues{err: err}
	}
	switch query {
	case queryGetUserNickname:
		userID := args[0].(string)
		for _, item := range b.moments {
			if item.User.UserID == userID {
				return rowValues{values: []any{item.User.Nickname}}
			}
		}
		return rowValues{err: pgx.ErrNoRows}
	case queryGetMoment:
		item, ok := b.moments[args[0].(string)]
		if !ok {
			return rowValues{err: pgx.ErrNoRows}
		}
		// queryGetMoment 合并了 tags/mentions 两个 ARRAY_AGG 列（审查 P2-16）。
		return rowValues{values: []any{item.MomentID, item.User.UserID, item.User.Nickname, item.Description, item.CreatedAt, item.LikesCount, item.CommentsCount, item.SharesCount, item.Location.LocationName, item.Location.Point.Latitude, item.Location.Point.Longitude, item.PrivacyLevel, item.IsDeleted, item.Tags, item.Mentions}}
	default:
		return rowValues{err: persistence.ErrNotImplemented}
	}
}

func (b *backendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if b.queryErr != nil {
		return nil, b.queryErr
	}
	if strings.Contains(query, "FROM moments") && strings.Contains(query, "privacy_level =") {
		// 全球搜索（spec moment.md → moments/search 收口）：无 S2 半径子句，keyword 从 $1 起
		argIndex := 0
		var keywordStr string
		if len(args) > argIndex {
			if s, ok := args[argIndex].(string); ok {
				keywordStr = s
				argIndex++
			}
		}
		var tagFilter string
		if len(args) > argIndex {
			if s, ok := args[argIndex].(string); ok {
				tagFilter = s
				argIndex++
			}
		}
		limit := -1
		if strings.Contains(query, " LIMIT $") && len(args) > argIndex {
			limit = args[argIndex].(int)
		}
		items := make([]MomentItem, 0)
		for _, item := range b.moments {
			if item.PrivacyLevel != "public" {
				continue
			}
			if keywordStr != "" && !containsTagInDescription(item.Description, keywordStr) {
				continue
			}
			if tagFilter != "" && !containsTag(item.Tags, tagFilter) {
				continue
			}
			items = append(items, item)
		}
		if limit >= 0 {
			// popular 排序的互动数来自 likes_count 计数冗余列（语义已复活为
			// reactions 总数）：此桩用 LikesCount 字段模拟该列，按新 SQL 片段识别 popular。
			popular := strings.Contains(query, "m.likes_count + m.comments_count") && strings.Contains(query, "POWER(")
			sort.Slice(items, func(i, j int) bool {
				if popular {
					si := items[i].LikesCount + items[i].CommentsCount
					sj := items[j].LikesCount + items[j].CommentsCount
					if si != sj {
						return si > sj
					}
				}
				if items[i].CreatedAt == items[j].CreatedAt {
					return items[i].MomentID > items[j].MomentID
				}
				return items[i].CreatedAt > items[j].CreatedAt
			})
			if len(items) > limit {
				items = items[:limit]
			}
		}
		rows := make([][]any, 0, len(items))
		for _, item := range items {
			rows = append(rows, []any{item.MomentID, item.User.UserID, item.User.Nickname, item.Description, item.CreatedAt, item.LikesCount, item.CommentsCount, item.SharesCount, item.Location.LocationName, item.Location.Point.Latitude, item.Location.Point.Longitude, item.PrivacyLevel, item.IsDeleted})
		}
		return &rowsValues{rows: rows}, nil
	}
	switch query {
	case queryListMyMoments:
		userID := args[0].(string)
		items := make([]MomentItem, 0)
		for _, item := range b.moments {
			if item.User.UserID == userID {
				items = append(items, item)
			}
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].MomentID > items[j].MomentID
			}
			return items[i].CreatedAt > items[j].CreatedAt
		})
		rows := make([][]any, 0, len(items))
		for _, item := range items {
			rows = append(rows, []any{item.MomentID, item.User.UserID, item.User.Nickname, item.Description, item.CreatedAt, item.LikesCount, item.CommentsCount, item.SharesCount, item.Location.LocationName, item.Location.Point.Latitude, item.Location.Point.Longitude, item.PrivacyLevel, item.IsDeleted})
		}
		return &rowsValues{rows: rows}, nil
	case queryFilterAllowedMomentTags:
		input := args[0].([]string)
		rows := make([][]any, 0, len(input))
		for _, tag := range input {
			if _, blocked := b.blockedTags[tag]; blocked {
				continue
			}
			rows = append(rows, []any{tag})
		}
		return &rowsValues{rows: rows}, nil
	case queryListMomentTagsByMomentID, queryListMomentTagsByMomentIDs:
		batchTagRows := func(ids []string) [][]any {
			out := make([][]any, 0)
			for _, mid := range ids {
				item, ok := b.moments[mid]
				if !ok {
					continue
				}
				for _, tag := range item.Tags {
					out = append(out, []any{mid, tag})
				}
			}
			return out
		}
		switch v := args[0].(type) {
		case string:
			return &rowsValues{rows: batchTagRows([]string{v})}, nil
		case []string:
			return &rowsValues{rows: batchTagRows(v)}, nil
		}
		return &rowsValues{rows: nil}, nil
	case queryListMomentMentionsByMomentID, queryListMomentMentionsByMomentIDs:
		batchMentionRows := func(ids []string) [][]any {
			out := make([][]any, 0)
			for _, mid := range ids {
				item, ok := b.moments[mid]
				if !ok {
					continue
				}
				for _, mentionedUserID := range item.Mentions {
					out = append(out, []any{mid, mentionedUserID})
				}
			}
			return out
		}
		switch v := args[0].(type) {
		case string:
			return &rowsValues{rows: batchMentionRows([]string{v})}, nil
		case []string:
			return &rowsValues{rows: batchMentionRows(v)}, nil
		}
		return &rowsValues{rows: nil}, nil
	case queryListFavoriteMoments:
		userID := args[0].(string)
		cursorAt := ""
		if args[1] != nil {
			cursorAt = args[1].(time.Time).UTC().Format(time.RFC3339)
		}
		cursorMomentID := args[2].(string)
		limit := args[3].(int)
		type favoriteRow struct {
			moment      MomentItem
			favoritedAt string
		}
		favoriteRows := make([]favoriteRow, 0)
		for momentID, favoritedAt := range b.favorites[userID] {
			item, ok := b.moments[momentID]
			if !ok {
				continue
			}
			if cursorAt != "" {
				if favoritedAt > cursorAt || (favoritedAt == cursorAt && momentID >= cursorMomentID) {
					continue
				}
			}
			favoriteRows = append(favoriteRows, favoriteRow{moment: item, favoritedAt: favoritedAt})
		}
		sort.Slice(favoriteRows, func(i, j int) bool {
			if favoriteRows[i].favoritedAt == favoriteRows[j].favoritedAt {
				return favoriteRows[i].moment.MomentID > favoriteRows[j].moment.MomentID
			}
			return favoriteRows[i].favoritedAt > favoriteRows[j].favoritedAt
		})
		if len(favoriteRows) > limit {
			favoriteRows = favoriteRows[:limit]
		}
		rows := make([][]any, 0, len(favoriteRows))
		for _, row := range favoriteRows {
			item := row.moment
			rows = append(rows, []any{item.MomentID, item.User.UserID, item.User.Nickname, item.Description, item.CreatedAt, item.LikesCount, item.CommentsCount, item.SharesCount, item.Location.LocationName, item.Location.Point.Latitude, item.Location.Point.Longitude, item.PrivacyLevel, item.IsDeleted, row.favoritedAt})
		}
		return &rowsValues{rows: rows}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func TestPostgresRepositoryReadsQueryableDB(t *testing.T) {
	backend := newBackendStub()
	backend.moments["moment_1"] = MomentItem{MomentID: "moment_1", Description: "hello", CreatedAt: "2026-03-20T00:00:00Z", LikesCount: 1, CommentsCount: 2, SharesCount: 3, User: UserInfo{UserID: "user_alice", Nickname: "user_alice"}, PrivacyLevel: "public", Location: &Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 35.0, Longitude: 112.0}, LocationName: "南天门"}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	item, ok, err := repo.Get(context.Background(), "moment_1")
	if err != nil || !ok || item.MomentID != "moment_1" || item.Description != "hello" {
		t.Fatalf("unexpected item: %#v ok=%v err=%v", item, ok, err)
	}
	list, err := repo.ListMine(context.Background(), "user_alice", 20, "")
	if err != nil || len(list.Items) != 1 || list.Items[0].MomentID != "moment_1" {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
}

func TestPostgresRepositoryWritesQueryableDB(t *testing.T) {
	backend := newBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Tags: []string{" travel ", "hotspot", "travel"}, Mentions: []string{"user_bob", "user_bob", "user_charlie"}, Location: &Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 35.0, Longitude: 112.0}, LocationName: "南天门"}})
	if err != nil || created.MomentID == "" || created.Description != "hello" {
		t.Fatalf("unexpected created item: %#v err=%v", created, err)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "travel" || created.Tags[1] != "hotspot" {
		t.Fatalf("unexpected persisted tags: %#v", created.Tags)
	}
	if len(created.Mentions) != 2 || created.Mentions[0] != "user_bob" || created.Mentions[1] != "user_charlie" {
		t.Fatalf("unexpected persisted mentions: %#v", created.Mentions)
	}
	if err := repo.IncrementCommentCount(context.Background(), created.MomentID); err != nil {
		t.Fatalf("increment comment count: %v", err)
	}
	opened, ok, err := repo.Get(context.Background(), created.MomentID)
	if err != nil || !ok || opened.CommentsCount != 1 {
		t.Fatalf("unexpected opened item: %#v ok=%v err=%v", opened, ok, err)
	}
	if len(opened.Mentions) != 2 || opened.Mentions[0] != "user_bob" || opened.Mentions[1] != "user_charlie" {
		t.Fatalf("unexpected opened mentions: %#v", opened.Mentions)
	}
	list, err := repo.ListMine(context.Background(), "user_alice", 20, "")
	if err != nil || len(list.Items) != 1 || list.Items[0].MomentID != created.MomentID || list.Items[0].CommentsCount != 1 {
		t.Fatalf("unexpected list after write: %#v err=%v", list, err)
	}
	if len(list.Items[0].Mentions) != 2 || list.Items[0].Mentions[0] != "user_bob" || list.Items[0].Mentions[1] != "user_charlie" {
		t.Fatalf("unexpected list mentions: %#v", list.Items[0].Mentions)
	}
}

func TestPostgresRepositoryGetReturnsNotFoundOnNoRows(t *testing.T) {
	backend := newBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	_, ok, err := repo.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ok {
		t.Fatal("expected not found")
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := newBackendStub()
	backend.execErrs[queryInsertMoment] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	_, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "南天门"}})
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestPostgresRepositoryIncrementCommentCountReturnsErrorOnExecFailure(t *testing.T) {
	backend := newBackendStub()
	backend.execErrs[queryIncrementMomentCommentCount] = errors.New("update failed")
	backend.moments["moment_1"] = MomentItem{MomentID: "moment_1", User: UserInfo{UserID: "user_alice"}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	if err := repo.IncrementCommentCount(context.Background(), "moment_1"); err == nil {
		t.Fatal("expected increment error")
	}
	if backend.moments["moment_1"].CommentsCount != 0 {
		t.Fatalf("unexpected comments count: %#v", backend.moments["moment_1"])
	}
}

func TestPostgresRepositoryCreateRejectsBlockedTags(t *testing.T) {
	backend := newBackendStub()
	backend.blockedTags["blocked"] = struct{}{}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	// 含被屏蔽标签 → ErrBlockedTag（handler 映射为 10400，spec moment.md create 错误码）
	_, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Tags: []string{"travel", "blocked", "hotspot"}, Location: &Location{LocationName: "南天门"}})
	if !errors.Is(err, ErrBlockedTag) {
		t.Fatalf("expected ErrBlockedTag, got %v", err)
	}
	// 全部标签允许 → 成功
	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Tags: []string{"travel", "hotspot"}, Location: &Location{LocationName: "南天门"}})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "travel" || created.Tags[1] != "hotspot" {
		t.Fatalf("unexpected persisted tags: %#v", created.Tags)
	}
}

func TestPostgresRepositoryListsFavorites(t *testing.T) {
	backend := newBackendStub()
	backend.moments["moment_1"] = MomentItem{MomentID: "moment_1", Description: "one", CreatedAt: "2026-03-20T00:00:00Z", User: UserInfo{UserID: "user_alice"}, PrivacyLevel: "public", Location: &Location{LocationName: "南天门"}, Tags: []string{"a"}}
	backend.moments["moment_2"] = MomentItem{MomentID: "moment_2", Description: "two", CreatedAt: "2026-03-19T00:00:00Z", User: UserInfo{UserID: "user_alice"}, PrivacyLevel: "public", Location: &Location{LocationName: "南天门"}, Tags: []string{"b"}}
	backend.moments["moment_3"] = MomentItem{MomentID: "moment_3", Description: "three", CreatedAt: "2026-03-18T00:00:00Z", User: UserInfo{UserID: "user_alice"}, PrivacyLevel: "public", Location: &Location{LocationName: "南天门"}, Tags: []string{"c"}}
	backend.favorites["user_bob"] = map[string]string{
		"moment_1": "2026-03-23T00:00:00Z",
		"moment_2": "2026-03-22T00:00:00Z",
		"moment_3": "2026-03-22T00:00:00Z",
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	favorites, err := repo.ListFavorites(context.Background(), "user_bob", 2, "")
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(favorites.Items) != 2 || favorites.Items[0].MomentID != "moment_1" || favorites.Items[1].MomentID != "moment_3" {
		t.Fatalf("unexpected favorites: %#v", favorites)
	}
	if favorites.NextCursor == nil {
		t.Fatal("expected favorites next cursor")
	}
	if !favorites.Items[0].IsFavorited {
		t.Fatalf("expected favorited state: %#v", favorites.Items[0])
	}
	favorites, err = repo.ListFavorites(context.Background(), "user_bob", 2, *favorites.NextCursor)
	if err != nil {
		t.Fatalf("list favorites second page: %v", err)
	}
	if len(favorites.Items) != 1 || favorites.Items[0].MomentID != "moment_2" || favorites.NextCursor != nil {
		t.Fatalf("unexpected second page favorites: %#v", favorites)
	}
}

func TestPostgresRepositoryInteractionReadsReturnErrorsWithoutFallback(t *testing.T) {
	backend := newBackendStub()
	backend.queryErr = errors.New("boom")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	if _, err := repo.ListFavorites(context.Background(), "user_bob", 20, ""); err == nil {
		t.Fatal("expected list favorites error")
	}
}

func containsTagInDescription(description, keyword string) bool {
	return strings.Contains(description, keyword)
}

func TestPostgresRepositorySearchSortByPopularReturnsOlderPopularMoments(t *testing.T) {
	// Regression test: before the fix, the Postgres search path always fetched
	// LIMIT (limit+1) rows ordered by created_at DESC, then re-sorted in Go for
	// non-newest modes. This meant older but more popular moments were excluded
	// from the candidate window. This test verifies that sort_by=popular returns
	// older moments when they have higher engagement.
	backend := newBackendStub()
	// newest moment, low engagement
	backend.moments["moment_new"] = MomentItem{
		MomentID: "moment_new", Description: "new quiet post", CreatedAt: "2026-04-15T00:00:00Z",
		LikesCount: 0, CommentsCount: 0, SharesCount: 0,
		User: UserInfo{UserID: "user_alice"}, PrivacyLevel: "public",
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 35.0, Longitude: 112.0}},
	}
	// older moment, high engagement
	backend.moments["moment_old_popular"] = MomentItem{
		MomentID: "moment_old_popular", Description: "old viral post", CreatedAt: "2026-04-01T00:00:00Z",
		LikesCount: 100, CommentsCount: 50, SharesCount: 0,
		User: UserInfo{UserID: "user_alice"}, PrivacyLevel: "public",
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 35.0, Longitude: 112.0}},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	keyword := "post"
	result, err := repo.Search(context.Background(), SearchMomentsRequest{
		Keyword:    &keyword,
		SortBy:     "popular",
		Pagination: &PaginationParams{Limit: 1},
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].MomentID != "moment_old_popular" {
		t.Fatalf("expected most popular moment (moment_old_popular), got %s — older popular moment was excluded by newest-first limit", result.Items[0].MomentID)
	}
}

func TestPostgresRepositorySearchSortByNewestAppliesLimit(t *testing.T) {
	// Verify that sort_by=newest still uses the efficient LIMIT path and returns
	// only the newest moments up to the requested limit.
	backend := newBackendStub()
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("moment_%d", i)
		backend.moments[id] = MomentItem{
			MomentID: id, Description: "post", CreatedAt: fmt.Sprintf("2026-04-%02dT00:00:00Z", 10+i),
			LikesCount: 0, CommentsCount: 0, SharesCount: 0,
			User: UserInfo{UserID: "user_alice"}, PrivacyLevel: "public",
			Location: &Location{Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 35.0, Longitude: 112.0}},
		}
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	result, err := repo.Search(context.Background(), SearchMomentsRequest{
		SortBy:     "newest",
		Pagination: &PaginationParams{Limit: 3},
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	if result.Items[0].MomentID != "moment_4" {
		t.Fatalf("expected newest moment (moment_4), got %s", result.Items[0].MomentID)
	}
	if result.NextCursor == nil {
		t.Fatal("expected next cursor for newest with more results")
	}
}
