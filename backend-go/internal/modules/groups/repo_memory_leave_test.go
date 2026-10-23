package groups

import (
	"context"
	"testing"
)

// TestInMemoryLeaveGroupRemovesMembership 防回归（审查 P1-6）：内存实现
// LeaveGroup 必须从成员表移除记录（对齐 PG 的 DELETE 语义）。此前只清空
// 存储项上的 ViewerState——cloneGroupForUser 的 ViewerState 恒按 members
// map 重建，该写法无效，退群后 IsMember/ListMembers/MembersCount 仍把
// 用户算作成员。
func TestInMemoryLeaveGroupRemovesMembership(t *testing.T) {
	repo := NewInMemoryRepository()
	created, err := repo.Create(context.Background(), "owner1", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	gid := created.Group.GroupID
	if _, err := repo.JoinGroup(context.Background(), "u2", gid, nil); err != nil {
		t.Fatalf("JoinGroup failed: %v", err)
	}

	if err := repo.LeaveGroup(context.Background(), "u2", gid); err != nil {
		t.Fatalf("LeaveGroup failed: %v", err)
	}

	// 成员表移除：IsMember 必须回落 false。
	isMember, err := repo.IsMember(context.Background(), gid, "u2")
	if err != nil {
		t.Fatalf("IsMember failed: %v", err)
	}
	if isMember {
		t.Fatal("u2 should not be a member after LeaveGroup")
	}

	// Get 视角：退群者 viewer_state 为 nil，群成员数回落为 1（仅 owner）。
	got, ok, err := repo.Get(grpCtx("u2"), gid)
	if err != nil || !ok {
		t.Fatalf("Get failed: ok=%v err=%v", ok, err)
	}
	if got.ViewerState != nil {
		t.Fatalf("expected nil viewer_state after leaving, got %+v", got.ViewerState)
	}
	if got.Group.MembersCount != 1 {
		t.Fatalf("expected members_count=1 after leaving, got %d", got.Group.MembersCount)
	}

	// ListMembers 不再包含退群者。
	members, _, err := repo.ListMembers(context.Background(), gid, nil, 100, "")
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	for _, m := range members {
		if m.User.UserID == "u2" {
			t.Fatal("ListMembers should not include the user after LeaveGroup")
		}
	}
}

// TestInMemoryLeaveGroupNonMemberIsNoop 对齐 PG 语义：对非成员退群
// （DELETE 0 行命中）静默成功，成员资格判定在 service 层完成。
func TestInMemoryLeaveGroupNonMemberIsNoop(t *testing.T) {
	repo := NewInMemoryRepository()
	created, err := repo.Create(context.Background(), "owner1", validCreateReq())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.LeaveGroup(context.Background(), "stranger", created.Group.GroupID); err != nil {
		t.Fatalf("LeaveGroup for non-member should succeed silently, got %v", err)
	}
	isMember, _ := repo.IsMember(context.Background(), created.Group.GroupID, "stranger")
	if isMember {
		t.Fatal("stranger must not become a member via LeaveGroup")
	}
}
