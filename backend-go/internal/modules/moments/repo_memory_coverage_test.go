package moments

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 防回归（P1-2）：已软删瞬间按不存在处理，Update 不得把软删行静默改回
// （此前 Update 不检查 IsDeleted，软删时清空的 description 可被复活落库）。
func TestMemoryUpdateSoftDeletedMomentReturnsNotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Delete(ctx, created.MomentID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	edited := "edited"
	if err := repo.Update(ctx, "u1", UpdateMomentRequest{Description: &edited}); !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("update on soft-deleted moment should be ErrMomentNotFound, got %v", err)
	}
}

// 防回归（P1-3）：Update 的 blocked tag 口径对齐 Create（含被屏蔽标签整体
// 拒绝 ErrBlockedTag），且拒绝时不留下「tags 已删未插回」的半更新状态。
func TestMemoryUpdateBlockedTagRejected(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.SetBlockedTags("bad_tag")
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi", Tags: []string{"good"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Update(ctx, "u1", UpdateMomentRequest{MomentID: created.MomentID, Tags: []string{"good", "bad_tag"}}); !errors.Is(err, ErrBlockedTag) {
		t.Fatalf("update with blocked tag should be ErrBlockedTag, got %v", err)
	}
	got, found, err := repo.Get(ctx, created.MomentID)
	if err != nil || !found {
		t.Fatalf("get after rejected update: found=%v err=%v", found, err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "good" {
		t.Fatalf("tags should be untouched on rejected update, got %#v", got.Tags)
	}
}

func TestMemoryMomentCountersAndFavorites(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// 计数器
	if err := repo.IncrementCommentCount(ctx, created.MomentID); err != nil {
		t.Fatalf("increment comments: %v", err)
	}
	if err := repo.DecrementCommentCount(ctx, created.MomentID); err != nil {
		t.Fatalf("decrement comments: %v", err)
	}
	// 减到负数钳制为 0
	if err := repo.DecrementCommentCount(ctx, created.MomentID); err != nil {
		t.Fatalf("decrement below zero: %v", err)
	}
	if err := repo.IncrementSharesCount(ctx, created.MomentID); err != nil {
		t.Fatalf("increment shares: %v", err)
	}
	// 不存在的 moment 计数器 no-op
	if err := repo.DecrementCommentCount(ctx, "missing"); err != nil {
		t.Fatalf("decrement missing: %v", err)
	}
	if err := repo.IncrementSharesCount(ctx, "missing"); err != ErrMomentNotFound {
		t.Fatalf("increment shares missing: expected not found, got %v", err)
	}

	// NotificationContext
	item, found, err := repo.NotificationContext(ctx, created.MomentID)
	if err != nil || !found || item.MomentID != created.MomentID {
		t.Fatalf("notification context: %#v found=%v err=%v", item, found, err)
	}

	// 收藏/取消收藏
	if err := repo.Favorite(ctx, "u2", created.MomentID); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	if err := repo.Favorite(ctx, "u2", "missing"); err != ErrMomentNotFound {
		t.Fatalf("favorite missing: expected not found, got %v", err)
	}
	if err := repo.Unfavorite(ctx, "u2", created.MomentID); err != nil {
		t.Fatalf("unfavorite: %v", err)
	}

	// MyState
	if state := repo.LoadMyState(ctx, created.MomentID, "u2"); state == nil {
		t.Fatal("my state should exist for logged-in viewer")
	}
	if state := repo.LoadMyState(ctx, created.MomentID, ""); state != nil {
		t.Fatal("anonymous viewer must have no my_state")
	}
	if state := repo.LoadMyState(ctx, "missing", "u2"); state == nil || state.IsFavorited {
		t.Fatalf("missing moment my_state: %#v", state)
	}
}

func TestMemoryMomentDeleteIdempotent(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	created, err := repo.Create(ctx, "u1", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Delete(ctx, created.MomentID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// 软删除幂等：重复删除返回成功（与 Postgres UPDATE 语义一致）。
	if err := repo.Delete(ctx, created.MomentID); err != nil {
		t.Fatalf("double delete: expected idempotent success, got %v", err)
	}
	// 软删除语义（spec api_spec_moment.md「删除语义」）：记录保留为占位项。
	item, ok, _ := repo.Get(ctx, created.MomentID)
	if !ok || !item.IsDeleted || item.Description != "" {
		t.Fatalf("expected retained soft-delete placeholder, got ok=%v item=%#v", ok, item)
	}
}

// TestMemoryDeleteKeepsSoftDeletedPlaceholders：软删后作者 fetch_created 与收藏者
// fetch_favorites 均返回占位项（is_deleted=true、description 为空），其余场景过滤
// （spec api_spec_moment.md「删除语义」可见性）。
func TestMemoryDeleteKeepsSoftDeletedPlaceholders(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	created, err := repo.Create(ctx, "author", CreateMomentRequest{Description: "hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Favorite(ctx, "collector", created.MomentID); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	if err := repo.Delete(ctx, created.MomentID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	mine, err := repo.ListMine(ctx, "author", 20, "")
	if err != nil {
		t.Fatalf("list mine: %v", err)
	}
	if len(mine.Items) != 1 || !mine.Items[0].IsDeleted || mine.Items[0].Description != "" {
		t.Fatalf("fetch_created must surface soft-delete placeholder, got %#v", mine.Items)
	}

	favs, err := repo.ListFavorites(ctx, "collector", 20, "")
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(favs.Items) != 1 || !favs.Items[0].IsDeleted || favs.Items[0].Description != "" {
		t.Fatalf("fetch_favorites must surface soft-delete placeholder, got %#v", favs.Items)
	}
	if !favs.Items[0].IsFavorited {
		t.Fatalf("favorited placeholder must keep is_favorited=true, got %#v", favs.Items[0])
	}

	// 其余场景按不存在处理：搜索与存在性检查过滤软删项。
	result, err := repo.Search(ctx, SearchMomentsRequest{SortBy: "popular"}, "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, it := range result.Items {
		if it.MomentID == created.MomentID {
			t.Fatal("soft-deleted moment must be filtered from search")
		}
	}
}

// TestMemoryListFavoritesOrdersByFavoritedAt：排序为 favorited_at 时间倒序 +
// moment_id DESC 稳定 tie-break，keyset 游标不重不漏
// （spec api_spec_moment.md moments/fetch_favorites 排序）。
func TestMemoryListFavoritesOrdersByFavoritedAt(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	ids := make([]string, 0, 3)
	for _, label := range []string{"moment_a", "moment_b", "moment_c"} {
		m, err := repo.Create(ctx, "author", CreateMomentRequest{Description: label})
		if err != nil {
			t.Fatalf("create %s: %v", label, err)
		}
		ids = append(ids, m.MomentID)
	}
	aID, bID, cID := ids[0], ids[1], ids[2]
	// Create 生成的 ID 单调递增（UnixNano+seq）→ c > a，同刻 tie-break 时 c 在前。
	// 直接注入收藏时刻以构造确定顺序与同刻 tie-break：
	// favorited_at: b(03-22) < a=c(03-23 同刻，moment_id DESC → c 前)。
	repo.favorites["viewer"] = map[string]time.Time{
		aID: time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
		bID: time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		cID: time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
	}

	favs, err := repo.ListFavorites(ctx, "viewer", 2, "")
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(favs.Items) != 2 || favs.NextCursor == nil {
		t.Fatalf("expected page of 2 with cursor, got %#v", favs)
	}
	if favs.Items[0].MomentID != cID || favs.Items[1].MomentID != aID {
		t.Fatalf("expected favorited_at DESC with moment_id DESC tie-break [c a], got [%s %s]",
			favs.Items[0].MomentID, favs.Items[1].MomentID)
	}
	// FavoritedMomentItem.favorited_at 随响应返回（spec api_spec_moment.md 本模块类型）。
	if favs.Items[0].FavoritedAt == "" || favs.Items[0].FavoritedAt != "2026-03-23T00:00:00Z" {
		t.Fatalf("expected favorited_at echoed on items, got %#v", favs.Items[0])
	}

	page2, err := repo.ListFavorites(ctx, "viewer", 2, *favs.NextCursor)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].MomentID != bID || page2.NextCursor != nil {
		t.Fatalf("unexpected second page: %#v", page2)
	}
}
