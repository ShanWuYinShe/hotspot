package groups

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// helper: 在事务中创建一个测试 group 并返回 groupID 与 ownerID
func createTestGroup(t *testing.T, ctx context.Context, repo *PostgresRepository, ownerID, name string, joinMethod GroupJoinMethod) string {
	t.Helper()
	group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
		Name:       name,
		JoinMethod: joinMethod,
		Location: Location{LocationName: "Beijing", Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 39.9, Longitude: 116.4}},
	})
	if err != nil {
		t.Fatalf("create test group: %v", err)
	}
	return group.Group.GroupID
}

func TestPostgresIT_CreateAndGetGroup(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_owner_1", "Owner")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Test Group",
			JoinMethod: GroupJoinMethodDirect,
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
		if group.OwnerID == nil || *group.OwnerID != ownerID {
			t.Fatalf("owner_id: got %s, want %q", *group.OwnerID, ownerID)
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

// TestPostgresIT_GetToleratesNullDescriptionAndAvatar 回归：description/
// avatar_url 为 NULL 的群行必须可读（COALESCE 兜底），不得报 scan 错误。
func TestPostgresIT_GetToleratesNullDescriptionAndAvatar(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_null_owner", "NullOwner")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Null Desc Group", GroupJoinMethodDirect)
		queryable, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no queryable")
		}
		if err := queryable.Exec(ctx, `UPDATE groups SET description = NULL, avatar_url = NULL WHERE group_id = $1`, groupID); err != nil {
			t.Fatalf("null out columns: %v", err)
		}
		got, found, err := repo.Get(ctx, groupID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		// NULL 经 COALESCE 读成空串：description 走 StrPtr 归一为 nil（与 API
		// 建群空描述形态一致），avatar_url 直扫为指向空串的指针。
		// 关键断言是 Get 不报 scan 错误。
		if got.Description != nil {
			t.Fatalf("description: got %#v, want nil", got.Description)
		}
		if got.Group.AvatarURL == nil || *got.Group.AvatarURL != "" {
			t.Fatalf("avatar_url: got %#v, want empty", got.Group.AvatarURL)
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
			JoinMethod: GroupJoinMethodDirect,
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
		if result.Status != EnterGroupStatusJoined {
			t.Fatalf("status: got %q, want %q", result.Status, EnterGroupStatusJoined)
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
			JoinMethod: GroupJoinMethodDirect,
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
		// 子行填充：成员/公告/封禁/申请/邀请全部就位——解散必须同事务清理，
		// 只删 groups 行会留下幽灵详情（成员数假象）与孤儿子表（线上实测）。
		memberID := testdb.InsertUser(t, txFactory, "g_dismiss_member", "Member")
		applicantID := testdb.InsertUser(t, txFactory, "g_dismiss_applicant", "Applicant")
		inviteeID := testdb.InsertUser(t, txFactory, "g_dismiss_invitee", "Invitee")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Dismiss Test Group",
			JoinMethod: GroupJoinMethodDirect,
			Location: Location{LocationName: "Beijing", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 39.9, Longitude: 116.4}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		if _, err := repo.CreateAnnouncement(ctx, ownerID, CreateGroupAnnouncementRequest{
			GroupID: group.Group.GroupID, Title: "t", Content: "c",
		}); err != nil {
			t.Fatalf("CreateAnnouncement: %v", err)
		}
		if err := repo.BanMember(ctx, ownerID, BanMemberRequest{
			GroupID: group.Group.GroupID, UserID: memberID,
		}); err != nil {
			t.Fatalf("BanMember: %v", err)
		}
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		seed := func(query string, args ...any) {
			t.Helper()
			if err := qdb.Exec(context.Background(), query, args...); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
		seed(`INSERT INTO group_applications (application_id, group_id, applicant_user_id, message, status, created_at)
		      VALUES ('app_dismiss_it', $1, $2, '', 'pending', now())`, group.Group.GroupID, applicantID)
		seed(`INSERT INTO group_invites (invite_id, group_id, inviter_user_id, invitee_user_id, message, status, created_at, expires_at)
		      VALUES ('inv_dismiss_it', $1, $2, $3, '', 'pending', now(), now() + interval '1 day')`, group.Group.GroupID, ownerID, inviteeID)

		// 解散：HTTP 200 但子行必须同事务清空（幽灵详情/假成员数回归断言）
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

		// 子表必须全部清空（幽灵成员数/详情缺陷回归断言）
		for _, table := range []string{
			"group_members", "group_applications", "group_invites",
			"group_bans", "group_announcements",
		} {
			var n int
			if err := qdb.QueryRow(context.Background(),
				"SELECT count(*) FROM "+table+" WHERE group_id = $1",
				group.Group.GroupID).Scan(&n); err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			if n != 0 {
				t.Fatalf("expected %s to be empty after dismiss, got %d rows", table, n)
			}
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
			JoinMethod: GroupJoinMethodDirect,
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

		// 审计 F-05 防线二：SQL 守卫挡住对 owner 行的角色变更（服务层已拒，
		// 此处验证即便绕过服务层、owner 行也不可被降级/改动）。
		memberRole := GroupRoleMember
		if err := repo.SetMemberRole(ctx, ownerID, SetMemberRoleRequest{
			GroupID: group.Group.GroupID,
			UserID:  ownerID,
			Role:    memberRole,
		}); err != nil {
			t.Fatalf("SetMemberRole on owner should be silently guarded, got %v", err)
		}
		members, _, err = repo.ListMembers(ctx, group.Group.GroupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers (post guard): %v", err)
		}
		for _, m := range members {
			if m.User.UserID == ownerID && m.Role != GroupRoleOwner {
				t.Fatalf("owner row was mutated: got %q, want %q", m.Role, GroupRoleOwner)
			}
		}
	})
}

func TestPostgresIT_Update(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_update_owner", "Owner")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Update Test", GroupJoinMethodDirect)

		newName := "Renamed"
		if err := repo.Update(ctx, ownerID, UpdateGroupRequest{
			GroupID: groupID,
			Name:    newName,
		}); err != nil {
			t.Fatalf("Update: %v", err)
		}

		got, found, err := repo.Get(ctx, groupID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if got.Group.Name != newName {
			t.Fatalf("name: got %q, want %q", got.Group.Name, newName)
		}
	})
}

func TestPostgresIT_SearchMembers(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_smem_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_smem_member", "Member")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Search Member Test", GroupJoinMethodDirect)

		if _, err := repo.JoinGroup(ctx, memberID, groupID, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}

		matched, _, err := repo.SearchMembers(ctx, groupID, "owner", nil, 10, "")
		if err != nil {
			t.Fatalf("SearchMembers: %v", err)
		}
		if len(matched) != 1 {
			t.Fatalf("expected 1 matched member, got %d", len(matched))
		}
		if matched[0].User.UserID != ownerID {
			t.Fatalf("matched user_id: got %q, want %q", matched[0].User.UserID, ownerID)
		}
	})
}

func TestPostgresIT_ListMyGroups(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_lmg_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_lmg_member", "Member")

		g1 := createTestGroup(t, ctx, repo, ownerID, "Group A", GroupJoinMethodDirect)
		g2 := createTestGroup(t, ctx, repo, memberID, "Group B", GroupJoinMethodDirect)

		if _, err := repo.JoinGroup(ctx, memberID, g1, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}

		mine, _, err := repo.ListMyGroups(ctx, memberID, 10, "")
		if err != nil {
			t.Fatalf("ListMyGroups: %v", err)
		}
		if len(mine) != 2 {
			t.Fatalf("expected 2 groups for member, got %d", len(mine))
		}
		seenG1, seenG2 := false, false
		for _, it := range mine {
			if it.Group.GroupID == g1 {
				seenG1 = true
			}
			if it.Group.GroupID == g2 {
				seenG2 = true
			}
		}
		if !seenG1 || !seenG2 {
			t.Fatalf("expected to see both groups, seenG1=%v seenG2=%v", seenG1, seenG2)
		}
	})
}

func TestPostgresIT_CreateListUpdateDeleteAnnouncement(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_ann_owner", "Owner")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Announcement Test", GroupJoinMethodDirect)

		created, err := repo.CreateAnnouncement(ctx, ownerID, CreateGroupAnnouncementRequest{
			GroupID: groupID,
			Title:   "Hello",
			Content: "World",
		})
		if err != nil {
			t.Fatalf("CreateAnnouncement: %v", err)
		}
		if created.AnnouncementID == "" {
			t.Fatal("expected non-empty announcement id")
		}

		items, err := repo.ListAnnouncements(ctx, groupID)
		if err != nil {
			t.Fatalf("ListAnnouncements: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("expected 1 announcement, got %d", len(items))
		}
		if items[0].Title != "Hello" {
			t.Fatalf("title: got %q", items[0].Title)
		}

		updatedTitle := "Hello2"
		pinned := true
		if err := repo.UpdateAnnouncement(ctx, groupID, UpdateGroupAnnouncementRequest{
			GroupID:        groupID,
			AnnouncementID: created.AnnouncementID,
			Title:          &updatedTitle,
			IsPinned:       &pinned,
		}); err != nil {
			t.Fatalf("UpdateAnnouncement: %v", err)
		}
		items, err = repo.ListAnnouncements(ctx, groupID)
		if err != nil {
			t.Fatalf("ListAnnouncements (post update): %v", err)
		}
		if items[0].Title != "Hello2" {
			t.Fatalf("updated title: got %q, want %q", items[0].Title, "Hello2")
		}
		if !items[0].IsPinned {
			t.Fatal("expected IsPinned=true after update")
		}

		if err := repo.DeleteAnnouncement(ctx, groupID, created.AnnouncementID); err != nil {
			t.Fatalf("DeleteAnnouncement: %v", err)
		}
		items, err = repo.ListAnnouncements(ctx, groupID)
		if err != nil {
			t.Fatalf("ListAnnouncements (post delete): %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("expected 0 announcements after delete, got %d", len(items))
		}
	})
}

func TestPostgresIT_ApplicationApproveReject(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_app_owner", "Owner")
		applicantID := testdb.InsertUser(t, txFactory, "g_app_applicant", "Applicant")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Application Test", GroupJoinMethodDirect)
		approval := true
		if err := repo.Update(ctx, ownerID, UpdateGroupRequest{GroupID: groupID, RequiresApproval: &approval}); err != nil {
			t.Fatalf("set requires_approval: %v", err)
		}

		msg := "please"
		result, err := repo.JoinGroup(ctx, applicantID, groupID, nil, &msg)
		if err != nil {
			t.Fatalf("JoinGroup (approval): %v", err)
		}
		if result.Status != EnterGroupStatusPendingApproval {
			t.Fatalf("status: got %q, want %q", result.Status, EnterGroupStatusPendingApproval)
		}
		if result.ApplicationID == nil || *result.ApplicationID == "" {
			t.Fatal("expected non-empty application id")
		}
		applicationID := result.ApplicationID

		pending := ApplicationStatusPending
		apps, _, err := repo.ListApplications(ctx, groupID, &pending, 10, "")
		if err != nil {
			t.Fatalf("ListApplications: %v", err)
		}
		if len(apps) != 1 {
			t.Fatalf("expected 1 pending application, got %d", len(apps))
		}
		// 申请人昵称必须解析（此前占位填 user_id，管理收件箱直接展示裸 id）。
		if apps[0].Applicant.Nickname != "Applicant" {
			t.Fatalf("applicant nickname: got %q, want %q", apps[0].Applicant.Nickname, "Applicant")
		}

		if err := repo.ApproveApplication(ctx, ownerID, *applicationID); err != nil {
			t.Fatalf("ApproveApplication: %v", err)
		}
		members, _, err := repo.ListMembers(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers (post approve): %v", err)
		}
		found := false
		for _, m := range members {
			if m.User.UserID == applicantID {
				found = true
			}
		}
		if !found {
			t.Fatal("expected applicant to be a member after approval")
		}

		// Create a second application and reject it
		applicant2ID := testdb.InsertUser(t, txFactory, "g_app_applicant2", "Applicant2")
		result2, err := repo.JoinGroup(ctx, applicant2ID, groupID, nil, nil)
		if err != nil {
			t.Fatalf("JoinGroup (2): %v", err)
		}
		if result2.ApplicationID == nil {
			t.Fatal("expected application to be created for 2nd applicant")
		}
		if err := repo.RejectApplication(ctx, *result2.ApplicationID, nil); err != nil {
			t.Fatalf("RejectApplication: %v", err)
		}
		apps, _, err = repo.ListApplications(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListApplications (post reject): %v", err)
		}
		var rejected *GroupApplicationItem
		for i, a := range apps {
			if a.Applicant.UserID == applicant2ID {
				rejected = &apps[i]
				break
			}
		}
		if rejected == nil {
			t.Fatal("expected rejected application in list")
		}
		if rejected.Status != ApplicationStatusRejected {
			t.Fatalf("status: got %q, want %q", rejected.Status, ApplicationStatusRejected)
		}
	})
}

func TestPostgresIT_UpdatePreferences(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_set_owner", "Owner")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Settings Test", GroupJoinMethodDirect)

		muted := true
		pinned := true
		if err := repo.UpdatePreferences(ctx, ownerID, UpdateGroupPreferencesRequest{
			GroupID:  groupID,
			IsMuted:  &muted,
			IsPinned: &pinned,
		}); err != nil {
			t.Fatalf("UpdatePreferences: %v", err)
		}

		muted = false
		if err := repo.UpdatePreferences(ctx, ownerID, UpdateGroupPreferencesRequest{
			GroupID: groupID,
			IsMuted: &muted,
		}); err != nil {
			t.Fatalf("UpdatePreferences (toggle): %v", err)
		}
	})
}

// 审计 F-04 收口：服务层 IsBanned 与审批事务提交之间存在竞态窗口，审批
// 事务内必须复检封禁——已封禁申请人的审批整体失败（申请保持 pending、
// 不产生成员行）。
func TestPostgresIT_ApproveApplicationBannedInTx(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_banapp_owner", "Owner")
		applicantID := testdb.InsertUser(t, txFactory, "g_banapp_applicant", "Applicant")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Banned Approve Test", GroupJoinMethodDirect)
		approval := true
		if err := repo.Update(ctx, ownerID, UpdateGroupRequest{GroupID: groupID, RequiresApproval: &approval}); err != nil {
			t.Fatalf("set requires_approval: %v", err)
		}

		result, err := repo.JoinGroup(ctx, applicantID, groupID, nil, nil)
		if err != nil {
			t.Fatalf("JoinGroup (approval): %v", err)
		}
		if result.ApplicationID == nil || *result.ApplicationID == "" {
			t.Fatal("expected non-empty application id")
		}

		// 服务层检查之后、审批事务之前，申请人被封禁（模拟竞态窗口）。
		// 双防线生效顺序：审批事务内的封禁复检先行，返回精确原因
		// ErrBannedInGroup；即便复检被绕过，级联撤销 + pending 守卫也会让
		// 审批以 ErrAlreadyProcessed 失败——已封禁申请人不可能被通过。
		if err := repo.BanMember(ctx, ownerID, BanMemberRequest{GroupID: groupID, UserID: applicantID}); err != nil {
			t.Fatalf("BanMember: %v", err)
		}

		if err := repo.ApproveApplication(ctx, ownerID, *result.ApplicationID); !errors.Is(err, ErrBannedInGroup) {
			t.Fatalf("ApproveApplication on banned applicant: got %v, want ErrBannedInGroup", err)
		}

		// 申请保持非 pending（已被级联置 revoked），且未产生成员行
		apps, _, err := repo.ListApplications(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListApplications: %v", err)
		}
		found := false
		for _, a := range apps {
			if a.ApplicationID == *result.ApplicationID {
				found = true
				if a.Status != ApplicationStatusRevoked {
					t.Fatalf("application status: got %q, want %q", a.Status, ApplicationStatusRevoked)
				}
			}
		}
		if !found {
			t.Fatal("application record should still exist (revoked by cascade)")
		}
		members, _, err := repo.ListMembers(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		for _, m := range members {
			if m.User.UserID == applicantID {
				t.Fatal("banned applicant must not become a member")
			}
		}
	})
}

func TestPostgresIT_ListMyApplicationsAndCancel(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_lma_owner", "Owner")
		applicantID := testdb.InsertUser(t, txFactory, "g_lma_applicant", "Applicant")
		groupID := createTestGroup(t, ctx, repo, ownerID, "MyApps Test", GroupJoinMethodDirect)
		approval := true
		if err := repo.Update(ctx, ownerID, UpdateGroupRequest{GroupID: groupID, RequiresApproval: &approval}); err != nil {
			t.Fatalf("set requires_approval: %v", err)
		}

		result, err := repo.JoinGroup(ctx, applicantID, groupID, nil, nil)
		if err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}
		if result.ApplicationID == nil {
			t.Fatal("expected application to be created")
		}

		pending := ApplicationStatusPending
		apps, _, err := repo.ListMyApplications(ctx, applicantID, &pending, 10, "")
		if err != nil {
			t.Fatalf("ListMyApplications: %v", err)
		}
		if len(apps) != 1 {
			t.Fatalf("expected 1 application, got %d", len(apps))
		}

		if err := repo.CancelApplication(ctx, *result.ApplicationID); err != nil {
			t.Fatalf("CancelApplication: %v", err)
		}
		apps, _, err = repo.ListMyApplications(ctx, applicantID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMyApplications (post cancel): %v", err)
		}
		if len(apps) != 1 {
			t.Fatalf("expected 1 application, got %d", len(apps))
		}
		if apps[0].Status != ApplicationStatusRevoked {
			t.Fatalf("status: got %q, want %q", apps[0].Status, ApplicationStatusRevoked)
		}
	})
}

func TestPostgresIT_InviteFlow(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_inv_owner", "Owner")
		inviteeID := testdb.InsertUser(t, txFactory, "g_inv_invitee", "Invitee")
		invitee2ID := testdb.InsertUser(t, txFactory, "g_inv_invitee2", "Invitee2")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Invite Test", GroupJoinMethodDirect)

		msg := "welcome"
		inv, err := repo.InviteMember(ctx, ownerID, InviteMemberRequest{
			GroupID: groupID,
			UserID:  inviteeID,
			Message: &msg,
		}, nil)
		if err != nil {
			t.Fatalf("InviteMember: %v", err)
		}
		if inv.InviteID == nil || *inv.InviteID == "" {
			t.Fatal("expected non-empty invite id")
		}

		pending := ApplicationStatusPending
		sent, _, err := repo.ListSentInvites(ctx, groupID, &pending, 10, "")
		if err != nil {
			t.Fatalf("ListSentInvites: %v", err)
		}
		if len(sent) != 1 {
			t.Fatalf("expected 1 sent invite, got %d", len(sent))
		}
		if sent[0].Invitee.UserID != inviteeID {
			t.Fatalf("invitee: got %q, want %q", sent[0].Invitee.UserID, inviteeID)
		}

		// Duplicate invite should return the same pending invite id
		inv2, err := repo.InviteMember(ctx, ownerID, InviteMemberRequest{
			GroupID: groupID,
			UserID:  inviteeID,
		}, nil)
		if err != nil {
			t.Fatalf("InviteMember (duplicate): %v", err)
		}
		if *inv2.InviteID != *inv.InviteID {
			t.Fatalf("expected deduped invite id, got %q vs %q", *inv2.InviteID, *inv.InviteID)
		}

		// Invitee sees the invite
		myInvites, _, err := repo.ListMyInvites(ctx, inviteeID, &pending, 10, "")
		if err != nil {
			t.Fatalf("ListMyInvites: %v", err)
		}
		if len(myInvites) != 1 {
			t.Fatalf("expected 1 invite, got %d", len(myInvites))
		}
		if myInvites[0].Group.GroupID != groupID {
			t.Fatalf("group id: got %q, want %q", myInvites[0].Group.GroupID, groupID)
		}
		// 邀请人昵称必须解析（此前占位填 user_id，通知待办区块直接展示裸 id）。
		if myInvites[0].Inviter.Nickname != "Owner" {
			t.Fatalf("inviter nickname: got %q, want %q", myInvites[0].Inviter.Nickname, "Owner")
		}

		// Accept the invite
		if err := repo.AcceptInvite(ctx, inviteeID, *inv.InviteID); err != nil {
			t.Fatalf("AcceptInvite: %v", err)
		}
		members, _, err := repo.ListMembers(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers (post accept): %v", err)
		}
		found := false
		for _, m := range members {
			if m.User.UserID == inviteeID {
				found = true
			}
		}
		if !found {
			t.Fatal("expected invitee to be a member after accept")
		}

		// Create another invite and reject it
		msg2 := "hi"
		inv3, err := repo.InviteMember(ctx, ownerID, InviteMemberRequest{
			GroupID: groupID,
			UserID:  invitee2ID,
			Message: &msg2,
		}, nil)
		if err != nil {
			t.Fatalf("InviteMember (2): %v", err)
		}
		if err := repo.RejectInvite(ctx, *inv3.InviteID); err != nil {
			t.Fatalf("RejectInvite: %v", err)
		}
		myInvites, _, err = repo.ListMyInvites(ctx, invitee2ID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMyInvites (post reject): %v", err)
		}
		if len(myInvites) != 1 {
			t.Fatalf("expected 1 invite, got %d", len(myInvites))
		}
		if myInvites[0].Status != ApplicationStatusRejected {
			t.Fatalf("status: got %q, want %q", myInvites[0].Status, ApplicationStatusRejected)
		}
	})
}

func TestPostgresIT_RevokeInvite(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_revoke_owner", "Owner")
		inviteeID := testdb.InsertUser(t, txFactory, "g_revoke_invitee", "Invitee")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Revoke Test", GroupJoinMethodDirect)

		inv, err := repo.InviteMember(ctx, ownerID, InviteMemberRequest{
			GroupID: groupID,
			UserID:  inviteeID,
		}, nil)
		if err != nil {
			t.Fatalf("InviteMember: %v", err)
		}

		if err := repo.RevokeInvite(ctx, groupID, *inv.InviteID); err != nil {
			t.Fatalf("RevokeInvite: %v", err)
		}
		pending := ApplicationStatusPending
		sent, _, err := repo.ListSentInvites(ctx, groupID, &pending, 10, "")
		if err != nil {
			t.Fatalf("ListSentInvites (post revoke): %v", err)
		}
		if len(sent) != 0 {
			t.Fatalf("expected 0 pending invites after revoke, got %d", len(sent))
		}
	})
}

func TestPostgresIT_MuteUnmuteMember(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_mute_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_mute_member", "Member")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Mute Test", GroupJoinMethodDirect)

		if _, err := repo.JoinGroup(ctx, memberID, groupID, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}

		if err := repo.MuteMember(ctx, ownerID, MuteMemberRequest{
			GroupID:         groupID,
			UserID:          memberID,
			DurationMinutes: 60,
		}); err != nil {
			t.Fatalf("MuteMember: %v", err)
		}

		if err := repo.UnmuteMember(ctx, ownerID, UnmuteMemberRequest{
			GroupID: groupID,
			UserID:  memberID,
		}); err != nil {
			t.Fatalf("UnmuteMember: %v", err)
		}
	})
}

func TestPostgresIT_BanUnbanAndListBanned(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_ban_owner", "Owner")
		badMemberID := testdb.InsertUser(t, txFactory, "g_ban_member", "Bad")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Ban Test", GroupJoinMethodDirect)

		if _, err := repo.JoinGroup(ctx, badMemberID, groupID, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}

		reason := "spam"
		if err := repo.BanMember(ctx, ownerID, BanMemberRequest{
			GroupID: groupID,
			UserID:  badMemberID,
			Reason:  &reason,
		}); err != nil {
			t.Fatalf("BanMember: %v", err)
		}

		members, _, err := repo.ListMembers(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers (post ban): %v", err)
		}
		for _, m := range members {
			if m.User.UserID == badMemberID {
				t.Fatal("banned member should not appear in member list")
			}
		}

		banned, _, err := repo.ListBannedMembers(ctx, groupID, 10, "")
		if err != nil {
			t.Fatalf("ListBannedMembers: %v", err)
		}
		if len(banned) != 1 {
			t.Fatalf("expected 1 banned member, got %d", len(banned))
		}
		if banned[0].User.UserID != badMemberID {
			t.Fatalf("banned user_id: got %q, want %q", banned[0].User.UserID, badMemberID)
		}
		// 被封禁者与操作者昵称必须解析（此前占位填 user_id，名单直接展示裸 id）。
		if banned[0].User.Nickname != "Bad" {
			t.Fatalf("banned nickname: got %q, want %q", banned[0].User.Nickname, "Bad")
		}
		if banned[0].BannedBy == nil || banned[0].BannedBy.Nickname != "Owner" {
			t.Fatalf("banned_by nickname: got %#v, want Owner", banned[0].BannedBy)
		}

		if err := repo.UnbanMember(ctx, ownerID, UnbanMemberRequest{
			GroupID: groupID,
			UserID:  badMemberID,
		}); err != nil {
			t.Fatalf("UnbanMember: %v", err)
		}
		banned, _, err = repo.ListBannedMembers(ctx, groupID, 10, "")
		if err != nil {
			t.Fatalf("ListBannedMembers (post unban): %v", err)
		}
		if len(banned) != 0 {
			t.Fatalf("expected 0 banned members after unban, got %d", len(banned))
		}
	})
}

func TestPostgresIT_RemoveMember(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_rm_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_rm_member", "Member")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Remove Test", GroupJoinMethodDirect)

		if _, err := repo.JoinGroup(ctx, memberID, groupID, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}

		if err := repo.RemoveMember(ctx, ownerID, RemoveMemberRequest{
			GroupID: groupID,
			UserID:  memberID,
		}); err != nil {
			t.Fatalf("RemoveMember: %v", err)
		}

		members, _, err := repo.ListMembers(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers (post remove): %v", err)
		}
		if len(members) != 1 {
			t.Fatalf("expected 1 member after remove, got %d", len(members))
		}
	})
}

func TestPostgresIT_TransferOwnership(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_to_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "g_to_member", "Member")
		groupID := createTestGroup(t, ctx, repo, ownerID, "Transfer Test", GroupJoinMethodDirect)

		if _, err := repo.JoinGroup(ctx, memberID, groupID, nil, nil); err != nil {
			t.Fatalf("JoinGroup: %v", err)
		}

		if err := repo.TransferOwnership(ctx, ownerID, TransferOwnershipRequest{
			GroupID:    groupID,
			NewOwnerID: memberID,
		}); err != nil {
			t.Fatalf("TransferOwnership: %v", err)
		}

		got, found, err := repo.Get(ctx, groupID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if got.OwnerID == nil || *got.OwnerID != memberID {
			t.Fatalf("owner_id: got %s, want %q", *got.OwnerID, memberID)
		}

		members, _, err := repo.ListMembers(ctx, groupID, nil, 10, "")
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		var ownerRole, newOwnerRole *GroupRole
		for _, m := range members {
			r := m.Role
			if m.User.UserID == ownerID {
				ownerRole = &r
			}
			if m.User.UserID == memberID {
				newOwnerRole = &r
			}
		}
		if ownerRole == nil || *ownerRole != GroupRoleAdmin {
			t.Fatalf("previous owner role: got %v, want %q", ownerRole, GroupRoleAdmin)
		}
		if newOwnerRole == nil || *newOwnerRole != GroupRoleOwner {
			t.Fatalf("new owner role: got %v, want %q", newOwnerRole, GroupRoleOwner)
		}
	})
}

func TestPostgresIT_CountOwnedGroups(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "u_cog_owner", "Owner")
		memberID := testdb.InsertUser(t, txFactory, "u_cog_member", "Member")

		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "Owned Count Group",
			JoinMethod: GroupJoinMethodDirect,
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

		count, err := repo.CountOwnedGroups(ctx, ownerID)
		if err != nil {
			t.Fatalf("CountOwnedGroups owner: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 owned group, got %d", count)
		}
		count, err = repo.CountOwnedGroups(ctx, memberID)
		if err != nil {
			t.Fatalf("CountOwnedGroups member: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 owned groups for member, got %d", count)
		}
	})
}

func TestPostgresIT_CommonGroupCount(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_cgc_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_cgc_bob", "Bob")

		count, err := repo.CountCommonGroups(ctx, alice, bob)
		if err != nil {
			t.Fatalf("CommonGroupCount: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 common groups, got %d", count)
		}
	})
}

func TestPostgresIT_SearchGroups_S2SpatialFiltering(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_s2_owner", "Owner")

		falseVal := false
		trueVal := true

		// 1. 附近可见群组 A (~290m)
		gA, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:         "Golang Beijing Near",
			IsSearchable: &trueVal,
			Location: Location{LocationName: "Loc", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 42.1260, Longitude: 88.5678}},
		})
		if err != nil {
			t.Fatalf("create gA: %v", err)
		}

		// 2. 附近可见群组 B (~1300m)
		gB, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:         "Rust Beijing Medium",
			IsSearchable: &trueVal,
			Location: Location{LocationName: "Loc", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 42.1350, Longitude: 88.5678}},
		})
		if err != nil {
			t.Fatalf("create gB: %v", err)
		}

		// 3. 附近不可搜索群组 C
		gC, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:         "Hidden Group",
			IsSearchable: &falseVal,
			Location: Location{LocationName: "Loc", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 42.1260, Longitude: 88.5678}},
		})
		if err != nil {
			t.Fatalf("create gC: %v", err)
		}

		// 4. 附近解散软删除群组 D
		gD, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:         "Dismissed Group",
			IsSearchable: &trueVal,
			Location: Location{LocationName: "Loc", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 42.1260, Longitude: 88.5678}},
		})
		if err != nil {
			t.Fatalf("create gD: %v", err)
		}
		if err := repo.Dismiss(ctx, gD.Group.GroupID); err != nil {
			t.Fatalf("dismiss gD: %v", err)
		}

		// 5. 远距离群组 E (上海 ~3000km)
		gE, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:         "Shanghai Group Far",
			IsSearchable: &trueVal,
			Location: Location{LocationName: "Shanghai", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 31.2304, Longitude: 121.4737}},
		})
		if err != nil {
			t.Fatalf("create gE: %v", err)
		}

		_ = gC
		_ = gE

		// 检索以参考点为中心、半径 5000m 内的群组，按距离排序
		radius := 5000
		resp, err := repo.Search(ctx, SearchGroupsRequest{
			ReferencePoint: GeoPoint{Latitude: 42.1234, Longitude: 88.5678},
			Radius:         &radius,
			SortBy:         "distance",
		})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}

		if len(resp.Items) != 2 {
			t.Fatalf("expected exactly 2 search results, got %d: %#v", len(resp.Items), resp.Items)
		}

		foundIDs := make(map[string]bool)
		for _, item := range resp.Items {
			foundIDs[item.Group.GroupID] = true
			if item.Location == nil || item.Location.Distance == nil {
				t.Fatalf("expected relative distance populated for group %s", item.Group.GroupID)
			}
		}

		if !foundIDs[gA.Group.GroupID] || !foundIDs[gB.Group.GroupID] {
			t.Fatalf("expected gA and gB in results: %#v", foundIDs)
		}
		if foundIDs[gC.Group.GroupID] {
			t.Fatal("unsearchable group gC leaked into search results")
		}
		if foundIDs[gD.Group.GroupID] {
			t.Fatal("dismissed group gD leaked into search results")
		}
		if foundIDs[gE.Group.GroupID] {
			t.Fatal("far group gE leaked into 5km search results")
		}

		// 验证首项为更近的 gA
		if resp.Items[0].Group.GroupID != gA.Group.GroupID {
			t.Fatalf("expected gA as first result due to closer distance, got %s", resp.Items[0].Group.GroupID)
		}
		if *resp.Items[0].Location.Distance >= *resp.Items[1].Location.Distance {
			t.Fatalf("expected items sorted by distance: dist(gA)=%.1f >= dist(gB)=%.1f",
				*resp.Items[0].Location.Distance, *resp.Items[1].Location.Distance)
		}
	})
}

func TestPostgresIT_SearchGroups_KeywordAndSpatialFilter(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		ownerID := testdb.InsertUser(t, txFactory, "g_kw_owner", "Owner")

		trueVal := true
		// 附近匹配关键字
		gMatch, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:         "HotSpot Enthusiasts Beijing",
			IsSearchable: &trueVal,
			Location: Location{LocationName: "Loc", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 42.1260, Longitude: 88.5678}},
		})
		if err != nil {
			t.Fatalf("create gMatch: %v", err)
		}

		// 附近但不匹配关键字
		_, err = repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:         "Outdoor Hikers Beijing",
			IsSearchable: &trueVal,
			Location: Location{LocationName: "Loc", Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 42.1260, Longitude: 88.5678}},
		})
		if err != nil {
			t.Fatalf("create gOther: %v", err)
		}

		radius := 5000
		resp, err := repo.Search(ctx, SearchGroupsRequest{
			ReferencePoint: GeoPoint{Latitude: 42.1234, Longitude: 88.5678},
			Radius:         &radius,
			Keyword:        "HotSpot",
			SortBy:         "relevance",
		})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}

		if len(resp.Items) != 1 || resp.Items[0].Group.GroupID != gMatch.Group.GroupID {
			t.Fatalf("expected only gMatch in keyword search, got %#v", resp.Items)
		}
	})
}
