package groups

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func covTestLocation() Location {
	return Location{
		Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 35.1, Longitude: 112.2},
		LocationName: "Cov Location",
	}
}

func covCreateTestGroup(t *testing.T, ctx context.Context, repo *PostgresRepository, ownerID, name string, mutate func(*CreateGroupRequest)) GroupItem {
	t.Helper()
	req := CreateGroupRequest{Name: name, Location: covTestLocation()}
	if mutate != nil {
		mutate(&req)
	}
	group, err := repo.Create(ctx, ownerID, req)
	if err != nil {
		t.Fatalf("create group %s: %v", name, err)
	}
	return group
}

// TestCovPostgresListGroupsByUser 覆盖 users 聚合读的 keyset 分页/游标/过滤。
func TestCovPostgresListGroupsByUser(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		owner := testdb.InsertUser(t, txFactory, "covg2_lgbu_owner", "CovOwner")
		user := testdb.InsertUser(t, txFactory, "covg2_lgbu_user", "CovUser")
		stranger := testdb.InsertUser(t, txFactory, "covg2_lgbu_stranger", "Stranger")
		queryable, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no queryable")
		}

		joinedAts := []string{"2026-01-01T00:00:01Z", "2026-01-01T00:00:02Z", "2026-01-01T00:00:03Z"}
		ids := make([]string, 0, 3)
		for i := 0; i < 3; i++ {
			group := covCreateTestGroup(t, ctx, repo, owner, fmt.Sprintf("Cov LGBU %d", i), nil)
			if _, err := repo.JoinGroup(ctx, user, group.Group.GroupID, nil); err != nil {
				t.Fatalf("join group %d: %v", i, err)
			}
			if err := queryable.Exec(ctx, `UPDATE group_members SET joined_at = $3 WHERE group_id = $1 AND user_id = $2`, group.Group.GroupID, user, joinedAts[i]); err != nil {
				t.Fatalf("pin joined_at %d: %v", i, err)
			}
			ids = append(ids, group.Group.GroupID)
		}

		// 不可搜索群：应被过滤；未加入的群：不属于该用户。
		notSearchable := false
		hiddenGroup := covCreateTestGroup(t, ctx, repo, owner, "Cov LGBU Hidden", func(req *CreateGroupRequest) { req.IsSearchable = &notSearchable })
		if _, err := repo.JoinGroup(ctx, user, hiddenGroup.Group.GroupID, nil); err != nil {
			t.Fatalf("join hidden group: %v", err)
		}
		_ = covCreateTestGroup(t, ctx, repo, owner, "Cov LGBU Unjoined", nil)

		page1, err := repo.ListGroupsByUser(ctx, user, 2, "")
		if err != nil {
			t.Fatalf("list page 1: %v", err)
		}
		if len(page1.Items) != 2 {
			t.Fatalf("page 1 size = %d, want 2: %#v", len(page1.Items), page1.Items)
		}
		if page1.Items[0].Group.GroupID != ids[2] || page1.Items[1].Group.GroupID != ids[1] {
			t.Fatalf("page 1 order = %v, want [%s %s]", []string{page1.Items[0].Group.GroupID, page1.Items[1].Group.GroupID}, ids[2], ids[1])
		}
		first := page1.Items[0]
		if first.Group.Name == "" {
			t.Fatalf("unexpected first item: %#v", first)
		}
		// 空描述经 StrPtr 归一为 nil（与 Get 的 COALESCE 口径一致）。
		if first.Description != nil && *first.Description != "" {
			t.Fatalf("unexpected description: %#v", first.Description)
		}
		if first.Location == nil || first.Location.LocationName != "Cov Location" || first.Location.Point.Latitude != 35.1 {
			t.Fatalf("unexpected location: %#v", first.Location)
		}
		if first.JoinMethod != GroupJoinMethodDirect || !first.IsSearchable || first.MaxMembers != 200 {
			t.Fatalf("unexpected join method/flags: %#v", first)
		}
		if first.Group.MembersCount != 2 {
			t.Fatalf("members_count = %d, want 2", first.Group.MembersCount)
		}
		if page1.NextCursor == nil || *page1.NextCursor == "" {
			t.Fatal("expected next cursor on page 1")
		}

		page2, err := repo.ListGroupsByUser(ctx, user, 2, *page1.NextCursor)
		if err != nil {
			t.Fatalf("list page 2: %v", err)
		}
		if len(page2.Items) != 1 || page2.Items[0].Group.GroupID != ids[0] {
			t.Fatalf("page 2 = %#v", page2.Items)
		}
		if page2.NextCursor != nil {
			t.Fatalf("page 2 cursor = %v, want nil", *page2.NextCursor)
		}

		// 非法游标 → ErrInvalidCursor。
		if _, err := repo.ListGroupsByUser(ctx, user, 2, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("invalid cursor err = %v, want ErrInvalidCursor", err)
		}
		// limit 归一化：超过 MaxPageSize 不报错。
		capped, err := repo.ListGroupsByUser(ctx, user, 1000, "")
		if err != nil || len(capped.Items) != 3 {
			t.Fatalf("capped list = %d, %v", len(capped.Items), err)
		}
		// 非成员 → 空列表。
		empty, err := repo.ListGroupsByUser(ctx, stranger, 10, "")
		if err != nil || len(empty.Items) != 0 || empty.NextCursor != nil {
			t.Fatalf("stranger list = %#v, %v", empty, err)
		}

		// 服务层委托。
		svc := NewServiceWithRepo(repo)
		delegated, err := svc.ListGroupsByUser(ctx, user, 2, "")
		if err != nil || len(delegated.Items) != 2 {
			t.Fatalf("service list = %d, %v", len(delegated.Items), err)
		}
		if count, err := svc.OwnedGroupCount(ctx, owner); err != nil || count == 0 {
			t.Fatalf("owned group count = %d, %v", count, err)
		}
	})
}

// TestCovPostgresListGroupsByUserCursorHelpers 覆盖游标编解码的非法输入分支。
func TestCovPostgresListGroupsByUserCursorHelpers(t *testing.T) {
	if got := encodeGroupsByUserCursor(time.Now().UTC(), ""); got == "" {
		t.Fatal("expected non-empty token for empty group id")
	}
	if joinedAt, groupID, err := decodeGroupsByUserCursor(""); err != nil || !joinedAt.IsZero() || groupID != "" {
		t.Fatalf("empty cursor = %v, %q, %v", joinedAt, groupID, err)
	}
	if _, _, err := decodeGroupsByUserCursor("bogus"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bogus cursor err = %v", err)
	}
	// 缺 ID 的载荷 → ErrInvalidCursor。
	token, err := groupsByUserCursorCodec.Encode(groupsByUserCursor{At: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatalf("encode token: %v", err)
	}
	if _, _, err := decodeGroupsByUserCursor(token); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("missing id err = %v", err)
	}
	// 时间不可解析 → ErrInvalidCursor。
	badTimeToken, err := groupsByUserCursorCodec.Encode(groupsByUserCursor{At: "not-a-time", ID: "group_1"})
	if err != nil {
		t.Fatalf("encode bad time token: %v", err)
	}
	if _, _, err := decodeGroupsByUserCursor(badTimeToken); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bad time err = %v", err)
	}
	// 正常回环。
	now := time.Now().UTC().Truncate(time.Microsecond)
	token = encodeGroupsByUserCursor(now, "group_1")
	joinedAt, groupID, err := decodeGroupsByUserCursor(token)
	if err != nil || groupID != "group_1" || !joinedAt.Equal(now) {
		t.Fatalf("roundtrip = %v, %q, %v", joinedAt, groupID, err)
	}
}

// TestCovPostgresGetInviteAndPasswordHash 覆盖 GetInvite 与 GetGroupPasswordHash。
func TestCovPostgresGetInviteAndPasswordHash(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		owner := testdb.InsertUser(t, txFactory, "covg2_inv_owner", "Owner")
		invitee := testdb.InsertUser(t, txFactory, "covg2_inv_invitee", "Invitee")

		group := covCreateTestGroup(t, ctx, repo, owner, "Cov Invite Group", nil)
		message := "welcome"
		invitation, err := repo.InviteMember(ctx, owner, InviteMemberRequest{GroupID: group.Group.GroupID, UserID: invitee, Message: &message}, nil)
		if err != nil {
			t.Fatalf("invite member: %v", err)
		}
		if invitation.InviteID == nil || *invitation.InviteID == "" {
			t.Fatalf("invitation = %#v", invitation)
		}
		item, inviteeID, found, err := repo.GetInvite(ctx, *invitation.InviteID)
		if err != nil || !found {
			t.Fatalf("get invite = %#v, %q, %v, %v", item, inviteeID, found, err)
		}
		if inviteeID != invitee {
			t.Fatalf("invitee = %q, want %q", inviteeID, invitee)
		}
		if item.InviteID != *invitation.InviteID || item.Group.GroupID != group.Group.GroupID {
			t.Fatalf("invite item = %#v", item)
		}
		if item.Message == nil || *item.Message != message {
			t.Fatalf("invite message = %#v", item.Message)
		}
		if item.Status != ApplicationStatusPending || item.Inviter.UserID != owner {
			t.Fatalf("invite status/inviter = %#v", item)
		}
		if _, _, found, err := repo.GetInvite(ctx, "covg2_missing_invite"); err != nil || found {
			t.Fatalf("missing invite found=%v err=%v", found, err)
		}

		// 无密码群 → nil, nil。
		hash, err := repo.GetGroupPasswordHash(ctx, group.Group.GroupID)
		if err != nil || hash != nil {
			t.Fatalf("password hash (none) = %#v, %v", hash, err)
		}
		// 有密码群 → 返回存储哈希。
		passwordHash := "stored-hash"
		pwGroup := covCreateTestGroup(t, ctx, repo, owner, "Cov Password Group", func(req *CreateGroupRequest) {
			req.JoinMethod = GroupJoinMethodPassword
			req.PasswordHash = &passwordHash
		})
		hash, err = repo.GetGroupPasswordHash(ctx, pwGroup.Group.GroupID)
		if err != nil || hash == nil || *hash != passwordHash {
			t.Fatalf("password hash = %#v, %v", hash, err)
		}
		// 群不存在 → 扫描 ErrNoRows 错误。
		if _, err := repo.GetGroupPasswordHash(ctx, "covg2_missing_group"); err == nil {
			t.Fatal("expected error for missing group password hash")
		}
	})
}

// TestCovPostgresDismissAsAdminAndPurgeUserData 覆盖平台解散与注销级联清理。
func TestCovPostgresDismissAsAdminAndPurgeUserData(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		owner := testdb.InsertUser(t, txFactory, "covg2_dis_owner", "Owner")
		member := testdb.InsertUser(t, txFactory, "covg2_dis_member", "Member")
		applicant := testdb.InsertUser(t, txFactory, "covg2_dis_applicant", "Applicant")

		requiresApproval := true
		group := covCreateTestGroup(t, ctx, repo, owner, "Cov Dismiss Group", func(req *CreateGroupRequest) {
			req.RequiresApproval = &requiresApproval
		})
		groupID := group.Group.GroupID
		if _, err := repo.JoinGroup(ctx, member, groupID, nil); err != nil {
			t.Fatalf("join member: %v", err)
		}
		result, err := repo.JoinGroup(ctx, applicant, groupID, nil)
		if err != nil || result.Status != EnterGroupStatusPendingApproval {
			t.Fatalf("pending application = %#v, %v", result, err)
		}
		invitee := testdb.InsertUser(t, txFactory, "covg2_dis_invitee", "Invitee")
		if _, err := repo.InviteMember(ctx, owner, InviteMemberRequest{GroupID: groupID, UserID: invitee}, nil); err != nil {
			t.Fatalf("invite: %v", err)
		}

		hit, err := repo.DismissAsAdmin(ctx, groupID)
		if err != nil || !hit {
			t.Fatalf("dismiss as admin = %v, %v", hit, err)
		}
		if _, ok, err := repo.Get(ctx, groupID); err != nil || ok {
			t.Fatalf("dismissed group still readable: ok=%v err=%v", ok, err)
		}
		// 二次解散：软删条件不再命中 → hit=false。
		hit, err = repo.DismissAsAdmin(ctx, groupID)
		if err != nil || hit {
			t.Fatalf("second dismiss = %v, %v", hit, err)
		}

		// 注销级联：删除该用户的 group_members 行。
		purgeGroup := covCreateTestGroup(t, ctx, repo, owner, "Cov Purge Group", nil)
		purgeUser := testdb.InsertUser(t, txFactory, "covg2_purge_user", "PurgeUser")
		if _, err := repo.JoinGroup(ctx, purgeUser, purgeGroup.Group.GroupID, nil); err != nil {
			t.Fatalf("join purge user: %v", err)
		}
		if err := repo.PurgeUserData(ctx, purgeUser); err != nil {
			t.Fatalf("purge user data: %v", err)
		}
		if member, err := repo.IsMember(ctx, purgeGroup.Group.GroupID, purgeUser); err != nil || member {
			t.Fatalf("purged membership remains: member=%v err=%v", member, err)
		}
		// 幂等：重复清理不报错。
		if err := repo.PurgeUserData(ctx, purgeUser); err != nil {
			t.Fatalf("purge user data (idempotent): %v", err)
		}

		// Service 委托：可清理实现走 repo；非清理实现返回 nil。
		svc := NewServiceWithRepo(repo)
		if err := svc.PurgeUserData(ctx, purgeUser); err != nil {
			t.Fatalf("service purge: %v", err)
		}
		plain := NewServiceWithRepo(&stubGroupsRepo{})
		if err := plain.PurgeUserData(ctx, purgeUser); err != nil {
			t.Fatalf("service purge without purger = %v, want nil", err)
		}
	})
}
