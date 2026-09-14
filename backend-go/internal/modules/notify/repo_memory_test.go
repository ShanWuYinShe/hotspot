package notify

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/types"
)

// TestInMemoryAppendMany_FansOutToAllUsers verifies AppendMany delivers the
// notify to every recipient (the postgres path now does this in a single
// batched transaction; this guards the in-memory mirror used by tests).
func TestInMemoryAppendMany_FansOutToAllUsers(t *testing.T) {
	repo := NewInMemoryRepository()
	users := []string{"u1", "u2", "u3"}
	item := NotifyItem{Type: "mention", Payload: map[string]any{"moment_id": "m1"}}

	if err := repo.AppendMany(context.Background(), users, item); err != nil {
		t.Fatalf("AppendMany failed: %v", err)
	}

	for _, uid := range users {
		page, err := repo.List(context.Background(), uid, NotifyListRequest{})
		if err != nil {
			t.Fatalf("List(%s) failed: %v", uid, err)
		}
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notify, got %d", uid, len(page.Items))
		}
		if page.Items[0].Type != "mention" {
			t.Fatalf("user %s: expected type mention, got %s", uid, page.Items[0].Type)
		}
	}
}

// TestInMemoryAppendMany_SkipsEmptyUserIDs verifies empty user ids are ignored
// (matching the postgres batch path's filtering).
func TestInMemoryAppendMany_SkipsEmptyUserIDs(t *testing.T) {
	repo := NewInMemoryRepository()
	users := []string{"u1", "", "u2", ""}
	item := NotifyItem{Type: "reaction"}

	if err := repo.AppendMany(context.Background(), users, item); err != nil {
		t.Fatalf("AppendMany failed: %v", err)
	}

	for _, uid := range []string{"u1", "u2"} {
		page, _ := repo.List(context.Background(), uid, NotifyListRequest{})
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notify, got %d", uid, len(page.Items))
		}
	}
}

// TestInMemoryAppendMany_EmptyListIsNoop verifies a nil/empty recipient list
// is a safe no-op.
func TestInMemoryAppendMany_EmptyListIsNoop(t *testing.T) {
	repo := NewInMemoryRepository()
	if err := repo.AppendMany(context.Background(), nil, NotifyItem{Type: "x"}); err != nil {
		t.Fatalf("AppendMany(nil) failed: %v", err)
	}
	if err := repo.AppendMany(context.Background(), []string{}, NotifyItem{Type: "x"}); err != nil {
		t.Fatalf("AppendMany([]) failed: %v", err)
	}
}

// TestInMemoryAppendMany_DeliversAllWithSharedId verifies every recipient
// gets a notify even though the in-memory AppendMany passes the same
// NotifyItem (and thus the same NotifyID) to each Append. The postgres
// batch path now derives a distinct per-row id from a shared prefix (the
// table PK is notify_id alone, so a shared id would collide); this test
// confirms the in-memory fan-out still delivers to all users.
func TestInMemoryAppendMany_DeliversAllWithSharedId(t *testing.T) {
	repo := NewInMemoryRepository()
	users := []string{"u1", "u2"}
	item := NotifyItem{NotifyID: "shared_notif", Type: "mention"}
	if err := repo.AppendMany(context.Background(), users, item); err != nil {
		t.Fatalf("AppendMany failed: %v", err)
	}
	for _, uid := range users {
		page, _ := repo.List(context.Background(), uid, NotifyListRequest{})
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notify, got %d", uid, len(page.Items))
		}
	}
}

// --- AppendBatchPerUser（@ 提及批量：每用户独立 notify_id） ---

// TestInMemoryAppendBatchPerUser_DeliversPerUserItems verifies each recipient
// gets its own item (and thus its own notify_id), mirroring the postgres
// per-row batch insert.
func TestInMemoryAppendBatchPerUser_DeliversPerUserItems(t *testing.T) {
	repo := NewInMemoryRepository()
	users := []string{"u1", "u2", "u3"}
	items := []NotifyItem{
		{NotifyID: "notif_1", Type: "mentioned", Payload: map[string]any{"moment_id": "m1"}},
		{NotifyID: "notif_2", Type: "mentioned", Payload: map[string]any{"moment_id": "m1"}},
		{NotifyID: "notif_3", Type: "mentioned", Payload: map[string]any{"moment_id": "m1"}},
	}
	if err := repo.AppendBatchPerUser(context.Background(), users, items); err != nil {
		t.Fatalf("AppendBatchPerUser failed: %v", err)
	}
	for i, uid := range users {
		page, _ := repo.List(context.Background(), uid, NotifyListRequest{})
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notify, got %d", uid, len(page.Items))
		}
		if page.Items[0].NotifyID != items[i].NotifyID {
			t.Fatalf("user %s: notify_id = %q, want %q", uid, page.Items[0].NotifyID, items[i].NotifyID)
		}
	}
}

// TestInMemoryAppendBatchPerUser_SkipsEmptyUserIDs verifies empty user ids are
// ignored (matching AppendMany and the postgres batch filtering).
func TestInMemoryAppendBatchPerUser_SkipsEmptyUserIDs(t *testing.T) {
	repo := NewInMemoryRepository()
	err := repo.AppendBatchPerUser(context.Background(),
		[]string{"u1", "", "u2"},
		[]NotifyItem{{NotifyID: "notif_1", Type: "mentioned"}, {NotifyID: "notif_2", Type: "mentioned"}, {NotifyID: "notif_3", Type: "mentioned"}})
	if err != nil {
		t.Fatalf("AppendBatchPerUser failed: %v", err)
	}
	for _, uid := range []string{"u1", "u2"} {
		page, _ := repo.List(context.Background(), uid, NotifyListRequest{})
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notify, got %d", uid, len(page.Items))
		}
	}
}

// TestInMemoryAppendBatchPerUser_LengthMismatchIsError verifies the positional
// pairing contract is enforced.
func TestInMemoryAppendBatchPerUser_LengthMismatchIsError(t *testing.T) {
	repo := NewInMemoryRepository()
	if err := repo.AppendBatchPerUser(context.Background(), []string{"u1"}, []NotifyItem{}); err == nil {
		t.Fatal("expected length mismatch error")
	}
}

// PruneReadBefore：只删「已读且 created_at 超期」的通知，未读与新鲜通知保留。
func TestPruneReadBeforeDeletesOnlyExpiredRead(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	old := time.Now().UTC().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	fresh := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	appendAt := func(userID, id, createdAt string, isRead bool) {
		if err := repo.Append(ctx, userID, NotifyItem{NotifyID: id, Type: "new_follower", IsRead: isRead, CreatedAt: createdAt}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	appendAt("u1", "notif_old_read", old, true)    // 删
	appendAt("u1", "notif_old_unread", old, false) // 留（未读）
	appendAt("u1", "notif_fresh_read", fresh, true)
	appendAt("u2", "notif_other", old, true) // 删（跨用户）

	pruned, err := repo.PruneReadBefore(ctx, time.Now().UTC().Add(-90*24*time.Hour))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if pruned != 2 {
		t.Fatalf("expected 2 pruned, got %d", pruned)
	}
	items, err := repo.List(ctx, "u1", NotifyListRequest{Pagination: &types.PaginationParams{Limit: 10}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items.Items) != 2 {
		t.Fatalf("expected 2 remaining for u1, got %d", len(items.Items))
	}
}
