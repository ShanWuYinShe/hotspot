package moments

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// momentsContractState 契约用例上下文：PG 侧 moments/moment_tags/moment_mentions/
// moment_favorites 依赖 users 外键；内存实现无需预置用户。
type momentsContractState struct {
	ctx     context.Context
	author  string
	viewer  string
	viewer2 string
}

// TestMomentsRepositoryContract 接入双实现契约一致性基建：
// 创建/读取、我的瞬间 keyset 分页、搜索（关键词大小写不敏感/标签/隐私/收藏态）、
// 收藏生命周期与占位项、计数维护、编辑 patch 语义、软删语义。
func TestMomentsRepositoryContract(t *testing.T) {
	author := "it_moments_contract_author"
	viewer := "it_moments_contract_viewer"
	viewer2 := "it_moments_contract_viewer2"

	impls := []testdb.ContractImpl[Repository, momentsContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, momentsContractState, func()) {
				return NewInMemoryRepository(), momentsContractState{ctx: context.Background(), author: author, viewer: viewer, viewer2: viewer2}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, momentsContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				// 事务 scope 隔离：池直写会在包并行执行期间污染其它包的真实库测试
				//（admin StatsGeo 等基于全库基线+增量断言；2026-10-24 门禁实测失败）。
				// moments 读路径曾在 rows 未关闭时发嵌套查询（已修：先 rows.Close()），
				// 因此可在单连接事务下正常执行。
				txFactory, cleanup := testdb.BeginScope(t)
				testdb.InsertUser(t, txFactory, author, "nick_"+author)
				testdb.InsertUser(t, txFactory, viewer, "nick_"+viewer)
				testdb.InsertUser(t, txFactory, viewer2, "nick_"+viewer2)
				return NewPostgresRepository(txFactory), momentsContractState{ctx: context.Background(), author: author, viewer: viewer, viewer2: viewer2}, cleanup
			},
		},
	}

	location := &Location{
		LocationName: "Contract Place",
		Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 35.1, Longitude: 112.6},
	}

	create := func(t *testing.T, repo Repository, state momentsContractState, req CreateMomentRequest) MomentItem {
		t.Helper()
		req.Location = location
		item, err := repo.Create(state.ctx, state.author, req)
		if err != nil {
			t.Fatalf("create moment: %v", err)
		}
		if item.MomentID == "" {
			t.Fatal("created moment id must be non-empty")
		}
		return item
	}

	sortedStrings := func(in []string) []string {
		out := append([]string(nil), in...)
		for i := 1; i < len(out); i++ {
			for j := i; j > 0 && out[j] < out[j-1]; j-- {
				out[j], out[j-1] = out[j-1], out[j]
			}
		}
		return out
	}
	equalStrings := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, momentsContractState]{
		{
			Name: "create and get round-trips fields",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				created := create(t, repo, state, CreateMomentRequest{
					Description: "hello world",
					Tags:        []string{"zzz", "aaa", "aaa"},
					Mentions:    []string{state.viewer},
				})
				got, found, err := repo.Get(state.ctx, created.MomentID)
				if err != nil || !found {
					t.Fatalf("get: %#v err=%v", got, err)
				}
				if got.Description != "hello world" {
					t.Fatalf("description = %q", got.Description)
				}
				if !equalStrings(sortedStrings(got.Tags), []string{"aaa", "zzz"}) {
					t.Fatalf("tags = %#v", got.Tags)
				}
				if !equalStrings(sortedStrings(got.Mentions), []string{state.viewer}) {
					t.Fatalf("mentions = %#v", got.Mentions)
				}
				if got.PrivacyLevel != "public" {
					t.Fatalf("privacy = %q", got.PrivacyLevel)
				}
				if got.User.UserID != state.author || got.User.Nickname == "" {
					t.Fatalf("user = %#v", got.User)
				}
				if got.Location == nil || got.Location.LocationName != "Contract Place" {
					t.Fatalf("location = %#v", got.Location)
				}
				if _, err := time.Parse(time.RFC3339, got.CreatedAt); err != nil {
					t.Fatalf("created_at must be RFC3339: %q", got.CreatedAt)
				}
				if got.IsDeleted || got.CommentsCount != 0 || got.SharesCount != 0 || got.LikesCount != 0 {
					t.Fatalf("unexpected initial state: %#v", got)
				}
			},
		},
		{
			Name: "list mine is paginated and keyset-disjoint",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				for i := 0; i < 5; i++ {
					create(t, repo, state, CreateMomentRequest{Description: "mine"})
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.ListMine(state.ctx, state.author, 2, cursor)
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						if seen[item.MomentID] {
							t.Fatalf("duplicate moment across pages: %s", item.MomentID)
						}
						seen[item.MomentID] = true
					}
					pages++
					if page.NextCursor == nil {
						break
					}
					cursor = *page.NextCursor
					if pages > 5 {
						t.Fatal("pagination did not terminate")
					}
				}
				if len(seen) != 5 {
					t.Fatalf("seen %d moments, want 5", len(seen))
				}
				if _, err := repo.ListMine(state.ctx, state.author, 2, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
		{
			Name: "search keyword is case-insensitive and filters private/deleted",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				hit := create(t, repo, state, CreateMomentRequest{Description: "Hello World"})
				create(t, repo, state, CreateMomentRequest{Description: "unrelated"})
				private := create(t, repo, state, CreateMomentRequest{Description: "Hello private", PrivacyLevel: "private"})
				deleted := create(t, repo, state, CreateMomentRequest{Description: "Hello deleted"})
				if err := repo.Delete(state.ctx, deleted.MomentID); err != nil {
					t.Fatalf("delete: %v", err)
				}

				keyword := "hello"
				page, err := repo.Search(state.ctx, SearchMomentsRequest{Keyword: &keyword, SortBy: "newest", Pagination: &PaginationParams{Limit: 10}}, "")
				if err != nil {
					t.Fatalf("search: %v", err)
				}
				if len(page.Items) != 1 || page.Items[0].MomentID != hit.MomentID {
					t.Fatalf("search items = %#v", page.Items)
				}

				tag := "contract-tag"
				tagged := create(t, repo, state, CreateMomentRequest{Description: "tagged", Tags: []string{tag}})
				byTag, err := repo.Search(state.ctx, SearchMomentsRequest{Tag: &tag, SortBy: "newest"}, "")
				if err != nil {
					t.Fatalf("search by tag: %v", err)
				}
				if len(byTag.Items) != 1 || byTag.Items[0].MomentID != tagged.MomentID {
					t.Fatalf("tag search = %#v", byTag.Items)
				}

				_ = private
				if _, err := repo.Search(state.ctx, SearchMomentsRequest{Keyword: &keyword, SortBy: "newest", Pagination: &PaginationParams{Cursor: "not-a-cursor"}}, ""); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid search cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
		{
			Name: "favorite lifecycle and viewer state",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				item := create(t, repo, state, CreateMomentRequest{Description: "favorite me"})
				for i := 0; i < 2; i++ {
					if err := repo.Favorite(state.ctx, state.viewer, item.MomentID); err != nil {
						t.Fatalf("favorite %d: %v", i, err)
					}
				}
				favorites, err := repo.ListFavorites(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list favorites: %v", err)
				}
				if len(favorites.Items) != 1 || favorites.Items[0].MomentID != item.MomentID {
					t.Fatalf("favorites = %#v (duplicate favorite must be idempotent)", favorites.Items)
				}
				if !favorites.Items[0].IsFavorited || favorites.Items[0].FavoritedAt == "" {
					t.Fatalf("favorite state = %#v", favorites.Items[0])
				}
				userIDs, err := repo.ListFavoriteUserIDs(state.ctx, item.MomentID, 10)
				if err != nil {
					t.Fatalf("favorite user ids: %v", err)
				}
				if len(userIDs) != 1 || userIDs[0] != state.viewer {
					t.Fatalf("favorite user ids = %#v", userIDs)
				}
				keyword := "favorite"
				viewerSearch, err := repo.Search(state.ctx, SearchMomentsRequest{Keyword: &keyword, SortBy: "newest"}, state.viewer)
				if err != nil {
					t.Fatalf("viewer search: %v", err)
				}
				if len(viewerSearch.Items) != 1 || !viewerSearch.Items[0].IsFavorited {
					t.Fatalf("viewer search favorite state = %#v", viewerSearch.Items)
				}
				anonymousSearch, err := repo.Search(state.ctx, SearchMomentsRequest{Keyword: &keyword, SortBy: "newest"}, "")
				if err != nil {
					t.Fatalf("anonymous search: %v", err)
				}
				if len(anonymousSearch.Items) != 1 || anonymousSearch.Items[0].IsFavorited {
					t.Fatalf("anonymous search favorite state = %#v", anonymousSearch.Items)
				}
				if err := repo.Unfavorite(state.ctx, state.viewer, item.MomentID); err != nil {
					t.Fatalf("unfavorite: %v", err)
				}
				favorites, err = repo.ListFavorites(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list favorites after unfavorite: %v", err)
				}
				if len(favorites.Items) != 0 {
					t.Fatalf("favorites after unfavorite = %#v", favorites.Items)
				}
			},
		},
		{
			Name: "favorites pagination and deleted placeholder",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				ids := make([]string, 0, 3)
				for i := 0; i < 3; i++ {
					item := create(t, repo, state, CreateMomentRequest{Description: "fav page"})
					if err := repo.Favorite(state.ctx, state.viewer, item.MomentID); err != nil {
						t.Fatalf("favorite %d: %v", i, err)
					}
					ids = append(ids, item.MomentID)
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.ListFavorites(state.ctx, state.viewer, 2, cursor)
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					for _, item := range page.Items {
						if seen[item.MomentID] {
							t.Fatalf("duplicate favorite across pages: %s", item.MomentID)
						}
						seen[item.MomentID] = true
					}
					pages++
					if page.NextCursor == nil {
						break
					}
					cursor = *page.NextCursor
					if pages > 4 {
						t.Fatal("favorites pagination did not terminate")
					}
				}
				if len(seen) != 3 {
					t.Fatalf("seen %d favorites, want 3", len(seen))
				}
				if _, err := repo.ListFavorites(state.ctx, state.viewer, 2, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid favorites cursor err = %v, want ErrInvalidCursor", err)
				}
				// 软删后收藏列表保留占位项（spec「删除语义」）。
				if err := repo.Delete(state.ctx, ids[0]); err != nil {
					t.Fatalf("delete favorited moment: %v", err)
				}
				page, err := repo.ListFavorites(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list favorites after delete: %v", err)
				}
				foundPlaceholder := false
				for _, item := range page.Items {
					if item.MomentID == ids[0] {
						foundPlaceholder = true
						if !item.IsDeleted || item.Description != "" {
							t.Fatalf("deleted favorite placeholder = %#v", item)
						}
					}
				}
				if !foundPlaceholder {
					t.Fatalf("deleted favorite placeholder missing: %#v", page.Items)
				}
			},
		},
		{
			Name: "counters are maintained and clamped",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				item := create(t, repo, state, CreateMomentRequest{Description: "counters"})
				for i := 0; i < 2; i++ {
					if err := repo.IncrementCommentCount(state.ctx, item.MomentID); err != nil {
						t.Fatalf("increment comments: %v", err)
					}
				}
				if err := repo.DecrementCommentCount(state.ctx, item.MomentID); err != nil {
					t.Fatalf("decrement comments: %v", err)
				}
				if err := repo.IncrementSharesCount(state.ctx, item.MomentID); err != nil {
					t.Fatalf("increment shares: %v", err)
				}
				if err := repo.AdjustReactionCount(state.ctx, item.MomentID, 5); err != nil {
					t.Fatalf("adjust reactions +5: %v", err)
				}
				if err := repo.AdjustReactionCount(state.ctx, item.MomentID, -10); err != nil {
					t.Fatalf("adjust reactions -10: %v", err)
				}
				got, _, err := repo.Get(state.ctx, item.MomentID)
				if err != nil {
					t.Fatalf("get: %v", err)
				}
				if got.CommentsCount != 1 || got.SharesCount != 1 || got.LikesCount != 0 {
					t.Fatalf("counts = comments:%d shares:%d likes:%d", got.CommentsCount, got.SharesCount, got.LikesCount)
				}
			},
		},
		{
			Name: "update applies patch semantics",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				item := create(t, repo, state, CreateMomentRequest{Description: "before", Tags: []string{"old"}, Mentions: []string{state.viewer}})
				newDescription := "after"
				newPrivacy := "private"
				if err := repo.Update(state.ctx, state.author, UpdateMomentRequest{
					MomentID:     item.MomentID,
					Description:  &newDescription,
					PrivacyLevel: &newPrivacy,
					Tags:         []string{"new"},
				}); err != nil {
					t.Fatalf("update: %v", err)
				}
				got, _, err := repo.Get(state.ctx, item.MomentID)
				if err != nil {
					t.Fatalf("get after update: %v", err)
				}
				if got.Description != "after" || got.PrivacyLevel != "private" {
					t.Fatalf("updated = %#v", got)
				}
				if !equalStrings(got.Tags, []string{"new"}) {
					t.Fatalf("tags = %#v", got.Tags)
				}
				if !equalStrings(sortedStrings(got.Mentions), []string{state.viewer}) {
					t.Fatalf("mentions = %#v (nil patch must keep existing)", got.Mentions)
				}
				if err := repo.Update(state.ctx, state.viewer, UpdateMomentRequest{MomentID: item.MomentID, Description: &newDescription}); !errors.Is(err, ErrPermissionDenied) && !errors.Is(err, ErrMomentNotFound) {
					// repo 层两实现归属校验时机不同：内存先判 owner → ErrPermissionDenied；
					// PG 的 UPDATE 带 owner 条件 0 行命中 → ErrMomentNotFound。service 预检后均不可达，接受两者。
					t.Fatalf("non-owner update err = %v, want ErrPermissionDenied/ErrMomentNotFound", err)
				}
				if err := repo.Delete(state.ctx, item.MomentID); err != nil {
					t.Fatalf("delete: %v", err)
				}
				if err := repo.Update(state.ctx, state.author, UpdateMomentRequest{MomentID: item.MomentID, Description: &newDescription}); !errors.Is(err, ErrMomentNotFound) {
					t.Fatalf("update deleted err = %v, want ErrMomentNotFound", err)
				}
			},
		},
		{
			Name: "delete is soft and missing delete reports not found",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				item := create(t, repo, state, CreateMomentRequest{Description: "delete me"})
				if err := repo.Delete(state.ctx, item.MomentID); err != nil {
					t.Fatalf("delete: %v", err)
				}
				got, found, err := repo.Get(state.ctx, item.MomentID)
				if err != nil || !found {
					t.Fatalf("get deleted: %#v err=%v", got, err)
				}
				if !got.IsDeleted || got.Description != "" {
					t.Fatalf("deleted moment = %#v", got)
				}
				if err := repo.Delete(state.ctx, "moment_missing"); !errors.Is(err, ErrMomentNotFound) {
					t.Fatalf("delete missing err = %v, want ErrMomentNotFound", err)
				}
			},
		},
		{
			Name: "favorite user ids honor the fanout limit",
			Run: func(t *testing.T, repo Repository, state momentsContractState) {
				item := create(t, repo, state, CreateMomentRequest{Description: "fanout"})
				for _, userID := range []string{state.viewer, state.viewer2} {
					if err := repo.Favorite(state.ctx, userID, item.MomentID); err != nil {
						t.Fatalf("favorite %s: %v", userID, err)
					}
				}
				// 删除瞬间的收藏者广播由调用方传 fanout 上限：repo 必须真正限制行数，
				// 否则热门瞬间仍会把全部收藏行读进内存（审查 P2-17）。
				limited, err := repo.ListFavoriteUserIDs(state.ctx, item.MomentID, 1)
				if err != nil {
					t.Fatalf("limited ids: %v", err)
				}
				if len(limited) != 1 {
					t.Fatalf("limited ids = %v, want 1", limited)
				}
				if limited[0] != state.viewer && limited[0] != state.viewer2 {
					t.Fatalf("unexpected favorite user id: %q", limited[0])
				}
				all, err := repo.ListFavoriteUserIDs(state.ctx, item.MomentID, 10)
				if err != nil {
					t.Fatalf("all ids: %v", err)
				}
				if len(all) != 2 {
					t.Fatalf("all ids = %v, want 2", all)
				}
				unlimited, err := repo.ListFavoriteUserIDs(state.ctx, item.MomentID, 0)
				if err != nil {
					t.Fatalf("unlimited ids: %v", err)
				}
				if len(unlimited) != 2 {
					t.Fatalf("unlimited ids = %v, want 2", unlimited)
				}
			},
		},
	})
}
