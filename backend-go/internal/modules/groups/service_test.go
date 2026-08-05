package groups

import (
	"context"
	"errors"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
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
func (r *stubGroupsRepo) IsMember(_ context.Context, _ string, userID string) (bool, error) {
	for _, m := range r.members {
		if m.User.UserID == userID {
			return true, nil
		}
	}
	return false, nil
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
	_, err := svc.Update(grpCtx("u1"), UpdateGroupRequest{GroupID: "g1"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected access restricted, got %v", err)
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
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, members: []GroupMemberItem{{User: UserInfo{UserID: "u2"}}}}
	svc := NewServiceWithRepo(repo)
	items, _, err := svc.ListMembers(context.Background(), GroupMembersRequest{GroupID: "g1", Pagination: &PaginationParams{Limit: 150}})
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

// =========================================================================
// A) service_invite 补充测试
// =========================================================================

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
	// AcceptInvite only checks if the invite was found (via findInviteForUser);
	// it does NOT check invite.Status before calling AcceptInvite.
	// So it should succeed when acceptErr is nil.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{ApplicationID: "app1"})
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
	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{ApplicationID: "missing"})
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
	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{ApplicationID: "app1"})
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
	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{ApplicationID: "missing"})
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
	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{ApplicationID: "app1", Reason: &reason})
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
	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{ApplicationID: "app1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("expected db error, got %v", err)
	}
}

// --- SetMemberRole ---

func TestServiceSetMemberRoleHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.SetMemberRole(grpCtx("u1"), SetMemberRoleRequest{GroupID: "g1", UserID: "u2", Role: GroupRoleAdmin})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSetMemberRolePropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, setRoleErr: errors.New("db error")}
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
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.MuteMember(grpCtx("u1"), MuteMemberRequest{GroupID: "g1", UserID: "u2", DurationMinutes: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceMuteMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, muteErr: errors.New("db error")}
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
	err := svc.BanMember(grpCtx("u1"), BanMemberRequest{GroupID: "g1", UserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceBanMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, banErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.BanMember(grpCtx("u1"), BanMemberRequest{GroupID: "g1", UserID: "u2"})
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
	err := svc.UnbanMember(grpCtx("u1"), UnbanMemberRequest{GroupID: "g1", UserID: "u2"})
	if !errors.Is(err, ErrAccessRestricted) {
		t.Fatalf("expected ErrAccessRestricted, got %v", err)
	}
}

func TestServiceUnbanMemberHappyPath(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.UnbanMember(grpCtx("u1"), UnbanMemberRequest{GroupID: "g1", UserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUnbanMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, unbanErr: errors.New("db error")}
	svc := NewServiceWithRepo(repo)
	err := svc.UnbanMember(grpCtx("u1"), UnbanMemberRequest{GroupID: "g1", UserID: "u2"})
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
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.RemoveMember(grpCtx("u1"), RemoveMemberRequest{GroupID: "g1", UserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceRemoveMemberPropagatesRepoError(t *testing.T) {
	repo := &stubGroupsRepo{getResult: ownerGroup(), getFound: true, removeErr: errors.New("db error")}
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
			{ApplicationID: "app1", Applicant: UserInfo{UserID: "u1"}},
		},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.CancelApplication(grpCtx("u1"), CancelApplicationRequest{ApplicationID: "app1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceCancelApplicationPropagatesError(t *testing.T) {
	repo := &stubGroupsRepo{
		cancelErr: errors.New("db error"),
		apps: []GroupApplicationItem{
			{ApplicationID: "app1", Applicant: UserInfo{UserID: "u1"}},
		},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.CancelApplication(grpCtx("u1"), CancelApplicationRequest{ApplicationID: "app1"})
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
	payload, ok := sink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload not a map: %#v", sink.events[0].Payload)
	}
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
