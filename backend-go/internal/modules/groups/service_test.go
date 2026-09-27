package groups

import (
	"context"
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/httpx"
	"strings"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/realtime"
)

// stubGroupsRepo implements Repository with only the fields needed for each test.
// Unused methods return zero values.
type stubGroupsRepo struct {
	isBanned        map[string]bool
	isMemberResult  bool
	adminMembers    []AdminGroupMemberItem
	adminMembersCur string
	adminMembersErr error
	getResult       GroupItem
	getFound        bool
	getErr          error
	createResult    GroupItem
	createErr       error
	searchResult    PaginatedGroups
	searchErr       error
	updateErr       error
	dismissErr      error
	ownedCount      int

	members    []GroupMemberItem
	membersCur string
	membersErr error

	searchMemResult []GroupMemberItem
	searchMemCur    string
	searchMemErr    error

	myGroups    []GroupItem
	myGroupsCur string
	myGroupsErr error

	joinResult EnterGroupResult
	joinErr    error

	announceItems []GroupAnnouncementItem
	announceErr   error
	createAnnItem GroupAnnouncementItem
	createAnnErr  error
	updateAnnErr  error
	deleteAnnErr  error

	apps       []GroupApplicationItem
	appsCur    string
	appsErr    error
	appsStatus *ApplicationStatus
	approveErr error
	rejectErr  error

	settingsErr error
	leaveErr    error

	myApps       []GroupApplicationItem
	myAppsCur    string
	myAppsErr    error
	myAppsStatus *ApplicationStatus

	cancelErr error

	myInvites       []GroupInviteItem
	myInvitesStatus *ApplicationStatus
	getInvite       func(inviteID string) (item GroupInviteItem, invitee string, found bool) // GetInvite 直查槽位（nil 回退扫 myInvites）
	getInviteErr    error
	myInvitesCur    string
	myInvitesErr    error

	acceptErr    error
	rejectInvErr error

	inviteResult Invitation
	inviteErr    error

	sentInvites       []GroupSentInviteItem
	sentInvitesStatus *ApplicationStatus
	sentInvitesCur    string
	sentInvitesErr    error

	revokeInvErr error

	setRoleErr error
	muteErr    error
	unmuteErr  error
	banErr     error
	unbanErr   error

	bannedMem    []AdminBannedMemberItem
	bannedMemCur string
	bannedMemErr error

	removeErr   error
	transferErr error
}

func (r *stubGroupsRepo) Create(_ context.Context, _ string, _ CreateGroupRequest) (GroupItem, error) {
	return r.createResult, r.createErr
}
func (r *stubGroupsRepo) Get(_ context.Context, _ string) (GroupItem, bool, error) {
	return r.getResult, r.getFound, r.getErr
}
func (r *stubGroupsRepo) Search(_ context.Context, _ SearchGroupsRequest) (PaginatedGroups, error) {
	return r.searchResult, r.searchErr
}
func (r *stubGroupsRepo) Update(_ context.Context, _ string, _ UpdateGroupRequest) error {
	return r.updateErr
}
func (r *stubGroupsRepo) Dismiss(_ context.Context, _ string) error { return r.dismissErr }
func (r *stubGroupsRepo) ListMembers(_ context.Context, _ string, _ *GroupRole, _ int, _ string) ([]GroupMemberItem, string, error) {
	return r.members, r.membersCur, r.membersErr
}
func (r *stubGroupsRepo) IsMember(_ context.Context, _ string, userID string) (bool, error) {
	if r.isMemberResult {
		return true, nil
	}
	for _, m := range r.members {
		if m.User.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}
func (r *stubGroupsRepo) AreMembers(_ context.Context, groupIDs []string, userID string) (map[string]bool, error) {
	result := make(map[string]bool, len(groupIDs))
	for _, groupID := range groupIDs {
		if r.isMemberResult {
			result[groupID] = true
			continue
		}
		for _, m := range r.members {
			if m.User.UserID == userID {
				result[groupID] = true
				break
			}
		}
	}
	return result, nil
}
func (r *stubGroupsRepo) GetApplication(_ context.Context, applicationID string) (GroupApplicationItem, bool, error) {
	// Read from the `apps` field that existing tests populate.
	for _, a := range r.apps {
		if a.ApplicationID == applicationID {
			return a, true, nil
		}
	}
	return GroupApplicationItem{}, false, nil
}
func (r *stubGroupsRepo) SearchMembers(_ context.Context, _ string, _ string, _ *GroupRole, _ int, _ string) ([]GroupMemberItem, string, error) {
	return r.searchMemResult, r.searchMemCur, r.searchMemErr
}
func (r *stubGroupsRepo) ListMyGroups(_ context.Context, _ string, _ int, _ string) ([]GroupItem, string, error) {
	return r.myGroups, r.myGroupsCur, r.myGroupsErr
}
func (r *stubGroupsRepo) JoinGroup(_ context.Context, _ string, _ string, _ *string, _ *string) (EnterGroupResult, error) {
	return r.joinResult, r.joinErr
}
func (r *stubGroupsRepo) CreateAnnouncement(_ context.Context, _ string, _ CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	return r.createAnnItem, r.createAnnErr
}
func (r *stubGroupsRepo) ListAnnouncements(_ context.Context, _ string) ([]GroupAnnouncementItem, error) {
	return r.announceItems, r.announceErr
}
func (r *stubGroupsRepo) UpdateAnnouncement(_ context.Context, _ string, _ UpdateGroupAnnouncementRequest) error {
	return r.updateAnnErr
}
func (r *stubGroupsRepo) DeleteAnnouncement(_ context.Context, _ string, _ string) error {
	return r.deleteAnnErr
}
func (r *stubGroupsRepo) ListApplications(_ context.Context, _ string, status *ApplicationStatus, _ int, _ string) ([]GroupApplicationItem, string, error) {
	r.appsStatus = status
	return r.apps, r.appsCur, r.appsErr
}
func (r *stubGroupsRepo) ApproveApplication(_ context.Context, _ string, _ string) error {
	return r.approveErr
}
func (r *stubGroupsRepo) RejectApplication(_ context.Context, _ string, _ *string) error {
	return r.rejectErr
}
func (r *stubGroupsRepo) UpdatePreferences(_ context.Context, _ string, _ UpdateGroupPreferencesRequest) error {
	return r.settingsErr
}
func (r *stubGroupsRepo) LeaveGroup(_ context.Context, _ string, _ string) error { return r.leaveErr }
func (r *stubGroupsRepo) ListMyApplications(_ context.Context, _ string, status *ApplicationStatus, _ int, _ string) ([]GroupApplicationItem, string, error) {
	r.myAppsStatus = status
	return r.myApps, r.myAppsCur, r.myAppsErr
}
func (r *stubGroupsRepo) CancelApplication(_ context.Context, _ string) error { return r.cancelErr }

// GetInvite 桩：getInvite 槽位非空时按其返回（直查路径），否则回退扫
// myInvites 列表（模拟旧行为的测试兼容）。
func (r *stubGroupsRepo) GetInvite(_ context.Context, inviteID string) (GroupInviteItem, string, bool, error) {
	if r.getInvite != nil {
		item, invitee, found := r.getInvite(inviteID)
		return item, invitee, found, r.getInviteErr
	}
	for _, item := range r.myInvites {
		if item.InviteID == inviteID {
			// invitee 返回空：myInvites 桩数据不携带被邀请人，视为「属于
			// 查询者」（service 侧空 invitee 跳过归属校验）。
			return item, "", true, nil
		}
	}
	return GroupInviteItem{}, "", false, r.myInvitesErr
}

func (r *stubGroupsRepo) ListMyInvites(_ context.Context, _ string, status *ApplicationStatus, _ int, _ string) ([]GroupInviteItem, string, error) {
	r.myInvitesStatus = status
	return r.myInvites, r.myInvitesCur, r.myInvitesErr
}
func (r *stubGroupsRepo) AcceptInvite(_ context.Context, _ string, _ string) error {
	return r.acceptErr
}

func (r *stubGroupsRepo) IsBanned(_ context.Context, _ string, userID string) (bool, error) {
	if r.isBanned == nil {
		return false, nil
	}
	return r.isBanned[userID], nil
}

func (r *stubGroupsRepo) ListAdminMembers(_ context.Context, _ string, _ *GroupRole, _ int, _ string) ([]AdminGroupMemberItem, string, error) {
	return r.adminMembers, r.adminMembersCur, r.adminMembersErr
}
func (r *stubGroupsRepo) RejectInvite(_ context.Context, _ string) error { return r.rejectInvErr }
func (r *stubGroupsRepo) InviteMember(_ context.Context, _ string, _ InviteMemberRequest, _ *time.Time) (Invitation, error) {
	return r.inviteResult, r.inviteErr
}
func (r *stubGroupsRepo) ListSentInvites(_ context.Context, _ string, status *ApplicationStatus, _ int, _ string) ([]GroupSentInviteItem, string, error) {
	r.sentInvitesStatus = status
	return r.sentInvites, r.sentInvitesCur, r.sentInvitesErr
}
func (r *stubGroupsRepo) RevokeInvite(_ context.Context, _ string, _ string) error {
	return r.revokeInvErr
}
func (r *stubGroupsRepo) SetMemberRole(_ context.Context, _ string, _ SetMemberRoleRequest) error {
	return r.setRoleErr
}
func (r *stubGroupsRepo) MuteMember(_ context.Context, _ string, _ MuteMemberRequest) error {
	return r.muteErr
}
func (r *stubGroupsRepo) UnmuteMember(_ context.Context, _ string, _ UnmuteMemberRequest) error {
	return r.unmuteErr
}
func (r *stubGroupsRepo) BanMember(_ context.Context, _ string, _ BanMemberRequest) error {
	return r.banErr
}
func (r *stubGroupsRepo) UnbanMember(_ context.Context, _ string, _ UnbanMemberRequest) error {
	return r.unbanErr
}
func (r *stubGroupsRepo) ListBannedMembers(_ context.Context, _ string, _ int, _ string) ([]AdminBannedMemberItem, string, error) {
	return r.bannedMem, r.bannedMemCur, r.bannedMemErr
}
func (r *stubGroupsRepo) RemoveMember(_ context.Context, _ string, _ RemoveMemberRequest) error {
	return r.removeErr
}
func (r *stubGroupsRepo) TransferOwnership(_ context.Context, _ string, _ TransferOwnershipRequest) error {
	return r.transferErr
}

func grpCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func ownerGroup() GroupItem {
	return GroupItem{
		Group:       GroupInfo{GroupID: "g1", Name: "test"},
		ViewerState: &GroupViewerState{Role: GroupRoleOwner},
	}
}

func adminGroup() GroupItem {
	return GroupItem{
		Group:       GroupInfo{GroupID: "g1", Name: "test"},
		ViewerState: &GroupViewerState{Role: GroupRoleAdmin},
	}
}

func memberGroup() GroupItem {
	return GroupItem{
		Group:       GroupInfo{GroupID: "g1", Name: "test"},
		ViewerState: &GroupViewerState{Role: GroupRoleMember},
	}
}

// --- CRUD tests ---

func TestServiceCreateDelegatesToRepo(t *testing.T) {
	expected := GroupItem{Group: GroupInfo{GroupID: "g1"}}
	repo := &stubGroupsRepo{createResult: expected}
	svc := NewServiceWithRepo(repo)
	result, err := svc.Create(grpCtx("u1"), CreateGroupRequest{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Group.GroupID != "g1" {
		t.Fatalf("expected g1, got %s", result.Group.GroupID)
	}
}

func TestServiceCreatePropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{createErr: errors.New("boom")})
	_, err := svc.Create(grpCtx("u1"), CreateGroupRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceOpenTheDoorReturnsItem(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	item, ok, err := svc.OpenTheDoor(context.Background(), "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok || item.Group.GroupID != "g1" {
		t.Fatalf("expected found g1, got ok=%v", ok)
	}
}

func TestServiceOpenTheDoorNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	_, ok, err := svc.OpenTheDoor(context.Background(), "missing")
	if err != nil || ok {
		t.Fatalf("expected not found, got ok=%v err=%v", ok, err)
	}
}

func TestServiceOpenTheDoorFillsDistance(t *testing.T) {
	// RelativeLocation 契约（spec api_spec_types.md）：有参考点时响应 location
	// 必须携带 distance/bearing_degrees；详情此前漏算恒为 null。
	item := ownerGroup()
	item.Location = &RelativeLocation{
		Point:        GeoPoint{Latitude: geo.DefaultFallbackLatitude + 0.01, Longitude: geo.DefaultFallbackLongitude},
		LocationName: "somewhere",
	}
	svc := NewServiceWithRepo(&stubGroupsRepo{getResult: item, getFound: true})
	got, ok, err := svc.OpenTheDoor(context.Background(), "g1")
	if err != nil || !ok {
		t.Fatalf("unexpected: ok=%v err=%v", ok, err)
	}
	if got.Location == nil || got.Location.Distance == nil || *got.Location.Distance <= 0 {
		t.Fatalf("expected positive distance, got %+v", got.Location)
	}
	if got.Location.BearingDegrees == nil {
		t.Fatal("expected bearing_degrees to be set")
	}
}

func TestServiceUpdateDeniesMember(t *testing.T) {
	// 群管理守卫子树语义（spec api_spec_authz.md）：成员无管理权限 → 10200 + not_group_admin。
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	var apiErr *httpx.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected 10200 not_group_admin, got %v", err)
	}
}

func TestServiceUpdateAllowsAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: adminGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("expected admin update to succeed, got %v", err)
	}
}

func TestServiceUpdateRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	_, err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceUpdateHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDismissRequiresOwner(t *testing.T) {
	repo := &stubGroupsRepo{getResult: adminGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Dismiss(grpCtx("u1"), DismissGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceDismissRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	err := svc.Dismiss(grpCtx("u1"), DismissGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceDismissHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Dismiss(grpCtx("u1"), DismissGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Member management ---

func TestServiceListMembersRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	_, _, err := svc.ListMembers(context.Background(), GroupMembersRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceListMembersNormalizesLimit(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, isMemberResult: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}}
	svc := NewServiceWithRepo(repo)
	items, _, err := svc.ListMembers(grpCtx("u1"), GroupMembersRequest{GroupID: "g1", Pagination: &PaginationParams{Limit: 150}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

// --- Join/Leave ---

func TestServiceJoinGroupRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	_, err := svc.JoinGroup(grpCtx("u1"), EnterGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceJoinGroupAlreadyJoined(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	result, err := svc.JoinGroup(grpCtx("u1"), EnterGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != EnterGroupStatusJoined {
		t.Fatalf("expected joined status, got %s", result.Status)
	}
}

func TestServiceJoinGroupHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}}, getFound: true, joinResult: EnterGroupResult{Status: EnterGroupStatusJoined}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.JoinGroup(grpCtx("u1"), EnterGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != EnterGroupStatusJoined {
		t.Fatalf("expected joined, got %s", result.Status)
	}
}

func TestServiceLeaveGroupRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	err := svc.LeaveGroup(grpCtx("u1"), LeaveGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceLeaveGroupOwnerCannotLeave(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.LeaveGroup(grpCtx("u1"), LeaveGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceLeaveGroupNotMember(t *testing.T) {
	repo := &stubGroupsRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}}, getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.LeaveGroup(grpCtx("u1"), LeaveGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected not member, got %v", err)
	}
}

func TestServiceLeaveGroupHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.LeaveGroup(grpCtx("u1"), LeaveGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Role management ---

func TestServiceSetMemberRoleRequiresOwner(t *testing.T) {
	repo := &stubGroupsRepo{getResult: adminGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberRole(grpCtx("u1"), SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleAdmin})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceSetMemberRoleRejectsInvalidRole(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberRole(grpCtx("u1"), SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleOwner})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted for owner role, got %v", err)
	}
}

// --- Mute/Unmute ---

func TestServiceMuteMemberRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.MuteMember(grpCtx("u1"), MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 10})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceMuteMemberRejectsInvalidDuration(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.MuteMember(grpCtx("u1"), MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 0})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted for 0 duration, got %v", err)
	}
	err = svc.MuteMember(grpCtx("u1"), MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: MaxMuteDurationMinutes + 1})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted for too long duration, got %v", err)
	}
}

// --- Ban/Unban ---

func TestServiceBanMemberCannotBanSelf(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberBan(grpCtx("u1"), SetMemberBanRequest{GroupID: "g1", UserID: "u1", Active: true})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceBanMemberRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberBan(grpCtx("u1"), SetMemberBanRequest{GroupID: "g1", UserID: "u2", Active: true})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

// --- Transfer ownership ---

func TestServiceTransferOwnershipRequiresOwner(t *testing.T) {
	repo := &stubGroupsRepo{getResult: adminGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.TransferOwnership(grpCtx("u1"), TransferOwnershipRequest{GroupID: "g1", NewOwnerID: "u2"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceTransferOwnershipCannotTransferToSelf(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.TransferOwnership(grpCtx("u1"), TransferOwnershipRequest{GroupID: "g1", NewOwnerID: "u1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceTransferOwnershipTargetMustBeMember(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u3"}}}}
	svc := NewServiceWithRepo(repo)
	err := svc.TransferOwnership(grpCtx("u1"), TransferOwnershipRequest{GroupID: "g1", NewOwnerID: "u2"})
	if !errors.Is(err, ErrTargetNotMember) {
		t.Fatalf("expected target not member, got %v", err)
	}
}

func TestServiceTransferOwnershipHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}}
	svc := NewServiceWithRepo(repo)
	err := svc.TransferOwnership(grpCtx("u1"), TransferOwnershipRequest{GroupID: "g1", NewOwnerID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Announcements ---

func TestServiceCreateAnnouncementRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, err := svc.CreateAnnouncement(grpCtx("u1"), CreateGroupAnnouncementRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceCreateAnnouncementHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, createAnnItem: GroupAnnouncementItem{AnnouncementID: "ann1"}}
	svc := NewServiceWithRepo(repo)
	item, err := svc.CreateAnnouncement(grpCtx("u1"), CreateGroupAnnouncementRequest{GroupID: "g1", Title: "hi", Content: "body"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.AnnouncementID != "ann1" {
		t.Fatalf("expected ann1, got %s", item.AnnouncementID)
	}
}

func TestServiceListAnnouncementsDelegates(t *testing.T) {
	repo := &stubGroupsRepo{announceItems: []GroupAnnouncementItem{{AnnouncementID: "ann1"}}}
	svc := NewServiceWithRepo(repo)
	items, err := svc.ListAnnouncements(context.Background(), GroupAnnouncementsRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

// --- Invite ---

func TestServiceInviteMemberRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, err := svc.InviteMember(grpCtx("u1"), InviteMemberRequest{GroupID: "g1", UserID: "u2"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceInviteMemberRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	_, err := svc.InviteMember(grpCtx("u1"), InviteMemberRequest{GroupID: "g1", UserID: "u2"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceInviteMemberHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, inviteResult: Invitation{InviteID: strPtr("inv1")}}
	svc := NewServiceWithRepo(repo)
	inv, err := svc.InviteMember(grpCtx("u1"), InviteMemberRequest{GroupID: "g1", UserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *inv.InviteID != "inv1" {
		t.Fatalf("expected inv1, got %s", *inv.InviteID)
	}
}

// --- Remove member ---

func TestServiceRemoveMemberCannotRemoveSelf(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.RemoveMember(grpCtx("u1"), RemoveMemberRequest{GroupID: "g1", UserID: "u1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceRemoveMemberRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.RemoveMember(grpCtx("u1"), RemoveMemberRequest{GroupID: "g1", UserID: "u2"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

// --- List my groups ---

func TestServiceListMyGroupsNormalizesLimit(t *testing.T) {
	repo := &stubGroupsRepo{myGroups: []GroupItem{ownerGroup()}}
	svc := NewServiceWithRepo(repo)
	items, _, err := svc.ListMyGroups(grpCtx("u1"), MyGroupsRequest{Pagination: &PaginationParams{Limit: 150}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

// --- Applications ---

func TestServiceListApplicationsRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, _, err := svc.ListApplications(grpCtx("u1"), GroupApplicationsRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

// --- Accept/Reject invite ---

func TestServiceAcceptInviteRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	err := svc.AcceptInvite(grpCtx("u1"), AcceptInviteRequest{InviteID: "inv1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceRejectInviteRejectsExpired(t *testing.T) {
	rejected := ApplicationStatusRejected
	repo := &stubGroupsRepo{myInvites: []GroupInviteItem{{InviteID: "inv1", Status: rejected}}}
	svc := NewServiceWithRepo(repo)
	err := svc.RejectInvite(grpCtx("u1"), RejectInviteRequest{InviteID: "inv1"})
	if !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("expected invite expired, got %v", err)
	}
}

// --- Update my settings ---

func TestServiceUpdatePreferencesRejectsNotMember(t *testing.T) {
	repo := &stubGroupsRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}}, getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.UpdatePreferences(grpCtx("u1"), UpdateGroupPreferencesRequest{GroupID: "g1"})
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected not member, got %v", err)
	}
}

func TestServiceUpdatePreferencesHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.UpdatePreferences(grpCtx("u1"), UpdateGroupPreferencesRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =========================================================================
// A) service_invite 补充测试
// =========================================================================

// 直查归属校验：invitee 不符的邀请（他人的邀请）按不存在处理，不泄漏存在性。
func TestServiceAcceptInviteRejectsForeignInvite(t *testing.T) {
	repo := &stubGroupsRepo{
		getInvite: func(inviteID string) (GroupInviteItem, string, bool) {
			return GroupInviteItem{InviteID: "inv1", Group: GroupInfo{GroupID: "g1"}, Status: ApplicationStatusPending}, "u_other", true
		},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.AcceptInvite(grpCtx("u1"), AcceptInviteRequest{InviteID: "inv1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound for foreign invite, got %v", err)
	}
}

func TestServiceAcceptInviteHappyPath(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		myInvites: []GroupInviteItem{{
			InviteID: "inv1",
			Status:   pending,
			Group:    GroupInfo{GroupID: "g1", Name: "test", MembersCount: 5},
			Inviter:  UserInfo{UserID: "u2"},
		}},
		acceptErr: nil,
	}
	svc := NewServiceWithRepo(repo)
	err := svc.AcceptInvite(grpCtx("u1"), AcceptInviteRequest{InviteID: "inv1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceAcceptInvitePropagatesRepoError(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		myInvites: []GroupInviteItem{{InviteID: "inv1", Status: pending, Group: GroupInfo{GroupID: "g1"}, Inviter: UserInfo{UserID: "u2"}}},
		acceptErr: errors.New("db error"),
	}
	svc := NewServiceWithRepo(repo)
	err := svc.AcceptInvite(grpCtx("u1"), AcceptInviteRequest{InviteID: "inv1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceAcceptInviteRejectsExpired(t *testing.T) {
	rejected := ApplicationStatusRejected
	repo := &stubGroupsRepo{
		myInvites: []GroupInviteItem{{InviteID: "inv1", Status: rejected, Group: GroupInfo{GroupID: "g1"}, Inviter: UserInfo{UserID: "u2"}}},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.AcceptInvite(grpCtx("u1"), AcceptInviteRequest{InviteID: "inv1"})
	// 非 pending 邀请（已拒绝/撤销/过期）不得被接受——此前缺失该检查，
	// 被撤销的邀请仍可凭旧 invite_id 入群。
	if err != ErrInviteExpired {
		t.Fatalf("expected ErrInviteExpired, got %v", err)
	}
}

func TestServiceRejectInviteHappyPath(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		myInvites: []GroupInviteItem{{InviteID: "inv1", Status: pending, Group: GroupInfo{GroupID: "g1"}, Inviter: UserInfo{UserID: "u2"}}},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.RejectInvite(grpCtx("u1"), RejectInviteRequest{InviteID: "inv1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceRejectInvitePropagatesRepoError(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		myInvites:    []GroupInviteItem{{InviteID: "inv1", Status: pending, Group: GroupInfo{GroupID: "g1"}, Inviter: UserInfo{UserID: "u2"}}},
		rejectInvErr: errors.New("db error"),
	}
	svc := NewServiceWithRepo(repo)
	err := svc.RejectInvite(grpCtx("u1"), RejectInviteRequest{InviteID: "inv1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceListMyInvitesDelegatesToRepo(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		myInvites:    []GroupInviteItem{{InviteID: "inv1", Status: pending}},
		myInvitesCur: "cursor1",
	}
	svc := NewServiceWithRepo(repo)
	items, cursor, err := svc.ListMyInvites(grpCtx("u1"), MyInvitesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if cursor != "cursor1" {
		t.Fatalf("expected cursor1, got %s", cursor)
	}
}

// spec group.md → fetch_received：status 缺省 pending；查历史显式传 all。
func TestServiceListMyInvitesDefaultsToPending(t *testing.T) {
	repo := &stubGroupsRepo{}
	svc := NewServiceWithRepo(repo)
	if _, _, err := svc.ListMyInvites(grpCtx("u1"), MyInvitesRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.myInvitesStatus == nil || *repo.myInvitesStatus != ApplicationStatusPending {
		t.Fatalf("expected pending default, got %#v", repo.myInvitesStatus)
	}
}

func TestServiceListMyInvitesAllPassesThrough(t *testing.T) {
	repo := &stubGroupsRepo{}
	svc := NewServiceWithRepo(repo)
	all := ApplicationStatus("all")
	if _, _, err := svc.ListMyInvites(grpCtx("u1"), MyInvitesRequest{Status: &all}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.myInvitesStatus != nil {
		t.Fatalf("expected nil (no filter) for all, got %#v", repo.myInvitesStatus)
	}
}

// spec group.md → admin/applications/fetch：status 缺省 pending；查历史显式传 all。
func TestServiceListApplicationsDefaultsToPending(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	if _, _, err := svc.ListApplications(grpCtx("u1"), GroupApplicationsRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.appsStatus == nil || *repo.appsStatus != ApplicationStatusPending {
		t.Fatalf("expected pending default, got %#v", repo.appsStatus)
	}
}

func TestServiceListApplicationsAllPassesThrough(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	all := ApplicationStatus("all")
	if _, _, err := svc.ListApplications(grpCtx("u1"), GroupApplicationsRequest{GroupID: "g1", Status: &all}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.appsStatus != nil {
		t.Fatalf("expected nil (no filter) for all, got %#v", repo.appsStatus)
	}
}

func TestServiceListMyInvitesNormalizesLimit(t *testing.T) {
	repo := &stubGroupsRepo{
		myInvites: []GroupInviteItem{{InviteID: "inv1"}},
	}
	svc := NewServiceWithRepo(repo)
	items, _, err := svc.ListMyInvites(grpCtx("u1"), MyInvitesRequest{Pagination: &PaginationParams{Limit: 150}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

// =========================================================================
// B) service_member 补充测试
// =========================================================================

// --- ApproveApplication ---

func TestServiceApproveApplicationHappyPath(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		apps:      []GroupApplicationItem{{ApplicationID: "app1", GroupID: "g1", Status: pending, Applicant: UserInfo{UserID: "u3"}}},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceApproveApplicationRejectsNotFound(t *testing.T) {
	repo := &stubGroupsRepo{
		myGroups: []GroupItem{ownerGroup()},
		apps:     []GroupApplicationItem{}, // 无匹配申请
	}
	svc := NewServiceWithRepo(repo)
	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{GroupID: "g1", ApplicationID: "missing"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestServiceApproveApplicationPropagatesRepoError(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		getResult:  ownerGroup(),
		getFound:   true,
		apps:       []GroupApplicationItem{{ApplicationID: "app1", GroupID: "g1", Status: pending, Applicant: UserInfo{UserID: "u3"}}},
		approveErr: errors.New("db error"),
	}
	svc := NewServiceWithRepo(repo)
	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- RejectApplication ---

func TestServiceRejectApplicationRejectsNotFound(t *testing.T) {
	repo := &stubGroupsRepo{
		myGroups: []GroupItem{ownerGroup()},
		apps:     []GroupApplicationItem{},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{GroupID: "g1", ApplicationID: "missing"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestServiceRejectApplicationHappyPath(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		apps:      []GroupApplicationItem{{ApplicationID: "app1", GroupID: "g1", Status: pending, Applicant: UserInfo{UserID: "u3"}}},
	}
	svc := NewServiceWithRepo(repo)
	reason := "not suitable"
	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1", Reason: &reason})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceRejectApplicationPropagatesRepoError(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		apps:      []GroupApplicationItem{{ApplicationID: "app1", GroupID: "g1", Status: pending, Applicant: UserInfo{UserID: "u3"}}},
		rejectErr: errors.New("db error"),
	}
	svc := NewServiceWithRepo(repo)
	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- SetMemberRole ---

func TestServiceSetMemberRoleHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberRole(grpCtx("u1"), SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleAdmin})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSetMemberRolePropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}, setRoleErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberRole(grpCtx("u1"), SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleAdmin})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- MuteMember ---

func TestServiceMuteMemberHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}}
	svc := NewServiceWithRepo(repo)
	err := svc.MuteMember(grpCtx("u1"), MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceMuteMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}, muteErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.MuteMember(grpCtx("u1"), MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 10})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- BanMember ---

func TestServiceBanMemberHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberBan(grpCtx("u1"), SetMemberBanRequest{GroupID: "g1", UserID: "u2", Active: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceBanMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, banErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberBan(grpCtx("u1"), SetMemberBanRequest{GroupID: "g1", UserID: "u2", Active: true})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- UnbanMember ---

func TestServiceUnbanMemberRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberBan(grpCtx("u1"), SetMemberBanRequest{GroupID: "g1", UserID: "u2", Active: false})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected ErrAccessRestricted, got %v", err)
	}
}

func TestServiceUnbanMemberHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberBan(grpCtx("u1"), SetMemberBanRequest{GroupID: "g1", UserID: "u2", Active: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUnbanMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, unbanErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberBan(grpCtx("u1"), SetMemberBanRequest{GroupID: "g1", UserID: "u2", Active: false})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- ListBannedMembers ---

func TestServiceListBannedMembersRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, _, err := svc.ListBannedMembers(grpCtx("u1"), BannedMembersRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected ErrAccessRestricted, got %v", err)
	}
}

func TestServiceListBannedMembersRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	_, _, err := svc.ListBannedMembers(grpCtx("u1"), BannedMembersRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

// --- RemoveMember ---

func TestServiceRemoveMemberHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}}
	svc := NewServiceWithRepo(repo)
	err := svc.RemoveMember(grpCtx("u1"), RemoveMemberRequest{GroupID: "g1", UserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceRemoveMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}, removeErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.RemoveMember(grpCtx("u1"), RemoveMemberRequest{GroupID: "g1", UserID: "u2"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- TransferOwnership ---

func TestServiceTransferOwnershipPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{
		getResult:   ownerGroup(),
		getFound:    true,
		members:     []GroupMemberItem{{User: UserInfo{UserID: "u2"}}},
		transferErr: errors.New("db error"),
	}
	svc := NewServiceWithRepo(repo)
	err := svc.TransferOwnership(grpCtx("u1"), TransferOwnershipRequest{GroupID: "g1", NewOwnerID: "u2"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// =========================================================================
// C) service.go 额外错误路径
// =========================================================================

func TestServiceUpdatePropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, updateErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	_, err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "db error") {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceDismissPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, dismissErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.Dismiss(grpCtx("u1"), DismissGroupRequest{GroupID: "g1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "db error") {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceListMyApplicationsDelegatesToRepo(t *testing.T) {
	repo := &stubGroupsRepo{
		myApps:    []GroupApplicationItem{{ApplicationID: "app1"}},
		myAppsCur: "cursor1",
	}
	svc := NewServiceWithRepo(repo)
	items, cursor, err := svc.ListMyApplications(grpCtx("u1"), MyApplicationsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if cursor != "cursor1" {
		t.Fatalf("expected cursor1, got %s", cursor)
	}
}

func TestServiceListMyApplicationsNormalizesLimit(t *testing.T) {
	repo := &stubGroupsRepo{
		myApps: []GroupApplicationItem{{ApplicationID: "app1"}},
	}
	svc := NewServiceWithRepo(repo)
	items, _, err := svc.ListMyApplications(grpCtx("u1"), MyApplicationsRequest{Pagination: &PaginationParams{Limit: 150}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

// spec group.md → fetch_submitted：status 缺省 pending；查历史显式传 all。
func TestServiceListMyApplicationsDefaultsToPending(t *testing.T) {
	repo := &stubGroupsRepo{}
	svc := NewServiceWithRepo(repo)
	if _, _, err := svc.ListMyApplications(grpCtx("u1"), MyApplicationsRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.myAppsStatus == nil || *repo.myAppsStatus != ApplicationStatusPending {
		t.Fatalf("expected pending default, got %#v", repo.myAppsStatus)
	}
}

func TestServiceListMyApplicationsAllPassesThrough(t *testing.T) {
	repo := &stubGroupsRepo{}
	svc := NewServiceWithRepo(repo)
	all := ApplicationStatus("all")
	if _, _, err := svc.ListMyApplications(grpCtx("u1"), MyApplicationsRequest{Status: &all}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.myAppsStatus != nil {
		t.Fatalf("expected nil (no filter) for all, got %#v", repo.myAppsStatus)
	}
}

func TestServiceListMyApplicationsPropagatesError(t *testing.T) {
	repo := &stubGroupsRepo{myAppsErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	_, _, err := svc.ListMyApplications(grpCtx("u1"), MyApplicationsRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceCancelApplicationDelegatesToRepo(t *testing.T) {
	repo := &stubGroupsRepo{
		apps: []GroupApplicationItem{
			{ApplicationID: "app1", GroupID: "g1", Applicant: UserInfo{UserID: "u1"}},
		},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.CancelApplication(grpCtx("u1"), CancelApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceCancelApplicationPropagatesError(t *testing.T) {
	repo := &stubGroupsRepo{
		cancelErr: errors.New("db error"),
		apps: []GroupApplicationItem{
			{ApplicationID: "app1", GroupID: "g1", Applicant: UserInfo{UserID: "u1"}},
		},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.CancelApplication(grpCtx("u1"), CancelApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceListMyGroupsPropagatesError(t *testing.T) {
	repo := &stubGroupsRepo{myGroupsErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	_, _, err := svc.ListMyGroups(grpCtx("u1"), MyGroupsRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceListApplicationsHappyPath(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		apps:      []GroupApplicationItem{{ApplicationID: "app1", Status: pending}},
		appsCur:   "cursor1",
	}
	svc := NewServiceWithRepo(repo)
	items, cursor, err := svc.ListApplications(grpCtx("u1"), GroupApplicationsRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if cursor != "cursor1" {
		t.Fatalf("expected cursor1, got %s", cursor)
	}
}

func TestServiceListApplicationsPropagatesError(t *testing.T) {
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		appsErr:   errors.New("db error"),
	}
	svc := NewServiceWithRepo(repo)
	_, _, err := svc.ListApplications(grpCtx("u1"), GroupApplicationsRequest{GroupID: "g1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceInviteMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		inviteErr: errors.New("db error"),
	}
	svc := NewServiceWithRepo(repo)
	_, err := svc.InviteMember(grpCtx("u1"), InviteMemberRequest{GroupID: "g1", UserID: "u2"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceListSentInvitesRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	_, _, err := svc.ListSentInvites(grpCtx("u1"), GroupSentInvitesRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected ErrAccessRestricted, got %v", err)
	}
}

func TestServiceListSentInvitesRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	_, _, err := svc.ListSentInvites(grpCtx("u1"), GroupSentInvitesRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestServiceListSentInvitesHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{
		getResult:      ownerGroup(),
		getFound:       true,
		sentInvites:    []GroupSentInviteItem{{InviteID: "inv1"}},
		sentInvitesCur: "cursor1",
	}
	svc := NewServiceWithRepo(repo)
	items, cursor, err := svc.ListSentInvites(grpCtx("u1"), GroupSentInvitesRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if cursor != "cursor1" {
		t.Fatalf("expected cursor1, got %s", cursor)
	}
}

// spec group.md → admin/invites/fetch_sent：status 缺省 pending；查历史显式传 all。
func TestServiceListSentInvitesDefaultsToPending(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	if _, _, err := svc.ListSentInvites(grpCtx("u1"), GroupSentInvitesRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.sentInvitesStatus == nil || *repo.sentInvitesStatus != ApplicationStatusPending {
		t.Fatalf("expected pending default, got %#v", repo.sentInvitesStatus)
	}
}

func TestServiceListSentInvitesAllPassesThrough(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	all := ApplicationStatus("all")
	if _, _, err := svc.ListSentInvites(grpCtx("u1"), GroupSentInvitesRequest{GroupID: "g1", Status: &all}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.sentInvitesStatus != nil {
		t.Fatalf("expected nil (no filter) for all, got %#v", repo.sentInvitesStatus)
	}
}

func TestServiceRevokeInviteRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.RevokeInvite(grpCtx("u1"), RevokeSentInviteRequest{GroupID: "g1", InviteID: "inv1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected ErrAccessRestricted, got %v", err)
	}
}

func TestServiceRevokeInviteRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	err := svc.RevokeInvite(grpCtx("u1"), RevokeSentInviteRequest{GroupID: "g1", InviteID: "inv1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestServiceRevokeInviteHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.RevokeInvite(grpCtx("u1"), RevokeSentInviteRequest{GroupID: "g1", InviteID: "inv1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdateBroadcastsGroupUpdatedWithChangedFields(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	created, err := service.Create(ctx, CreateGroupRequest{Name: "Old Name", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sink := &groupRealtimeSink{}
	hub.SetGroupSubscriptions([]string{created.Group.GroupID}, sink)

	if _, err := service.Update(ctx, UpdateGroupRequest{GroupID: created.Group.GroupID, Name: "New Name", Description: "new desc"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeGroupUpdated {
		t.Fatalf("unexpected realtime events: %#v", sink.events)
	}
	payload := decodePayload(t, sink.events[0].Payload)
	fields, ok := payload["updated_fields"].(map[string]any)
	if !ok {
		t.Fatalf("updated_fields missing from payload: %#v", payload)
	}
	if fields["name"] != "New Name" || fields["description"] != "new desc" {
		t.Fatalf("unexpected updated_fields: %#v", fields)
	}
	if _, leaked := fields["is_searchable"]; leaked {
		t.Fatalf("is_searchable must not be pushed (admin-only): %#v", fields)
	}
	if _, leaked := fields["max_members"]; leaked {
		t.Fatalf("max_members must not be pushed (admin-only): %#v", fields)
	}
	if fields["join_method"] != nil || fields["requires_approval"] != nil || fields["has_password"] != nil {
		t.Fatalf("unchanged fields must not be pushed: %#v", fields)
	}
}

func (r *stubGroupsRepo) ListGroupsByUser(context.Context, string, int, string) (PaginatedGroups, error) {
	return PaginatedGroups{}, nil
}

func (r *stubGroupsRepo) CountCommonGroups(context.Context, string, string) (int, error) {
	return 0, nil
}

func (r *stubGroupsRepo) CountOwnedGroups(context.Context, string) (int, error) {
	return r.ownedCount, nil
}

// --- 审计修复新增：封禁旁路 / owner 自降级 / 非成员操作 ---

func TestServiceSetMemberRoleRejectsOwnerDemotion(t *testing.T) {
	// owner 对本人自降级会制造无人可管理的「孤儿群」，必须拒绝（10400）。
	owner := "u1"
	item := ownerGroup()
	item.OwnerID = &owner
	repo := &stubGroupsRepo{getResult: item, getFound: true, isMemberResult: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberRole(grpCtx("u1"), SetMemberRoleRequest{GroupID: "g1", UserID: "u1", Role: GroupRoleMember})
	if !errors.Is(err, ErrBusinessConflict) {
		t.Fatalf("expected business conflict for owner self-demotion, got %v", err)
	}
}

func TestServiceSetMemberRoleRejectsNonMemberTarget(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberRole(grpCtx("u1"), SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleAdmin})
	if !errors.Is(err, ErrTargetNotMember) {
		t.Fatalf("expected target not member, got %v", err)
	}
}

// 解禁目标必须是群成员（对齐 MuteMember 的 IsMember 校验）。
func TestServiceUnmuteMemberRejectsNonMemberTarget(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.UnmuteMember(grpCtx("u1"), UnmuteMemberRequest{GroupID: "g1", UserID: "u2"})
	if !errors.Is(err, ErrTargetNotMember) {
		t.Fatalf("expected target not member, got %v", err)
	}
}

func TestServiceMuteMemberRejectsNonMemberTarget(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.MuteMember(grpCtx("u1"), MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 10})
	if !errors.Is(err, ErrTargetNotMember) {
		t.Fatalf("expected target not member, got %v", err)
	}
}

func TestServiceRemoveMemberRejectsNonMemberTarget(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.RemoveMember(grpCtx("u1"), RemoveMemberRequest{GroupID: "g1", UserID: "u2"})
	if !errors.Is(err, ErrTargetNotMember) {
		t.Fatalf("expected target not member, got %v", err)
	}
}

func TestServiceInviteMemberRejectsBannedTarget(t *testing.T) {
	// 契约：邀请已被封禁的用户返回 10400 语义（ErrBannedInGroup）。
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, isBanned: map[string]bool{"u2": true}}
	svc := NewServiceWithRepo(repo)
	_, err := svc.InviteMember(grpCtx("u1"), InviteMemberRequest{GroupID: "g1", UserID: "u2"})
	if !errors.Is(err, ErrBannedInGroup) {
		t.Fatalf("expected banned in group, got %v", err)
	}
}

func TestServiceAcceptInviteRejectsBannedUser(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		isBanned:  map[string]bool{"u2": true},
		myInvites: []GroupInviteItem{{InviteID: "inv1", Status: pending, Group: GroupInfo{GroupID: "g1"}, Inviter: UserInfo{UserID: "u3"}}},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.AcceptInvite(grpCtx("u2"), AcceptInviteRequest{InviteID: "inv1"})
	if !errors.Is(err, ErrBannedInGroup) {
		t.Fatalf("expected banned in group, got %v", err)
	}
}

func TestServiceApproveApplicationRejectsBannedApplicant(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, isBanned: map[string]bool{"u2": true}}
	repo.apps = []GroupApplicationItem{{
		ApplicationID: "app1",
		GroupID:       "g1",
		Applicant:     UserInfo{UserID: "u2"},
		Status:        ApplicationStatusPending,
	}}
	svc := NewServiceWithRepo(repo)
	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if !errors.Is(err, ErrBannedInGroup) {
		t.Fatalf("expected banned in group, got %v", err)
	}
}

// decodePayload 把 WS payload（现为类型化 struct）经 JSON round-trip 转回
// map，让既有键级断言继续验证「序列化后的形状」——那才是真正的线上契约。
// 注意数字经 JSON 解码为 float64。
func decodePayload(t *testing.T, p any) map[string]any {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("payload is not an object: %v (%#v)", err, p)
	}
	return m
}
