package groups

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func TestPostgresIT_CreateAndGetGroup(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_owner_1", "Owner")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Test Group",
			JoinMethod: GroupJoinMethodOpen,
			Location: Location{LocationName: "Beijing", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 39.9, Longitude: 116.4}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if group.Group.GroupID == "" {
			t.Fatal("expected non-empty group_id")
		}
		if group.Group.Name != "Test Group" {
			t.Fatalf("name: got %q", group.Group.Name)
		}
		if group.OwnerID != ownerID {
			t.Fatalf("owner_id: got %q, want %q", group.OwnerID, ownerID)
		}

		got, found, err := repo.Get(ctx, group.Group.GroupID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if got.Group.GroupID != group.Group.GroupID {
			t.Fatalf("group_id: got %q, want %q", got.Group.GroupID, group.Group.GroupID)
		}
	})
}

func TestPostgresIT_JoinGroupAndListMembers(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_join_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_join_member", "Member")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Join Test Group",
			JoinMethod: GroupJoinMethodOpen,
			Location: Location{LocationName: "Beijing", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 39.9, Longitude: 116.4}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		result, err := repo.JoinGroup(ctx, memberID, group.Group.GroupID, nil, nil)
		if err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}
		if result.Status != JoinGroupStatusJoined {
			t.Fatalf("status: got %q, want %q", result.Status, JoinGroupStatusJoined)
		}

		members, _, err := repo.ListMembers(ctx, group.Group.GroupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		if len(members) != 2 {
			t.Fatalf("expected 2 members, got %d", len(members))
		}
	})
}

func TestPostgresIT_LeaveGroup(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_leave_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_leave_member", "Member")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Leave Test Group",
			JoinMethod: GroupJoinMethodOpen,
			Location: Location{LocationName: "Beijing", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 39.9, Longitude: 116.4}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		if _, err := repo.JoinGroup(ctx, memberID, group.Group.GroupID, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}
		if err := repo.LeaveGroup(ctx, memberID, group.Group.GroupID); err != nil {
			t.Fatalf("LeaveGroup: %v", err)
		}

		members, _, err := repo.ListMembers(ctx, group.Group.GroupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		if len(members) != 1 {
			t.Fatalf("expected 1 member after leave, got %d", len(members))
		}
	})
}

func TestPostgresIT_DismissGroup(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_dismiss_owner", "Owner")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Dismiss Test Group",
			JoinMethod: GroupJoinMethodOpen,
			Location: Location{LocationName: "Beijing", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 39.9, Longitude: 116.4}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		if err := repo.Dismiss(ctx, group.Group.GroupID); err != nil {
			t.Fatalf("Dismiss: %v", err)
		}

		_, found, err := repo.Get(ctx, group.Group.GroupID)
		if err != nil {
			t.Fatalf("Get after dismiss: %v", err)
		}
		if found {
			t.Fatal("expected found=false after dismiss")
		}
	})
}

func TestPostgresIT_SetMemberRole(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_role_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_role_member", "Member")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Role Test Group",
			JoinMethod: GroupJoinMethodOpen,
			Location: Location{LocationName: "Beijing", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 39.9, Longitude: 116.4}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := repo.JoinGroup(ctx, memberID, group.Group.GroupID, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}

		adminRole := GroupRoleAdmin
		if err := repo.SetMemberRole(ctx, ownerID, SetMemberRoleRequest{
			GroupID: group.Group.GroupID,
			UserID:  memberID,
			Role:    adminRole,
		}); err != nil {
			t.Fatalf("SetMemberRole: %v", err)
		}

		members, _, err := repo.ListMembers(ctx, group.Group.GroupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		for _, m := range members {
			if m.User.UserID == memberID && m.Role != GroupRoleAdmin {
				t.Fatalf("role: got %q, want %q", m.Role, GroupRoleAdmin)
			}
		}
	})
}
