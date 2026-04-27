package groups

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/modules/notifications"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type groupErrorRepo struct {
	getResult             GroupItem
	getFound              bool
	createErr             error
	getErr                error
	searchErr             error
	updateErr             error
	dismissErr            error
	listMembersErr        error
	searchMembersErr      error
	listMyGroupsErr       error
	listMyGroupsItems     []GroupItem
	joinGroupErr          error
	createAnnouncementErr error
	listAnnouncementsErr  error
	updateAnnouncementErr error
	deleteAnnouncementErr error
	listApplicationsErr   error
	listApplicationsItems []GroupApplicationItem
	approveApplicationErr error
	rejectApplicationErr  error
	listMyInvitesItems    []GroupInviteItem
	rejectInviteErr       error
}

func (r *groupErrorRepo) Create(context.Context, string, CreateGroupRequest) (GroupItem, error) {
	return GroupItem{}, r.createErr
}
func (r *groupErrorRepo) Get(context.Context, string) (GroupItem, bool, error) {
	return r.getResult, r.getFound, r.getErr
}
func (r *groupErrorRepo) Search(context.Context, SearchGroupsRequest) (PaginatedGroups, error) {
	return PaginatedGroups{}, r.searchErr
}
func (r *groupErrorRepo) Update(context.Context, string, UpdateGroupRequest) error {
	return r.updateErr
}
func (r *groupErrorRepo) Dismiss(context.Context, string) error {
	return r.dismissErr
}
func (r *groupErrorRepo) ListMembers(context.Context, string, *GroupRole, int, string) ([]GroupMemberItem, string, error) {
	return nil, "", r.listMembersErr
}
func (r *groupErrorRepo) SearchMembers(context.Context, string, string, *GroupRole, int, string) ([]GroupMemberItem, string, error) {
	return nil, "", r.searchMembersErr
}
func (r *groupErrorRepo) ListMyGroups(context.Context, string, int, string) ([]GroupItem, string, error) {
	return r.listMyGroupsItems, "", r.listMyGroupsErr
}
func (r *groupErrorRepo) JoinGroup(context.Context, string, string, *string, *string) (JoinGroupResult, error) {
	return JoinGroupResult{}, r.joinGroupErr
}
func (r *groupErrorRepo) CreateAnnouncement(context.Context, string, CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	return GroupAnnouncementItem{}, r.createAnnouncementErr
}
func (r *groupErrorRepo) ListAnnouncements(context.Context, string) ([]GroupAnnouncementItem, error) {
	return nil, r.listAnnouncementsErr
}
func (r *groupErrorRepo) UpdateAnnouncement(context.Context, string, UpdateGroupAnnouncementRequest) error {
	return r.updateAnnouncementErr
}
func (r *groupErrorRepo) DeleteAnnouncement(context.Context, string, string) error {
	return r.deleteAnnouncementErr
}
func (r *groupErrorRepo) ListApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error) {
	return r.listApplicationsItems, "", r.listApplicationsErr
}
func (r *groupErrorRepo) ApproveApplication(context.Context, string, string) error {
	return r.approveApplicationErr
}
func (r *groupErrorRepo) RejectApplication(context.Context, string, *string) error {
	return r.rejectApplicationErr
}
func (r *groupErrorRepo) UpdateMySettings(context.Context, string, UpdateGroupMySettingsRequest) error {
	return nil
}
func (r *groupErrorRepo) LeaveGroup(context.Context, string, string) error {
	return nil
}
func (r *groupErrorRepo) ListMyApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error) {
	return nil, "", nil
}
func (r *groupErrorRepo) CancelApplication(context.Context, string) error {
	return nil
}
func (r *groupErrorRepo) ListMyInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupInviteItem, string, error) {
	return r.listMyInvitesItems, "", nil
}
func (r *groupErrorRepo) AcceptInvite(context.Context, string, string) error {
	return nil
}
func (r *groupErrorRepo) RejectInvite(context.Context, string) error {
	return r.rejectInviteErr
}
func (r *groupErrorRepo) InviteMember(context.Context, string, InviteMemberRequest) (Invitation, error) {
	return Invitation{}, nil
}
func (r *groupErrorRepo) ListSentInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupSentInviteItem, string, error) {
	return nil, "", nil
}
func (r *groupErrorRepo) RevokeInvite(context.Context, string, string) error {
	return nil
}
func (r *groupErrorRepo) SetMemberRole(context.Context, string, SetMemberRoleRequest) error {
	return nil
}
func (r *groupErrorRepo) MuteMember(context.Context, string, MuteMemberRequest) error {
	return nil
}
func (r *groupErrorRepo) UnmuteMember(context.Context, string, UnmuteMemberRequest) error {
	return nil
}
func (r *groupErrorRepo) BanMember(context.Context, string, BanMemberRequest) error {
	return nil
}
func (r *groupErrorRepo) UnbanMember(context.Context, string, UnbanMemberRequest) error {
	return nil
}
func (r *groupErrorRepo) ListBannedMembers(context.Context, string, int, string) ([]BannedMemberItem, string, error) {
	return nil, "", nil
}
func (r *groupErrorRepo) RemoveMember(context.Context, string, RemoveMemberRequest) error {
	return nil
}
func (r *groupErrorRepo) TransferOwnership(context.Context, string, TransferOwnershipRequest) error {
	return nil
}

func TestCreateAndOpenGroup(t *testing.T) {
	handler := NewHandler(NewService())
	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"name":"Hot Group","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"}}`))
	createReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	ctx := platformauth.WithPrincipal(createReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)

	var createResp struct {
		Code int       `json:"code"`
		Data GroupItem `json:"data"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Code != 0 || createResp.Data.Group.GroupID == "" {
		t.Fatalf("unexpected create response: %#v", createResp)
	}

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"group_id":"`+createResp.Data.Group.GroupID+`"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)

	var openResp struct {
		Code int       `json:"code"`
		Data GroupItem `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 || openResp.Data.Group.Name != "Hot Group" {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
}

func TestSearchGroupsReturnsMatches(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	if _, err := service.Create(ctx, CreateGroupRequest{Name: "Searchable", Location: Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 31.2, Longitude: 121.5}, LocationName: "Here"}}); err != nil {
		t.Fatalf("create group: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Searchable","reference_point":{"latitude":31.2,"longitude":121.5}}`))
	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	var resp struct {
		Code int             `json:"code"`
		Data PaginatedGroups `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) == 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.Data.Items[0].Location.Distance == nil || resp.Data.Items[0].Location.BearingDegrees == nil {
		t.Fatalf("expected geo enrichment, got %#v", resp.Data.Items[0].Location)
	}
}

func TestSearchGroupsValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing criteria", body: `{"reference_point":{"latitude":31.2,"longitude":121.5}}`},
		{name: "radius out of range", body: `{"radius":99,"reference_point":{"latitude":31.2,"longitude":121.5}}`},
		{name: "invalid sort", body: `{"keyword":"abc","sort_by":"random","reference_point":{"latitude":31.2,"longitude":121.5}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.Search(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

func TestGroupsHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{createErr: errors.New("boom"), getErr: errors.New("boom"), searchErr: errors.New("boom")}))

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"name":"Hot Group","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"}}`))
	createReq.Header.Set("Idempotency-Key", "efgh1234efgh1234efgh1234efgh1234")
	createReq = createReq.WithContext(platformauth.WithPrincipal(createReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	assertGroupCode(t, createRR, 10600)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"group_id":"group_1"}`))
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)
	assertGroupCode(t, openRR, 10600)

	searchReq := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Hot Group"}`))
	searchRR := httptest.NewRecorder()
	handler.Search(searchRR, searchReq)
	assertGroupCode(t, searchRR, 10600)
}

func TestDismissGroupUsesGroupIDAndBroadcastsDismissedEvent(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	created, err := service.Create(ctx, CreateGroupRequest{Name: "Dismiss Me", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	sink := &groupRealtimeSink{}
	hub.RegisterUser("user_alice", "sess_alice", sink)
	if err := service.Dismiss(ctx, DismissGroupRequest{GroupID: created.Group.GroupID}); err != nil {
		t.Fatalf("dismiss group: %v", err)
	}
	if _, ok, err := repo.Get(context.Background(), created.Group.GroupID); err != nil || ok {
		t.Fatalf("expected group removed, got ok=%v err=%v", ok, err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != "GroupDismissed" {
		t.Fatalf("unexpected realtime events: %#v", sink.events)
	}
	payload, ok := sink.events[0].Payload.(map[string]any)
	if !ok || payload["group_id"] != created.Group.GroupID || payload["reason"] != "owner_dismissed" {
		t.Fatalf("unexpected dismiss payload: %#v", sink.events[0].Payload)
	}
}

func TestJoinApprovalSendsApplicationNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	repo := NewInMemoryRepository()
	service := NewServiceWithDeps(repo, notifications.NewServiceWithRepo(notificationRepo), nil)
	ownerCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_owner"})
	joinMethod := GroupJoinMethodApproval
	created, err := service.Create(ownerCtx, CreateGroupRequest{Name: "Approval Group", JoinMethod: joinMethod, Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	ownerSink := &groupRealtimeSink{}
	hub.RegisterUser("user_alice", "sess_owner", ownerSink)
	memberCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_member"})
	message := "please let me in"
	result, err := service.JoinGroup(memberCtx, JoinGroupRequest{GroupID: created.Group.GroupID, Message: &message})
	if err != nil {
		t.Fatalf("join group: %v", err)
	}
	if result.Status != JoinGroupStatusPendingApproval || result.Application == nil {
		t.Fatalf("unexpected join result: %#v", result)
	}
	if len(ownerSink.events) != 1 || ownerSink.events[0].Type != "GroupApplicationReceived" {
		t.Fatalf("unexpected owner events: %#v", ownerSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "group_application_received" {
		t.Fatalf("unexpected notifications: %#v", notificationsPage)
	}
}

func TestInviteAndAcceptFlowSendsNotifications(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	repo := NewInMemoryRepository()
	service := NewServiceWithDeps(repo, notifications.NewServiceWithRepo(notificationRepo), nil)
	ownerCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_owner"})
	created, err := service.Create(ownerCtx, CreateGroupRequest{Name: "Invite Group", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	inviteeSink := &groupRealtimeSink{}
	ownerSink := &groupRealtimeSink{}
	hub.RegisterUser("user_bob", "sess_bob", inviteeSink)
	hub.RegisterUser("user_alice", "sess_owner", ownerSink)
	message := "join us"
	invitation, err := service.InviteMember(ownerCtx, InviteMemberRequest{GroupID: created.Group.GroupID, UserID: "user_bob", Message: &message})
	if err != nil {
		t.Fatalf("invite member: %v", err)
	}
	if invitation.InviteID == "" {
		t.Fatalf("expected invite id, got %#v", invitation)
	}
	inviteNotifications, err := notificationRepo.List(context.Background(), "user_bob", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list invite notifications: %v", err)
	}
	if len(inviteeSink.events) != 1 || inviteeSink.events[0].Type != "GroupInviteReceived" {
		t.Fatalf("unexpected invitee events: %#v", inviteeSink.events)
	}
	if len(inviteNotifications.Items) != 1 || inviteNotifications.Items[0].Type != "group_invite_received" {
		t.Fatalf("unexpected invite notifications: %#v", inviteNotifications)
	}
	inviteeCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_bob"})
	if err := service.AcceptInvite(inviteeCtx, AcceptInviteRequest{InviteID: invitation.InviteID}); err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	ownerNotifications, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list owner notifications: %v", err)
	}
	if len(ownerSink.events) != 1 || ownerSink.events[0].Type != "GroupInviteResponse" {
		t.Fatalf("unexpected owner events: %#v", ownerSink.events)
	}
	if len(ownerNotifications.Items) != 1 || ownerNotifications.Items[0].Type != "group_invite_accepted" {
		t.Fatalf("unexpected owner notifications: %#v", ownerNotifications)
	}
}

func TestApproveApplicationSendsAcceptedNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	repo := NewInMemoryRepository()
	service := NewServiceWithDeps(repo, notifications.NewServiceWithRepo(notificationRepo), nil)
	ownerCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_owner"})
	joinMethod := GroupJoinMethodApproval
	created, err := service.Create(ownerCtx, CreateGroupRequest{Name: "Approval Group", JoinMethod: joinMethod, Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	memberCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_member"})
	result, err := service.JoinGroup(memberCtx, JoinGroupRequest{GroupID: created.Group.GroupID})
	if err != nil {
		t.Fatalf("join group: %v", err)
	}
	memberSink := &groupRealtimeSink{}
	hub.RegisterUser("user_bob", "sess_member", memberSink)
	if err := service.ApproveApplication(ownerCtx, ApproveGroupApplicationRequest{ApplicationID: result.Application.ApplicationID}); err != nil {
		t.Fatalf("approve application: %v", err)
	}
	memberNotifications, err := notificationRepo.List(context.Background(), "user_bob", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list member notifications: %v", err)
	}
	if len(memberSink.events) != 1 || memberSink.events[0].Type != "GroupApplicationResponse" {
		t.Fatalf("unexpected member events: %#v", memberSink.events)
	}
	if len(memberNotifications.Items) != 1 || memberNotifications.Items[0].Type != "group_application_accepted" {
		t.Fatalf("unexpected member notifications: %#v", memberNotifications)
	}
}

func TestApproveApplicationHandlerMapsSpecificErrors(t *testing.T) {
	tests := []struct {
		name     string
		repo     *groupErrorRepo
		expected int
	}{
		{
			name:     "not found",
			repo:     &groupErrorRepo{},
			expected: 10300,
		},
		{
			name:     "access restricted",
			repo:     &groupErrorRepo{listMyGroupsItems: []GroupItem{{Group: GroupInfo{GroupID: "group_1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}}}, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", Status: ApplicationStatusPending}}},
			expected: 10200,
		},
		{
			name:     "already processed",
			repo:     &groupErrorRepo{listMyGroupsItems: []GroupItem{{Group: GroupInfo{GroupID: "group_1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}}}, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", Status: ApplicationStatusAccepted}}},
			expected: 10400,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(NewServiceWithRepo(tc.repo))
			req := httptest.NewRequest(http.MethodPost, "/applications/approve", strings.NewReader(`{"application_id":"app_1"}`))
			req.Header.Set("Idempotency-Key", fmt.Sprintf("approve-app-%024d", tc.expected))
			req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
			rr := httptest.NewRecorder()
			handler.ApproveApplication(rr, req)
			assertGroupCode(t, rr, tc.expected)
		})
	}
}

func TestRejectApplicationHandlerMapsSpecificErrors(t *testing.T) {
	tests := []struct {
		name     string
		repo     *groupErrorRepo
		expected int
	}{
		{
			name:     "not found",
			repo:     &groupErrorRepo{},
			expected: 10300,
		},
		{
			name:     "access restricted",
			repo:     &groupErrorRepo{listMyGroupsItems: []GroupItem{{Group: GroupInfo{GroupID: "group_1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}}}, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", Status: ApplicationStatusPending}}},
			expected: 10200,
		},
		{
			name:     "already processed",
			repo:     &groupErrorRepo{listMyGroupsItems: []GroupItem{{Group: GroupInfo{GroupID: "group_1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}}}, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", Status: ApplicationStatusRejected}}},
			expected: 10400,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(NewServiceWithRepo(tc.repo))
			req := httptest.NewRequest(http.MethodPost, "/applications/reject", strings.NewReader(`{"application_id":"app_1"}`))
			req.Header.Set("Idempotency-Key", fmt.Sprintf("reject-app--%024d", tc.expected))
			req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
			rr := httptest.NewRecorder()
			handler.RejectApplication(rr, req)
			assertGroupCode(t, rr, tc.expected)
		})
	}
}

func TestRejectInviteHandlerMapsSpecificErrors(t *testing.T) {
	tests := []struct {
		name     string
		repo     *groupErrorRepo
		expected int
	}{
		{
			name:     "not found",
			repo:     &groupErrorRepo{},
			expected: 10300,
		},
		{
			name:     "expired",
			repo:     &groupErrorRepo{listMyInvitesItems: []GroupInviteItem{{InviteID: "invite_1", Status: ApplicationStatusRejected}}},
			expected: 10400,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(NewServiceWithRepo(tc.repo))
			req := httptest.NewRequest(http.MethodPost, "/me/invites/reject", strings.NewReader(`{"invite_id":"invite_1"}`))
			req.Header.Set("Idempotency-Key", fmt.Sprintf("reject-inv-%024d", tc.expected))
			req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
			rr := httptest.NewRecorder()
			handler.RejectInvite(rr, req)
			assertGroupCode(t, rr, tc.expected)
		})
	}
}

func TestCreateGroupRecordsResourceGuard(t *testing.T) {
	checker := &groupRecordingChecker{}
	resourceguard.SetDefaultChecker(checker)
	defer resourceguard.SetDefaultChecker(resourceguard.NoopChecker{})

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	service := NewServiceWithRepo(NewInMemoryRepository())
	if _, err := service.Create(ctx, CreateGroupRequest{Name: "Guarded Group", Location: Location{LocationName: "Somewhere"}}); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if len(checker.recorded) != 1 || !strings.HasPrefix(checker.recorded[0], "group:") {
		t.Fatalf("unexpected records: %#v", checker.recorded)
	}
}

func assertGroupCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}

type groupRealtimeSink struct{ events []realtime.Event }

func (s *groupRealtimeSink) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *groupRealtimeSink) Close() error                { return nil }

type groupRecordingChecker struct {
	recorded []string
}

func (c *groupRecordingChecker) Ready(context.Context) bool { return true }
func (c *groupRecordingChecker) MayExist(context.Context, resourceguard.ResourceKind, string) (bool, error) {
	return true, nil
}
func (c *groupRecordingChecker) Record(_ context.Context, kind resourceguard.ResourceKind, id string) error {
	c.recorded = append(c.recorded, string(kind)+":"+id)
	return nil
}

// --- Members ---

func TestListMembersHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/list", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMembers(rr, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []GroupMemberItem `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestListMembersHandlerMapsGroupNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/members/list", strings.NewReader(`{"group_id":"missing"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMembers(rr, req)
	assertGroupCode(t, rr, 10300)
}

func TestListMembersHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{}`},
		{name: "invalid json", body: `{`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/members/list", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.ListMembers(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

func TestSearchMembersHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/search", strings.NewReader(`{"group_id":"g1","keyword":"alice"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SearchMembers(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestSearchMembersHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{"keyword":"alice"}`},
		{name: "missing keyword", body: `{"group_id":"g1"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/members/search", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.SearchMembers(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

func TestUnmuteMemberHandlerMapsAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/unmute", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UnmuteMember(rr, req)
	assertGroupCode(t, rr, 10200)
}

func TestUnmuteMemberHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{"user_id":"u2"}`},
		{name: "missing user_id", body: `{"group_id":"g1"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/members/unmute", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.UnmuteMember(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

func TestBanMemberHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/ban", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	req.Header.Set("Idempotency-Key", "banmember123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.BanMember(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestBanMemberHandlerMapsAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/ban", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	req.Header.Set("Idempotency-Key", "banmember-restricted-123456789012345")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.BanMember(rr, req)
	assertGroupCode(t, rr, 10200)
}

func TestBanMemberHandlerCannotBanSelf(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/ban", strings.NewReader(`{"group_id":"g1","user_id":"user_alice"}`))
	req.Header.Set("Idempotency-Key", "ban-self-1234567890123456789012345678")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.BanMember(rr, req)
	assertGroupCode(t, rr, 10200)
}

func TestUnbanMemberHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/unban", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UnbanMember(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestUnbanMemberHandlerMapsGroupNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/members/unban", strings.NewReader(`{"group_id":"missing","user_id":"u2"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UnbanMember(rr, req)
	assertGroupCode(t, rr, 10300)
}

func TestListBannedMembersHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/list_banned", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListBannedMembers(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestListBannedMembersHandlerMapsAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/list_banned", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListBannedMembers(rr, req)
	assertGroupCode(t, rr, 10200)
}

func TestListSentInvitesHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/invites/list", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListSentInvites(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestListSentInvitesHandlerMapsAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/invites/list", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListSentInvites(rr, req)
	assertGroupCode(t, rr, 10200)
}

func TestRevokeInviteHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/invites/revoke", strings.NewReader(`{"group_id":"g1","invite_id":"inv1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeInvite(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestRevokeInviteHandlerMapsGroupNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/members/invites/revoke", strings.NewReader(`{"group_id":"missing","invite_id":"inv1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeInvite(rr, req)
	assertGroupCode(t, rr, 10300)
}

// --- Announcements ---

func TestCreateAnnouncementHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"group_id":"g1","title":"Notice","content":"Hello"}`))
	req.Header.Set("Idempotency-Key", "create-ann-123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.CreateAnnouncement(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestCreateAnnouncementHandlerMapsAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"group_id":"g1","title":"Notice","content":"Hello"}`))
	req.Header.Set("Idempotency-Key", "create-ann-restricted-123456789012345")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.CreateAnnouncement(rr, req)
	assertGroupCode(t, rr, 10200)
}

func TestCreateAnnouncementHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{"title":"Notice","content":"Hello"}`},
		{name: "missing title", body: `{"group_id":"g1","content":"Hello"}`},
		{name: "missing content", body: `{"group_id":"g1","title":"Notice"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(tc.body))
			req.Header.Set("Idempotency-Key", fmt.Sprintf("ann-val-%d-123456789012345678901234", len(tc.name)))
			rr := httptest.NewRecorder()
			handler.CreateAnnouncement(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

func TestListAnnouncementsHandlerReturnsSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/announcements/list", strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.ListAnnouncements(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestListAnnouncementsHandlerMapsGroupNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{listAnnouncementsErr: ErrGroupNotFound}))
	req := httptest.NewRequest(http.MethodPost, "/announcements/list", strings.NewReader(`{"group_id":"missing"}`))
	rr := httptest.NewRecorder()
	handler.ListAnnouncements(rr, req)
	assertGroupCode(t, rr, 10300)
}

func TestListAnnouncementsHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/announcements/list", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.ListAnnouncements(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

func TestUpdateAnnouncementHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(`{"group_id":"g1","announcement_id":"ann1","title":"Updated"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UpdateAnnouncement(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestUpdateAnnouncementHandlerMapsAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleMember, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(`{"group_id":"g1","announcement_id":"ann1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UpdateAnnouncement(rr, req)
	assertGroupCode(t, rr, 10200)
}

func TestUpdateAnnouncementHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{"announcement_id":"ann1"}`},
		{name: "missing announcement_id", body: `{"group_id":"g1"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.UpdateAnnouncement(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

func TestDeleteAnnouncementHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(`{"group_id":"g1","announcement_id":"ann1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.DeleteAnnouncement(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestDeleteAnnouncementHandlerMapsGroupNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{deleteAnnouncementErr: ErrGroupNotFound}))
	req := httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(`{"group_id":"missing","announcement_id":"ann1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.DeleteAnnouncement(rr, req)
	assertGroupCode(t, rr, 10300)
}

func TestDeleteAnnouncementHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{"announcement_id":"ann1"}`},
		{name: "missing announcement_id", body: `{"group_id":"g1"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.DeleteAnnouncement(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}

// --- Me ---

func TestListMyGroupsHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		listMyGroupsItems: []GroupItem{{Group: GroupInfo{GroupID: "g1"}, MyState: &MyGroupState{Role: GroupRoleOwner, IsJoined: true}}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/me/list", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyGroups(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestListMyGroupsHandlerMapsServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{listMyGroupsErr: errors.New("boom")}))
	req := httptest.NewRequest(http.MethodPost, "/me/list", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyGroups(rr, req)
	assertGroupCode(t, rr, 10600)
}

func TestJoinGroupHandlerReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	created, err := NewServiceWithRepo(repo).Create(ctx, CreateGroupRequest{Name: "Open Group", JoinMethod: GroupJoinMethodOpen, Location: Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/me/join", strings.NewReader(`{"group_id":"`+created.Group.GroupID+`"}`))
	req.Header.Set("Idempotency-Key", "join-group-123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_2"}))
	rr := httptest.NewRecorder()
	handler.JoinGroup(rr, req)

	var resp struct {
		Code int             `json:"code"`
		Data JoinGroupResult `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.Status != JoinGroupStatusJoined {
		t.Fatalf("expected joined status, got %s", resp.Data.Status)
	}
}

func TestJoinGroupHandlerMapsGroupNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/me/join", strings.NewReader(`{"group_id":"missing"}`))
	req.Header.Set("Idempotency-Key", "join-missing-123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.JoinGroup(rr, req)
	assertGroupCode(t, rr, 10300)
}

func TestJoinGroupHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/me/join", strings.NewReader(tc.body))
			req.Header.Set("Idempotency-Key", fmt.Sprintf("join-val-%d-123456789012345678901234", len(tc.name)))
			rr := httptest.NewRecorder()
			handler.JoinGroup(rr, req)
			assertGroupCode(t, rr, 10000)
		})
	}
}
