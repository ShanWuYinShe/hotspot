package groups

import (
	"context"
	"errors"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestCovPostgresRepositoryWithoutDB 校验「DB 未配置时优雅降级」契约：
// 所有仓储方法返回 ErrNotImplemented（或空结果），不得 panic。
func TestCovPostgresRepositoryWithoutDB(t *testing.T) {
	repo := NewPostgresRepository(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := repo.Create(ctx, "u1", CreateGroupRequest{Name: "g"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Create err = %v", err)
	}
	if _, _, err := repo.Get(ctx, "g1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Get err = %v", err)
	}
	if _, err := repo.Search(ctx, SearchGroupsRequest{Keyword: "k"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Search err = %v", err)
	}
	if err := repo.Update(ctx, "u1", UpdateGroupRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Update err = %v", err)
	}
	if err := repo.Dismiss(ctx, "g1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Dismiss err = %v", err)
	}
	if _, err := repo.DismissAsAdmin(ctx, "g1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("DismissAsAdmin err = %v", err)
	}
	if _, _, err := repo.ListMembers(ctx, "g1", nil, 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListMembers err = %v", err)
	}
	if _, _, err := repo.ListAdminMembers(ctx, "g1", nil, 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListAdminMembers err = %v", err)
	}
	if _, _, err := repo.SearchMembers(ctx, "g1", "k", nil, 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("SearchMembers err = %v", err)
	}
	if _, err := repo.IsMember(ctx, "g1", "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("IsMember err = %v", err)
	}
	if _, err := repo.AreMembers(ctx, []string{"g1"}, "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("AreMembers err = %v", err)
	}
	if empty, err := repo.AreMembers(ctx, nil, "u1"); err != nil || len(empty) != 0 {
		t.Fatalf("AreMembers(empty) = %v, %v", empty, err)
	}
	if _, err := repo.IsBanned(ctx, "g1", "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("IsBanned err = %v", err)
	}
	if _, _, err := repo.GetApplication(ctx, "a1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetApplication err = %v", err)
	}
	if _, _, err := repo.ListMyGroups(ctx, "u1", 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListMyGroups err = %v", err)
	}
	if _, err := repo.JoinGroup(ctx, "u1", "g1", nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("JoinGroup err = %v", err)
	}
	if _, err := repo.GetGroupPasswordHash(ctx, "g1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetGroupPasswordHash err = %v", err)
	}
	if _, err := repo.CreateAnnouncement(ctx, "u1", CreateGroupAnnouncementRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("CreateAnnouncement err = %v", err)
	}
	if _, err := repo.ListAnnouncements(ctx, "g1", 10); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListAnnouncements err = %v", err)
	}
	if _, _, err := repo.GetAnnouncement(ctx, "g1", "a1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetAnnouncement err = %v", err)
	}
	if err := repo.UpdateAnnouncement(ctx, "u1", UpdateGroupAnnouncementRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("UpdateAnnouncement err = %v", err)
	}
	if err := repo.DeleteAnnouncement(ctx, "g1", "a1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("DeleteAnnouncement err = %v", err)
	}
	if _, _, err := repo.ListApplications(ctx, "g1", nil, 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListApplications err = %v", err)
	}
	if err := repo.ApproveApplication(ctx, "u1", "a1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ApproveApplication err = %v", err)
	}
	if err := repo.RejectApplication(ctx, "a1", nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("RejectApplication err = %v", err)
	}
	if err := repo.UpdatePreferences(ctx, "u1", UpdateGroupPreferencesRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("UpdatePreferences err = %v", err)
	}
	if err := repo.LeaveGroup(ctx, "u1", "g1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("LeaveGroup err = %v", err)
	}
	if _, _, err := repo.ListMyApplications(ctx, "u1", nil, 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListMyApplications err = %v", err)
	}
	if err := repo.CancelApplication(ctx, "a1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("CancelApplication err = %v", err)
	}
	if _, _, _, err := repo.GetInvite(ctx, "i1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetInvite err = %v", err)
	}
	if _, _, err := repo.ListMyInvites(ctx, "u1", nil, 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListMyInvites err = %v", err)
	}
	if err := repo.AcceptInvite(ctx, "u1", "i1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("AcceptInvite err = %v", err)
	}
	if err := repo.RejectInvite(ctx, "i1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("RejectInvite err = %v", err)
	}
	if _, err := repo.InviteMember(ctx, "u1", InviteMemberRequest{GroupID: "g1"}, nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("InviteMember err = %v", err)
	}
	if _, _, err := repo.ListSentInvites(ctx, "g1", nil, 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListSentInvites err = %v", err)
	}
	if err := repo.RevokeInvite(ctx, "g1", "i1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("RevokeInvite err = %v", err)
	}
	if err := repo.SetMemberRole(ctx, "u1", SetMemberRoleRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("SetMemberRole err = %v", err)
	}
	if err := repo.MuteMember(ctx, "u1", MuteMemberRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MuteMember err = %v", err)
	}
	if err := repo.UnmuteMember(ctx, "u1", UnmuteMemberRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("UnmuteMember err = %v", err)
	}
	if err := repo.BanMember(ctx, "u1", BanMemberRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("BanMember err = %v", err)
	}
	if err := repo.UnbanMember(ctx, "u1", UnbanMemberRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("UnbanMember err = %v", err)
	}
	if _, _, err := repo.ListBannedMembers(ctx, "g1", 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListBannedMembers err = %v", err)
	}
	if err := repo.RemoveMember(ctx, "u1", RemoveMemberRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("RemoveMember err = %v", err)
	}
	if err := repo.TransferOwnership(ctx, "u1", TransferOwnershipRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("TransferOwnership err = %v", err)
	}
	if _, err := repo.ListGroupsByUser(ctx, "u1", 10, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("ListGroupsByUser err = %v", err)
	}
	if _, err := repo.CountCommonGroups(ctx, "u1", "u2"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("CountCommonGroups err = %v", err)
	}
	if _, err := repo.CountOwnedGroups(ctx, "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("CountOwnedGroups err = %v", err)
	}
	if err := repo.PurgeUserData(ctx, "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("PurgeUserData err = %v", err)
	}
	_ = now
}

// TestCovGroupPageCursorHelpers 覆盖通用分页游标编解码的非法输入分支。
func TestCovGroupPageCursorHelpers(t *testing.T) {
	if token := encodeGroupPageCursor(time.Now().UTC(), "id_1"); token == "" {
		t.Fatal("expected non-empty token")
	}
	if at, id, err := decodeGroupPageCursor(""); err != nil || !at.IsZero() || id != "" {
		t.Fatalf("empty cursor = %v, %q, %v", at, id, err)
	}
	if _, _, err := decodeGroupPageCursor("bogus"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bogus cursor err = %v", err)
	}
	missingIDToken, err := groupPageCursorCodec.Encode(groupPageCursor{At: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatalf("encode missing id: %v", err)
	}
	if _, _, err := decodeGroupPageCursor(missingIDToken); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("missing id err = %v", err)
	}
	badTimeToken, err := groupPageCursorCodec.Encode(groupPageCursor{At: "not-a-time", ID: "id_1"})
	if err != nil {
		t.Fatalf("encode bad time: %v", err)
	}
	if _, _, err := decodeGroupPageCursor(badTimeToken); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bad time err = %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	at, id, err := decodeGroupPageCursor(encodeGroupPageCursor(now, "id_1"))
	if err != nil || id != "id_1" || !at.Equal(now) {
		t.Fatalf("roundtrip = %v, %q, %v", at, id, err)
	}
}

// TestCovPostgresSearchWithViewerMembership 覆盖 Search 的批量成员状态回填
// （loadGroupMemberships 主路径：批量查询 + scan + 成员角色归一）。
func TestCovPostgresSearchWithViewerMembership(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		owner := testdb.InsertUser(t, txFactory, "covg2_search_owner", "Owner")
		viewer := testdb.InsertUser(t, txFactory, "covg2_search_viewer", "Viewer")
		group := covCreateTestGroup(t, ctx, repo, owner, "Cov Search Visible", nil)
		if _, err := repo.JoinGroup(ctx, viewer, group.Group.GroupID, nil); err != nil {
			t.Fatalf("join viewer: %v", err)
		}
		viewerCtx := platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: viewer})
		results, err := repo.Search(viewerCtx, SearchGroupsRequest{
			Keyword:        "Cov Search Visible",
			ReferencePoint: GeoPoint{Latitude: 35.1, Longitude: 112.2},
			Pagination:     &PaginationParams{Limit: 100},
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		found := false
		for _, item := range results.Items {
			if item.Group.GroupID == group.Group.GroupID {
				found = true
				if item.ViewerState == nil || item.ViewerState.Role != GroupRoleMember {
					t.Fatalf("viewer state = %#v", item.ViewerState)
				}
			}
		}
		if !found {
			t.Fatalf("expected group %s in search results", group.Group.GroupID)
		}
	})
}
