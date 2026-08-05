package notify

import (
	"context"
	"testing"
)

// TestInMemoryAppendMany_FansOutToAllUsers verifies AppendMany delivers the
// notification to every recipient (the postgres path now does this in a single
// batched transaction; this guards the in-memory mirror used by tests).
func TestInMemoryAppendMany_FansOutToAllUsers(t *testing.T) {
	repo := NewInMemoryRepository()
	users := []string{"u1", "u2", "u3"}
	item := NotifyItem{Type: "mention", Payload: map[string]any{"moment_id": "m1"}}

	if err := repo.AppendMany(context.Background(), users, item); err != nil {
		t.Fatalf("AppendMany failed: %v", err)
	}

	for _, uid := range users {
		page, err := repo.List(context.Background(), uid, NotificationListRequest{})
		if err != nil {
			t.Fatalf("List(%s) failed: %v", uid, err)
		}
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notification, got %d", uid, len(page.Items))
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
		page, _ := repo.List(context.Background(), uid, NotificationListRequest{})
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notification, got %d", uid, len(page.Items))
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
// gets a notification even though the in-memory AppendMany passes the same
// NotifyItem (and thus the same NotifyID) to each Append. The postgres
// batch path now derives a distinct per-row id from a shared prefix (the
// table PK is notification_id alone, so a shared id would collide); this test
// confirms the in-memory fan-out still delivers to all users.
func TestInMemoryAppendMany_DeliversAllWithSharedId(t *testing.T) {
	repo := NewInMemoryRepository()
	users := []string{"u1", "u2"}
	item := NotifyItem{NotifyID: "shared_notif", Type: "mention"}
	if err := repo.AppendMany(context.Background(), users, item); err != nil {
		t.Fatalf("AppendMany failed: %v", err)
	}
	for _, uid := range users {
		page, _ := repo.List(context.Background(), uid, NotificationListRequest{})
		if len(page.Items) != 1 {
			t.Fatalf("user %s: expected 1 notification, got %d", uid, len(page.Items))
		}
	}
}
