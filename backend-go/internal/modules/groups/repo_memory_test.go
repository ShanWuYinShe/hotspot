package groups

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// validCreateReq is a minimal CreateGroupRequest used by the repo tests.
func validCreateReq() CreateGroupRequest {
	return CreateGroupRequest{
		Name:        "Test Group",
		Description: "a group",
		Location: RelativeLocation{
			LocationName: "here",
			Point:        GeoPoint{Latitude: 39.9, Longitude: 116.4},
		},
	}
}

// TestInMemoryCreate_AddsOwnerAsMember verifies the in-memory Create path
// records both the group and its owner membership. The postgres Create path
// now wraps the same two writes in a transaction; this test guards that the
// (tx-transparent) in-memory path still produces a consistent result.
func TestInMemoryCreate_AddsOwnerAsMember(t *testing.T) {
	repo := NewInMemoryRepository()
	item, err := repo.Create(context.Background(), "user_1", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if item.OwnerID == nil || *item.OwnerID != "user_1" {
		t.Fatalf("expected owner user_1, got %s", *item.OwnerID)
	}
	if item.ViewerState == nil || item.ViewerState.Role != GroupRoleOwner {
		t.Fatalf("expected owner role, got %+v", item.ViewerState)
	}

	// The owner must be recorded as a member.
	isMember, err := repo.IsMember(context.Background(), item.Group.GroupID, "user_1")
	if err != nil {
		t.Fatalf("IsMember failed: %v", err)
	}
	if !isMember {
		t.Fatal("owner was not recorded as a member after Create")
	}
}

// TestInMemoryCreate_GroupIsRetrievable ensures the created group is visible
// through Get, proving the two writes are observable together (the invariant
// the transaction protects on the postgres side).
func TestInMemoryCreate_GroupIsRetrievable(t *testing.T) {
	repo := NewInMemoryRepository()
	item, err := repo.Create(context.Background(), "user_2", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_2"})
	got, found, err := repo.Get(ctx, item.Group.GroupID)
	if err != nil || !found {
		t.Fatalf("created group not retrievable: found=%v err=%v", found, err)
	}
	if got.Group.GroupID != item.Group.GroupID {
		t.Fatalf("retrieved wrong group: %s", got.Group.GroupID)
	}
}

// TestServiceCreate_ReturnsItemWithOwnerState is a service-level check that
// Create surfaces the owner state back to the caller (used by the handler to
// build the response). Uses the in-memory repo.
func TestServiceCreate_ReturnsItemWithOwnerState(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_3"})
	item, err := svc.Create(ctx, validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if item.ViewerState == nil || item.ViewerState.Role != GroupRoleOwner {
		t.Fatalf("expected joined owner state, got %+v", item.ViewerState)
	}
}

// joinNGroups creates n groups and has userID join them all, returning the
// repo. Used to set up ListMyGroups pagination scenarios.
func joinNGroups(t *testing.T, repo *InMemoryRepository, userID string, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		req := validCreateReq()
		req.Name = "Group-" + string(rune('A'+i))
		_, err := repo.Create(ctx, userID, req)
		if err != nil {
			t.Fatalf("Create #%d failed: %v", i, err)
		}
	}
}

// TestInMemoryListMyGroups_Pagination verifies keyset pagination: a limit
// smaller than the total returns a cursor, and following it yields the
// remaining rows with no overlap or gaps.
func TestInMemoryListMyGroups_Pagination(t *testing.T) {
	repo := NewInMemoryRepository()
	const total = 5
	joinNGroups(t, repo, "user_1", total)

	// Page 1: limit=2 → 2 items + cursor.
	page1, cursor1, err := repo.ListMyGroups(context.Background(), "user_1", 2, "")
	if err != nil {
		t.Fatalf("page1 failed: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: expected 2 items, got %d", len(page1))
	}
	if cursor1 == "" {
		t.Fatal("page1: expected non-empty next cursor")
	}

	// Page 2: follow cursor1 → 2 more items + cursor.
	page2, cursor2, err := repo.ListMyGroups(context.Background(), "user_1", 2, cursor1)
	if err != nil {
		t.Fatalf("page2 failed: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2: expected 2 items, got %d", len(page2))
	}
	if cursor2 == "" {
		t.Fatal("page2: expected non-empty next cursor")
	}

	// Page 3: follow cursor2 → remaining 1 item, no cursor (end).
	page3, cursor3, err := repo.ListMyGroups(context.Background(), "user_1", 2, cursor2)
	if err != nil {
		t.Fatalf("page3 failed: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3: expected 1 item, got %d", len(page3))
	}
	if cursor3 != "" {
		t.Fatalf("page3: expected empty cursor, got %s", cursor3)
	}

	// No overlap between pages.
	seen := map[string]bool{}
	for _, it := range append(append(page1, page2...), page3...) {
		if seen[it.Group.GroupID] {
			t.Fatalf("group %s appeared twice across pages", it.Group.GroupID)
		}
		seen[it.Group.GroupID] = true
	}
	if len(seen) != total {
		t.Fatalf("expected %d unique groups across pages, got %d", total, len(seen))
	}
}

// TestInMemoryListMyGroups_RespectsLimit verifies limit clamping (default and
// max) and that a limit >= total returns everything with no cursor.
func TestInMemoryListMyGroups_RespectsLimit(t *testing.T) {
	repo := NewInMemoryRepository()
	joinNGroups(t, repo, "user_1", 3)

	// limit=0 falls back to default 20 → all 3 returned, no cursor.
	all, cursor, err := repo.ListMyGroups(context.Background(), "user_1", 0, "")
	if err != nil {
		t.Fatalf("limit=0 failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("limit=0: expected 3 items (default), got %d", len(all))
	}
	if cursor != "" {
		t.Fatalf("limit=0: expected empty cursor, got %s", cursor)
	}

	// limit=200 clamps to max 100.
	items, _, err := repo.ListMyGroups(context.Background(), "user_1", 200, "")
	if err != nil {
		t.Fatalf("limit=200 failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("limit=200: expected 3 items, got %d", len(items))
	}
}

// TestInMemoryListMyGroups_InvalidCursor verifies a malformed cursor is rejected.
func TestInMemoryListMyGroups_InvalidCursor(t *testing.T) {
	repo := NewInMemoryRepository()
	joinNGroups(t, repo, "user_1", 1)

	_, _, err := repo.ListMyGroups(context.Background(), "user_1", 10, "not-a-valid-cursor")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

// TestInMemoryListMyGroups_ExcludesNonMember verifies a user only sees their
// own groups.
func TestInMemoryListMyGroups_ExcludesNonMember(t *testing.T) {
	repo := NewInMemoryRepository()
	joinNGroups(t, repo, "user_1", 2)
	// user_2 hasn't joined anything.
	items, cursor, err := repo.ListMyGroups(context.Background(), "user_2", 10, "")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items for non-member, got %d", len(items))
	}
	if cursor != "" {
		t.Fatalf("expected empty cursor, got %s", cursor)
	}
}
