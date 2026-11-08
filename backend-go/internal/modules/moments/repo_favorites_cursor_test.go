package moments

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
)

// TestListFavoritesKeepsSameSecondRowsAcrossPages 回归：同秒多条收藏翻页不丢失。
// favorited_at 以全精度写入，若游标截断到秒，keyset 谓词
// (favorited_at < cursor) 对同秒剩余行恒假，这些行将静默丢失。
// 直接注入亚秒精度的收藏时刻（经公开 Favorite API 无法稳定构造同秒场景），
// 断言跨页遍历无丢失、无重复。
func TestListFavoritesKeepsSameSecondRowsAcrossPages(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	base := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	stamps := map[string]time.Time{
		"moment_1": base.Add(500 * time.Millisecond),
		"moment_2": base.Add(300 * time.Millisecond),
		"moment_3": base.Add(100 * time.Millisecond),
	}
	for id := range stamps {
		repo.moments[id] = MomentItem{MomentID: id}
	}
	repo.favorites["user_alice"] = stamps

	seen := map[string]bool{}
	cursorValue := ""
	for page := 0; page < 5; page++ {
		paged, err := repo.ListFavorites(ctx, "user_alice", 2, cursorValue)
		if err != nil {
			t.Fatalf("ListFavorites: %v", err)
		}
		for _, item := range paged.Items {
			if seen[item.MomentID] {
				t.Fatalf("duplicate moment %s across pages", item.MomentID)
			}
			seen[item.MomentID] = true
		}
		if paged.NextCursor == nil {
			break
		}
		cursorValue = *paged.NextCursor
	}
	for id := range stamps {
		if !seen[id] {
			t.Fatalf("moment %s lost during pagination", id)
		}
	}
}

// TestFavoriteCursorRoundTripsNanoseconds 游标必须携带纳秒全精度并可解码回原时刻；
// 秒级游标（旧格式）一律拒绝。
func TestFavoriteCursorRoundTripsNanoseconds(t *testing.T) {
	at := time.Date(2026, 3, 29, 12, 0, 0, 123456789, time.UTC)
	token := encodeMomentFavoriteCursor(at, "moment_9")
	if token == "" {
		t.Fatal("expected cursor token")
	}
	parsed, id, err := decodeMomentFavoriteCursor(token)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if id != "moment_9" {
		t.Fatalf("unexpected id: %s", id)
	}
	parsedTime, ok := parsed.(time.Time)
	if !ok || !parsedTime.Equal(at) {
		t.Fatalf("expected %v, got %#v", at, parsed)
	}
	secondsToken, err := cursor.Encode(momentFavoriteCursorNamespace, momentInteractionCursor{At: at.Format(time.RFC3339), ID: "moment_9"})
	if err != nil {
		t.Fatalf("encode seconds cursor: %v", err)
	}
	if _, _, err := decodeMomentFavoriteCursor(secondsToken); err == nil {
		t.Fatal("expected seconds-precision cursor to be rejected")
	}
}
