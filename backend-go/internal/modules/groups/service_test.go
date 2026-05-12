package groups

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// stubGroupsRepo implements Repository with only the fields needed for each test.
// Unused methods return zero values.
type stubGroupsRepo struct {
	getResult    GroupItem
	getFound     bool
	getErr       error
	createResult GroupItem
	createErr    error
	searchResult PaginatedGroups
	searchErr    error
	updateErr    error
	dismissErr   error

	members    []GroupMemberItem
	membersCur string
	membersErr error

	searchMemResult []GroupMemberItem
	searchMemCur    string
	searchMemErr    error

	myGroups    []GroupItem
	myGroupsCur string
	myGroupsErr error

	joinResult JoinGroupResult
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
	approveErr error
	rejectErr  error

	settingsErr error
	leaveErr    error

	myApps    []GroupApplicationItem
	myAppsCur string
	myAppsErr error

	cancelErr error

	myInvites    []GroupInviteItem
	myInvitesCur string
	myInvitesErr error

	acceptErr    error
	rejectInvErr error

	inviteResult Invitation
	inviteErr    error

	sentInvites    []GroupSentInviteItem
	sentInvitesCur string
	sentInvitesErr error

	revokeInvErr error

	setRoleErr error
	muteErr    error
	unmuteErr  error
	banErr     error
	unbanErr   error

	bannedMem    []BannedMemberItem
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
func (r *stubGroupsRepo) SearchMembers(_ context.Context, _ string, _ string, _ *GroupRole, _ int, _ string) ([]GroupMemberItem, string, error) {
	return r.searchMemResult, r.searchMemCur, r.searchMemErr
}
func (r *stubGroupsRepo) ListMyGroups(_ context.Context, _ string, _ int, _ string) ([]GroupItem, string, error) {
	return r.myGroups, r.myGroupsCur, r.myGroupsErr
}
func (r *stubGroupsRepo) JoinGroup(_ context.Context, _ string, _ string, _ *string, _ *string) (JoinGroupResult, error) {
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
func (r *stubGroupsRepo) ListApplications(_ context.Context, _ string, _ *ApplicationStatus, _ int, _ string) ([]GroupApplicationItem, string, error) {
	return r.apps, r.appsCur, r.appsErr
}
func (r *stubGroupsRepo) ApproveApplication(_ context.Context, _ string, _ string) error {
	return r.approveErr
}
func (r *stubGroupsRepo) RejectApplication(_ context.Context, _ string, _ *string) error {
	return r.rejectErr
}
func (r *stubGroupsRepo) UpdateMySettings(_ context.Context, _ string, _ UpdateGroupMySettingsRequest) error {
	return r.settingsErr
}
func (r *stubGroupsRepo) LeaveGroup(_ context.Context, _ string, _ string) error { return r.leaveErr }
func (r *stubGroupsRepo) ListMyApplications(_ context.Context, _ string, _ *ApplicationStatus, _ int, _ string) ([]GroupApplicationItem, string, error) {
	return r.myApps, r.myAppsCur, r.myAppsErr
}
func (r *stubGroupsRepo) CancelApplication(_ context.Context, _ string) error { return r.cancelErr }
func (r *stubGroupsRepo) ListMyInvites(_ context.Context, _ string, _ *ApplicationStatus, _ int, _ string) ([]GroupInviteItem, string, error) {
	return r.myInvites, r.myInvitesCur, r.myInvitesErr
}
func (r *stubGroupsRepo) AcceptInvite(_ context.Context, _ string, _ string) error {
	return r.acceptErr
}
func (r *stubGroupsRepo) RejectInvite(_ context.Context, _ string) error { return r.rejectInvErr }
func (r *stubGroupsRepo) InviteMember(_ context.Context, _ string, _ InviteMemberRequest) (Invitation, error) {
	return r.inviteResult, r.inviteErr
}
func (r *stubGroupsRepo) ListSentInvites(_ context.Context, _ string, _ *ApplicationStatus, _ int, _ string) ([]GroupSentInviteItem, string, error) {
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
func (r *stubGroupsRepo) ListBannedMembers(_ context.Context, _ string, _ int, _ string) ([]BannedMemberItem, string, error) {
	return r.bannedMem, r.bannedMemCur, r.bannedMemErr
}
func (r *stubGroupsRepo) RemoveMember(_ context.Context, _ string, _ RemoveMemberRequest) error {
	return r.removeErr
}
func (r *stubGroupsRepo) TransferOwnership(_ context.Context, _ string, _ TransferOwnershipRequest) error {
	return r.transferErr
}

func (r *stubGroupsRepo) IsMember(_ context.Context, _ string, _ string) (bool, error) {
	return false, nil
}

func grpCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func ownerGroup() GroupItem {
	return GroupItem{
		Group:   GroupInfo{GroupID: "g1", Name: "test"},
		MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true},
	}
}

func adminGroup() GroupItem {
	return GroupItem{
		Group:   GroupInfo{GroupID: "g1", Name: "test"},
		MyState: &MyGroupState{Role: GroupRoleAdmin, IsJoined: true},
	}
}

func memberGroup() GroupItem {
	return GroupItem{
		Group:   GroupInfo{GroupID: "g1", Name: "test"},
		MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true},
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

func TestServiceUpdateRequiresOwner(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceUpdateRejectsNotFound(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceUpdateHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
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
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}}
	svc := NewServiceWithRepo(repo)
	items, _, err := svc.ListMembers(context.Background(), GroupMembersRequest{GroupID: "g1", Limit: 150})
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
	_, err := svc.JoinGroup(grpCtx("u1"), JoinGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceJoinGroupAlreadyJoined(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	result, err := svc.JoinGroup(grpCtx("u1"), JoinGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != JoinGroupStatusJoined {
		t.Fatalf("expected joined status, got %s", result.Status)
	}
}

func TestServiceJoinGroupHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}}, getFound: true, joinResult: JoinGroupResult{Status: JoinGroupStatusJoined}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.JoinGroup(grpCtx("u1"), JoinGroupRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != JoinGroupStatusJoined {
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
	err := svc.BanMember(grpCtx("u1"), BanMemberRequest{GroupID: "g1", UserID: "u1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
	}
}

func TestServiceBanMemberRequiresAdmin(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.BanMember(grpCtx("u1"), BanMemberRequest{GroupID: "g1", UserID: "u2"})
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
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, inviteResult: Invitation{InviteID: "inv1"}}
	svc := NewServiceWithRepo(repo)
	inv, err := svc.InviteMember(grpCtx("u1"), InviteMemberRequest{GroupID: "g1", UserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.InviteID != "inv1" {
		t.Fatalf("expected inv1, got %s", inv.InviteID)
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
	items, _, err := svc.ListMyGroups(grpCtx("u1"), MyGroupsRequest{Limit: 150})
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

func TestServiceUpdateMySettingsRejectsNotMember(t *testing.T) {
	repo := &stubGroupsRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}}, getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.UpdateMySettings(grpCtx("u1"), UpdateGroupMySettingsRequest{GroupID: "g1"})
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("expected not member, got %v", err)
	}
}

func TestServiceUpdateMySettingsHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: memberGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.UpdateMySettings(grpCtx("u1"), UpdateGroupMySettingsRequest{GroupID: "g1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
