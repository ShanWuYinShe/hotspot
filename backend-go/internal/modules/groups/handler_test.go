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
	"time"

	"hotspot/backend-go/internal/modules/notify"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type groupErrorRepo struct {
	Repository            // 嵌入接口：未显式 stub 的方法走 nil 接口（不应被覆盖测试调用）
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
	announceItems         []GroupAnnouncementItem
	listAnnouncementsErr  error
	updateAnnouncementErr error
	deleteAnnouncementErr error
	listApplicationsErr   error
	listApplicationsItems []GroupApplicationItem
	cancelApplicationErr  error
	approveApplicationErr error
	rejectApplicationErr  error
	listMyInvitesItems    []GroupInviteItem
	listMyInvitesErr      error
	rejectInviteErr       error
	revokeInviteErr       error
	transferOwnershipErr  error
	acceptInviteErr       error
	removeMemberErr       error
	isMemberResult        bool
	isMemberErr           error
	isBannedResult        bool
	isBannedErr           error
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
func (r *groupErrorRepo) IsMember(context.Context, string, string) (bool, error) {
	return r.isMemberResult, r.isMemberErr
}
func (r *groupErrorRepo) IsBanned(context.Context, string, string) (bool, error) {
	return r.isBannedResult, r.isBannedErr
}
func (r *groupErrorRepo) GetApplication(_ context.Context, applicationID string) (GroupApplicationItem, bool, error) {
	for _, a := range r.listApplicationsItems {
		if a.ApplicationID == applicationID {
			return a, true, nil
		}
	}
	return GroupApplicationItem{}, false, nil
}
func (r *groupErrorRepo) SearchMembers(context.Context, string, string, *GroupRole, int, string) ([]GroupMemberItem, string, error) {
	return nil, "", r.searchMembersErr
}
func (r *groupErrorRepo) ListMyGroups(context.Context, string, int, string) ([]GroupItem, string, error) {
	return r.listMyGroupsItems, "", r.listMyGroupsErr
}
func (r *groupErrorRepo) JoinGroup(context.Context, string, string, *string) (EnterGroupResult, error) {
	return EnterGroupResult{}, r.joinGroupErr
}
func (r *groupErrorRepo) CreateAnnouncement(context.Context, string, CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	return GroupAnnouncementItem{}, r.createAnnouncementErr
}
func (r *groupErrorRepo) ListAnnouncements(context.Context, string, int) ([]GroupAnnouncementItem, error) {
	return r.announceItems, r.listAnnouncementsErr
}
func (r *groupErrorRepo) GetAnnouncement(_ context.Context, _, announcementID string) (GroupAnnouncementItem, bool, error) {
	for _, item := range r.announceItems {
		if item.AnnouncementID == announcementID {
			return item, true, nil
		}
	}
	return GroupAnnouncementItem{}, false, r.listAnnouncementsErr
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
func (r *groupErrorRepo) UpdatePreferences(context.Context, string, UpdateGroupPreferencesRequest) error {
	return nil
}
func (r *groupErrorRepo) LeaveGroup(context.Context, string, string) error {
	return nil
}
func (r *groupErrorRepo) ListMyApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error) {
	return nil, "", nil
}
func (r *groupErrorRepo) CancelApplication(context.Context, string) error {
	return r.cancelApplicationErr
}
func (r *groupErrorRepo) ListMyInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupInviteItem, string, error) {
	return r.listMyInvitesItems, "", r.listMyInvitesErr
}
func (r *groupErrorRepo) GetInvite(_ context.Context, inviteID string) (GroupInviteItem, string, bool, error) {
	for _, item := range r.listMyInvitesItems {
		if item.InviteID == inviteID {
			return item, "user_alice", true, nil
		}
	}
	return GroupInviteItem{}, "", false, nil
}
func (r *groupErrorRepo) AcceptInvite(context.Context, string, string) error {
	return r.acceptInviteErr
}
func (r *groupErrorRepo) RejectInvite(context.Context, string) error {
	return r.rejectInviteErr
}
func (r *groupErrorRepo) InviteMember(context.Context, string, InviteMemberRequest, *time.Time) (Invitation, error) {
	return Invitation{}, r.createErr
}
func (r *groupErrorRepo) ListSentInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupSentInviteItem, string, error) {
	return nil, "", nil
}
func (r *groupErrorRepo) RevokeInvite(context.Context, string, string) error {
	return r.revokeInviteErr
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
func (r *groupErrorRepo) ListBannedMembers(context.Context, string, int, string) ([]AdminBannedMemberItem, string, error) {
	return nil, "", nil
}
func (r *groupErrorRepo) RemoveMember(context.Context, string, RemoveMemberRequest) error {
	return r.removeMemberErr
}
func (r *groupErrorRepo) TransferOwnership(context.Context, string, TransferOwnershipRequest) error {
	return r.transferOwnershipErr
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
		Code httpx.Code `json:"code"`
		Data GroupItem  `json:"data"`
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
		Code httpx.Code `json:"code"`
		Data GroupItem  `json:"data"`
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
		Code httpx.Code      `json:"code"`
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
		{name: "keyword too long", body: `{"keyword":"` + strings.Repeat("a", 101) + `","reference_point":{"latitude":31.2,"longitude":121.5}}`},
		{name: "tag too long", body: `{"tag":"` + strings.Repeat("a", 31) + `","reference_point":{"latitude":31.2,"longitude":121.5}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.Search(rr, req)
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
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
	assertGroupCode(t, createRR, httpx.CodeInternalUnavailable)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"group_id":"group_1"}`))
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)
	assertGroupCode(t, openRR, httpx.CodeInternalUnavailable)

	searchReq := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Hot Group"}`))
	searchRR := httptest.NewRecorder()
	handler.Search(searchRR, searchReq)
	assertGroupCode(t, searchRR, httpx.CodeInternalUnavailable)
}

// 游标非法属于参数错误（10000），不得落到 10600：用内存实现走真实游标
// 解码路径覆盖 ListMembers 与 ListSentInvites 两个代表端点。
func TestHandlerListMapsInvalidCursorToParamError(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:       true,
		isMemberResult: true,
		listMembersErr: ErrInvalidCursor,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	membersReq := httptest.NewRequest(http.MethodPost, "/members/fetch", strings.NewReader(`{"group_id":"g1","pagination":{"limit":10,"cursor":"@@malformed@@"}}`))
	membersReq = membersReq.WithContext(ctx)
	membersRR := httptest.NewRecorder()
	handler.ListMembers(membersRR, membersReq)
	assertGroupCode(t, membersRR, httpx.CodeInvalidRequest)
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
	hub.SetGroupSubscriptions([]string{created.Group.GroupID}, sink)
	if err := service.Dismiss(ctx, DismissGroupRequest{GroupID: created.Group.GroupID}); err != nil {
		t.Fatalf("dismiss group: %v", err)
	}
	if _, ok, err := repo.Get(context.Background(), created.Group.GroupID); err != nil || ok {
		t.Fatalf("expected group removed, got ok=%v err=%v", ok, err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != "GroupDismissed" {
		t.Fatalf("unexpected realtime events: %#v", sink.events)
	}
	payload := decodePayload(t, sink.events[0].Payload)
	if payload["group_id"] != created.Group.GroupID || payload["reason"] != "owner_dismissed" {
		t.Fatalf("unexpected dismiss payload: %#v", sink.events[0].Payload)
	}
}

func TestJoinApprovalSendsApplicationNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notify.NewInMemoryRepository()
	repo := NewInMemoryRepository()
	service := NewServiceWithDeps(repo, notify.NewServiceWithRepo(notificationRepo), nil)
	ownerCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_owner"})
	approval := true
	created, err := service.Create(ownerCtx, CreateGroupRequest{Name: "Approval Group", JoinMethod: GroupJoinMethodDirect, RequiresApproval: &approval, Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	ownerSink := &groupRealtimeSink{}
	hub.RegisterUser("user_alice", "sess_owner", ownerSink)
	memberCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_member"})
	message := "please let me in"
	result, err := service.JoinGroup(memberCtx, EnterGroupRequest{GroupID: created.Group.GroupID, Message: &message})
	if err != nil {
		t.Fatalf("join group: %v", err)
	}
	if result.Status != EnterGroupStatusPendingApproval || result.ApplicationID == nil {
		t.Fatalf("unexpected join result: %#v", result)
	}
	if len(ownerSink.events) != 1 || ownerSink.events[0].Type != "GroupApplicationReceived" {
		t.Fatalf("unexpected owner events: %#v", ownerSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notify.NotifyListRequest{})
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
	notificationRepo := notify.NewInMemoryRepository()
	repo := NewInMemoryRepository()
	service := NewServiceWithDeps(repo, notify.NewServiceWithRepo(notificationRepo), nil)
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
	if invitation.InviteID == nil || *invitation.InviteID == "" {
		t.Fatalf("expected invite id, got %#v", invitation)
	}
	inviteNotifications, err := notificationRepo.List(context.Background(), "user_bob", notify.NotifyListRequest{})
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
	if err := service.AcceptInvite(inviteeCtx, AcceptInviteRequest{InviteID: *invitation.InviteID}); err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	ownerNotifications, err := notificationRepo.List(context.Background(), "user_alice", notify.NotifyListRequest{})
	if err != nil {
		t.Fatalf("list owner notifications: %v", err)
	}
	// 邀请接受是纯实时事件（spec：GroupInviteResponse 不落库通知），
	// owner 只收到 WS 事件，通知中心无记录。
	if len(ownerSink.events) != 1 || ownerSink.events[0].Type != "GroupInviteResponse" {
		t.Fatalf("unexpected owner events: %#v", ownerSink.events)
	}
	if len(ownerNotifications.Items) != 0 {
		t.Fatalf("expected no owner notifications, got %#v", ownerNotifications)
	}
}

func TestApproveApplicationSendsAcceptedNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notify.NewInMemoryRepository()
	repo := NewInMemoryRepository()
	service := NewServiceWithDeps(repo, notify.NewServiceWithRepo(notificationRepo), nil)
	ownerCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_owner"})
	approval := true
	created, err := service.Create(ownerCtx, CreateGroupRequest{Name: "Approval Group", JoinMethod: GroupJoinMethodDirect, RequiresApproval: &approval, Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	memberCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_member"})
	result, err := service.JoinGroup(memberCtx, EnterGroupRequest{GroupID: created.Group.GroupID})
	if err != nil {
		t.Fatalf("join group: %v", err)
	}
	memberSink := &groupRealtimeSink{}
	hub.RegisterUser("user_bob", "sess_member", memberSink)
	if err := service.ApproveApplication(ownerCtx, ApproveGroupApplicationRequest{GroupID: created.Group.GroupID, ApplicationID: *result.ApplicationID}); err != nil {
		t.Fatalf("approve application: %v", err)
	}
	memberNotifications, err := notificationRepo.List(context.Background(), "user_bob", notify.NotifyListRequest{})
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
		expected httpx.Code
	}{
		{
			name:     "not found",
			repo:     &groupErrorRepo{},
			expected: httpx.CodeNotFound,
		},
		{
			name:     "access restricted",
			repo:     &groupErrorRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "group_1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}}, getFound: true, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", GroupID: "group_1", Status: ApplicationStatusPending}}},
			expected: httpx.CodeForbidden,
		},
		{
			name:     "already processed",
			repo:     &groupErrorRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "group_1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}}, getFound: true, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", GroupID: "group_1", Status: ApplicationStatusAccepted}}},
			expected: httpx.CodeConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(NewServiceWithRepo(tc.repo))
			req := httptest.NewRequest(http.MethodPost, "/applications/approve", strings.NewReader(`{"group_id":"g1","application_id":"app_1"}`))
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
		expected httpx.Code
	}{
		{
			name:     "not found",
			repo:     &groupErrorRepo{},
			expected: httpx.CodeNotFound,
		},
		{
			name:     "access restricted",
			repo:     &groupErrorRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "group_1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}}, getFound: true, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", GroupID: "group_1", Status: ApplicationStatusPending}}},
			expected: httpx.CodeForbidden,
		},
		{
			name:     "already processed",
			repo:     &groupErrorRepo{getResult: GroupItem{Group: GroupInfo{GroupID: "group_1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}}, getFound: true, listApplicationsItems: []GroupApplicationItem{{ApplicationID: "app_1", GroupID: "group_1", Status: ApplicationStatusRejected}}},
			expected: httpx.CodeConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(NewServiceWithRepo(tc.repo))
			req := httptest.NewRequest(http.MethodPost, "/applications/reject", strings.NewReader(`{"group_id":"g1","application_id":"app_1"}`))
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
		expected httpx.Code
	}{
		{
			name:     "not found",
			repo:     &groupErrorRepo{},
			expected: httpx.CodeNotFound,
		},
		{
			name:     "expired",
			repo:     &groupErrorRepo{listMyInvitesItems: []GroupInviteItem{{InviteID: "invite_1", Status: ApplicationStatusRejected}}},
			expected: httpx.CodeConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(NewServiceWithRepo(tc.repo))
			req := httptest.NewRequest(http.MethodPost, "/groups/invites/reject", strings.NewReader(`{"invite_id":"invite_1"}`))
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

func assertGroupCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
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
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:       true,
		isMemberResult: true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/list", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMembers(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
	assertGroupCode(t, rr, httpx.CodeNotFound)
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
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

func TestSearchMembersHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:       true,
		isMemberResult: true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/search", strings.NewReader(`{"group_id":"g1","keyword":"alice"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SearchMembers(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestSearchMembersDeniesNonMemberWhenNotSearchable(t *testing.T) {
	// spec api_spec_group.md members/search：未加入群组且 members_searchable=false
	// → 10200 + forbidden_reason=not_group_member（由 10400 迁移，api_spec_errors.md）。
	repo := &groupErrorRepo{
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}},
		getFound:       true,
		isMemberResult: false,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/search", strings.NewReader(`{"group_id":"g1","keyword":"alice"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SearchMembers(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestSearchMembersAllowsNonMemberWhenSearchable(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}, IsSearchable: true, MembersSearchable: true},
		getFound:       true,
		isMemberResult: false,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/search", strings.NewReader(`{"group_id":"g1","keyword":"alice"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SearchMembers(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
		{name: "keyword too long", body: `{"group_id":"g1","keyword":"` + strings.Repeat("a", 101) + `"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/members/search", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.SearchMembers(rr, req)
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

func TestSetMuteUnmuteMapsAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/set_mute", strings.NewReader(`{"group_id":"g1","user_id":"u2","active":false}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SetMute(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestSetMuteHandlerValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	tests := []struct {
		name string
		body string
	}{
		{name: "missing group_id", body: `{"user_id":"u2","active":false}`},
		{name: "missing user_id", body: `{"group_id":"g1","active":false}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/members/set_mute", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.SetMute(rr, req)
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

func TestBanMemberHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/set_ban", strings.NewReader(`{"group_id":"g1","user_id":"u2","active":true}`))
	req.Header.Set("Idempotency-Key", "banmember123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SetMemberBan(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/set_ban", strings.NewReader(`{"group_id":"g1","user_id":"u2","active":true}`))
	req.Header.Set("Idempotency-Key", "banmember-restricted-123456789012345")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SetMemberBan(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestBanMemberHandlerCannotBanSelf(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/set_ban", strings.NewReader(`{"group_id":"g1","user_id":"user_alice","active":true}`))
	req.Header.Set("Idempotency-Key", "ban-self-1234567890123456789012345678")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SetMemberBan(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestUnbanMemberHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/set_ban", strings.NewReader(`{"group_id":"g1","user_id":"u2","active":false}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SetMemberBan(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
	req := httptest.NewRequest(http.MethodPost, "/members/set_ban", strings.NewReader(`{"group_id":"missing","user_id":"u2","active":false}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.SetMemberBan(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestListBannedMembersHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/list_banned", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListBannedMembers(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/list_banned", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListBannedMembers(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestListSentInvitesHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/groups/admin/invites/fetch_sent", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListSentInvites(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/groups/admin/invites/fetch_sent", strings.NewReader(`{"group_id":"g1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListSentInvites(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestRevokeInviteHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/groups/admin/invites/revoke", strings.NewReader(`{"group_id":"g1","invite_id":"inv1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeInvite(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
	req := httptest.NewRequest(http.MethodPost, "/groups/admin/invites/revoke", strings.NewReader(`{"group_id":"missing","invite_id":"inv1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeInvite(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestRevokeInviteHandlerMapsExpiredInvite(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:       GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:        true,
		revokeInviteErr: ErrInviteExpired,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/groups/admin/invites/revoke", strings.NewReader(`{"group_id":"g1","invite_id":"inv1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeInvite(rr, req)
	assertGroupCode(t, rr, httpx.CodeConflict)
}

// --- Announcements ---

func TestCreateAnnouncementHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"group_id":"g1","title":"Notice","content":"Hello"}`))
	req.Header.Set("Idempotency-Key", "create-ann-123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.CreateAnnouncement(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"group_id":"g1","title":"Notice","content":"Hello"}`))
	req.Header.Set("Idempotency-Key", "create-ann-restricted-123456789012345")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.CreateAnnouncement(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
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
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

func TestListAnnouncementsHandlerReturnsSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/announcements/list", strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.ListAnnouncements(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
	assertGroupCode(t, rr, httpx.CodeNotFound)
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
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

func TestUpdateAnnouncementHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(`{"group_id":"g1","announcement_id":"ann1","title":"Updated"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UpdateAnnouncement(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(`{"group_id":"g1","announcement_id":"ann1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UpdateAnnouncement(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
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
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

func TestDeleteAnnouncementHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(`{"group_id":"g1","announcement_id":"ann1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.DeleteAnnouncement(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
	assertGroupCode(t, rr, httpx.CodeNotFound)
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
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

// --- Me ---

func TestListMyGroupsHandlerReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		listMyGroupsItems: []GroupItem{{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/groups/fetch_joined", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyGroups(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
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
	req := httptest.NewRequest(http.MethodPost, "/groups/fetch_joined", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyGroups(rr, req)
	assertGroupCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestJoinGroupHandlerReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	created, err := NewServiceWithRepo(repo).Create(ctx, CreateGroupRequest{Name: "Open Group", JoinMethod: GroupJoinMethodDirect, Location: Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/join", strings.NewReader(`{"group_id":"`+created.Group.GroupID+`"}`))
	req.Header.Set("Idempotency-Key", "join-group-123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_2"}))
	rr := httptest.NewRecorder()
	handler.JoinGroup(rr, req)

	var resp struct {
		Code httpx.Code       `json:"code"`
		Data EnterGroupResult `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.Status != EnterGroupStatusJoined {
		t.Fatalf("expected joined status, got %s", resp.Data.Status)
	}
}

func TestJoinGroupHandlerMapsGroupNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/join", strings.NewReader(`{"group_id":"missing"}`))
	req.Header.Set("Idempotency-Key", "join-missing-123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.JoinGroup(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
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
			req := httptest.NewRequest(http.MethodPost, "/join", strings.NewReader(tc.body))
			req.Header.Set("Idempotency-Key", fmt.Sprintf("join-val-%d-123456789012345678901234", len(tc.name)))
			rr := httptest.NewRecorder()
			handler.JoinGroup(rr, req)
			assertGroupCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

// --- Update / Dismiss / MySettings / Leave / Applications ---

func TestHandlerUpdateReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"group_id":"g1","name":"Updated Name"}`))
	req.Header.Set("Idempotency-Key", "update_success_groups_aaaaaaaaaaaa")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerUpdateRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{invalid`))
	req.Header.Set("Idempotency-Key", "update_invalid_json_groups_bbbbbbbb")
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerUpdateReturns10300WhenNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"group_id":"missing"}`))
	req.Header.Set("Idempotency-Key", "update_not_found_groups_cccccccccc")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerUpdateReturns10200WhenAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"group_id":"g1"}`))
	req.Header.Set("Idempotency-Key", "update_restricted_groups_dddddddddd")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestHandlerDismissReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/dismiss", strings.NewReader(`{"group_id":"g1"}`))
	req.Header.Set("Idempotency-Key", "dismiss-ok-1234567890123456789012345")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.Dismiss(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerDismissReachesServiceWithoutIdempotencyKey(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/dismiss", strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.Dismiss(rr, req)
	// Without an Idempotency-Key the request still reaches the service and
	// returns the business error (group "g1" does not exist here).
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerDismissRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/dismiss", strings.NewReader(`{broken`))
	req.Header.Set("Idempotency-Key", "dismiss-invalid-12345678901234567890123")
	rr := httptest.NewRecorder()
	handler.Dismiss(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerDismissReturns10300WhenNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/dismiss", strings.NewReader(`{"group_id":"missing"}`))
	req.Header.Set("Idempotency-Key", "dismiss-nf-123456789012345678901234")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.Dismiss(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerUpdatePreferencesReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/update_preferences", strings.NewReader(`{"group_id":"g1","is_muted":true}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UpdatePreferences(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerUpdatePreferencesRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/update_preferences", strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()
	handler.UpdatePreferences(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerUpdatePreferencesReturns10300WhenNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/update_preferences", strings.NewReader(`{"group_id":"missing"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.UpdatePreferences(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerLeaveGroupReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/leave", strings.NewReader(`{"group_id":"g1"}`))
	req.Header.Set("Idempotency-Key", "leave_success_groups_eeeeeeeeeeeee")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.LeaveGroup(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerLeaveGroupRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/leave", strings.NewReader(`{broken`))
	req.Header.Set("Idempotency-Key", "leave_invalid_json_groups_ffffffff")
	rr := httptest.NewRecorder()
	handler.LeaveGroup(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerLeaveGroupReturns10300WhenNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/leave", strings.NewReader(`{"group_id":"missing"}`))
	req.Header.Set("Idempotency-Key", "leave_not_found_groups_ggggggggggg")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.LeaveGroup(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerListMyApplicationsReturnsSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/groups/applications/fetch_submitted", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyApplications(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerListMyApplicationsRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/groups/applications/fetch_submitted", strings.NewReader(`{invalid`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyApplications(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerCancelApplicationReturnsSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{
		listApplicationsItems: []GroupApplicationItem{
			{ApplicationID: "app_1", GroupID: "g1", Applicant: UserInfo{UserID: "user_alice"}},
		},
	}))
	req := httptest.NewRequest(http.MethodPost, "/groups/applications/cancel", strings.NewReader(`{"group_id":"g1","application_id":"app_1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.CancelApplication(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerCancelApplicationRejectsMissingApplicationID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/groups/applications/cancel", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.CancelApplication(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerCancelApplicationMapsAlreadyProcessed(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{
		listApplicationsItems: []GroupApplicationItem{
			{ApplicationID: "app_1", GroupID: "g1", Applicant: UserInfo{UserID: "user_alice"}},
		},
		cancelApplicationErr: ErrAlreadyProcessed,
	}))
	req := httptest.NewRequest(http.MethodPost, "/groups/applications/cancel", strings.NewReader(`{"group_id":"g1","application_id":"app_1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.CancelApplication(rr, req)
	assertGroupCode(t, rr, httpx.CodeConflict)
}

// --- TransferOwnership ---

func TestHandlerTransferOwnershipReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:       true,
		isMemberResult: true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/transfer_ownership", strings.NewReader(`{"group_id":"g1","new_owner_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.TransferOwnership(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerTransferOwnershipRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/transfer_ownership", strings.NewReader(`{"new_owner_id":"user_bob"}`))
	rr := httptest.NewRecorder()
	handler.TransferOwnership(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerTransferOwnershipRejectsMissingNewOwnerID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/transfer_ownership", strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.TransferOwnership(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerTransferOwnershipReturns10300WhenNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{transferOwnershipErr: ErrGroupNotFound}))
	req := httptest.NewRequest(http.MethodPost, "/transfer_ownership", strings.NewReader(`{"group_id":"missing","new_owner_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.TransferOwnership(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerTransferOwnershipReturns10200WhenAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/transfer_ownership", strings.NewReader(`{"group_id":"g1","new_owner_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.TransferOwnership(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestHandlerTransferOwnershipReturns10400WhenTargetNotMember(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:       true,
		isMemberResult: false,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/transfer_ownership", strings.NewReader(`{"group_id":"g1","new_owner_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.TransferOwnership(rr, req)
	assertGroupCode(t, rr, httpx.CodeConflict)
}

func TestHandlerTransferOwnershipReturns10600OnUnknownError(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:            GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:             true,
		isMemberResult:       true,
		transferOwnershipErr: errors.New("boom"),
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/transfer_ownership", strings.NewReader(`{"group_id":"g1","new_owner_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.TransferOwnership(rr, req)
	assertGroupCode(t, rr, httpx.CodeInternalUnavailable)
}

// --- ListMyInvites ---

func TestHandlerListMyInvitesReturnsSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{
		listMyInvitesItems: []GroupInviteItem{
			{InviteID: "inv1", Group: GroupInfo{GroupID: "g1"}, Status: ApplicationStatusPending},
		},
	}))
	req := httptest.NewRequest(http.MethodPost, "/groups/invites/fetch_received", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyInvites(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items []GroupInviteItem `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 invite, got %d", len(resp.Data.Items))
	}
}

func TestHandlerListMyInvitesRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/groups/invites/fetch_received", strings.NewReader(`{broken`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyInvites(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerListMyInvitesReturns10600OnError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{listMyInvitesErr: errors.New("boom")}))
	req := httptest.NewRequest(http.MethodPost, "/groups/invites/fetch_received", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.ListMyInvites(rr, req)
	assertGroupCode(t, rr, httpx.CodeInternalUnavailable)
}

// --- AcceptInvite ---

func TestHandlerAcceptInviteReturnsSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{
		listMyInvitesItems: []GroupInviteItem{
			{InviteID: "inv1", Group: GroupInfo{GroupID: "g1"}, Status: ApplicationStatusPending},
		},
	}))
	req := httptest.NewRequest(http.MethodPost, "/groups/invites/accept", strings.NewReader(`{"invite_id":"inv1"}`))
	req.Header.Set("Idempotency-Key", "accept-invite-12345678901234567890")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.AcceptInvite(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerAcceptInviteRejectsMissingInviteID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/groups/invites/accept", strings.NewReader(`{}`))
	req.Header.Set("Idempotency-Key", "accept-invite-missing-12345678901234")
	rr := httptest.NewRecorder()
	handler.AcceptInvite(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerAcceptInviteReturns10300WhenNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{acceptInviteErr: ErrGroupNotFound}))
	req := httptest.NewRequest(http.MethodPost, "/groups/invites/accept", strings.NewReader(`{"invite_id":"missing"}`))
	req.Header.Set("Idempotency-Key", "accept-invite-nf-12345678901234567890")
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.AcceptInvite(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

// --- RemoveMember ---

func TestHandlerRemoveMemberReturnsSuccess(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:      GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:       true,
		isMemberResult: true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "ann1",
			GroupID:        "g1",
			Title:          "Updated",
		}},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/remove", strings.NewReader(`{"group_id":"g1","user_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RemoveMember(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerRemoveMemberRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/members/remove", strings.NewReader(`{"user_id":"user_bob"}`))
	rr := httptest.NewRecorder()
	handler.RemoveMember(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerRemoveMemberRejectsMissingUserID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/members/remove", strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.RemoveMember(rr, req)
	assertGroupCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerRemoveMemberReturns10300WhenNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{removeMemberErr: ErrGroupNotFound}))
	req := httptest.NewRequest(http.MethodPost, "/members/remove", strings.NewReader(`{"group_id":"missing","user_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RemoveMember(rr, req)
	assertGroupCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerRemoveMemberReturns10200WhenAccessRestricted(t *testing.T) {
	repo := &groupErrorRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}},
		getFound:  true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/remove", strings.NewReader(`{"group_id":"g1","user_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RemoveMember(rr, req)
	assertGroupCode(t, rr, httpx.CodeForbidden)
}

func TestHandlerRemoveMemberReturns10600OnUnknownError(t *testing.T) {
	repo := &groupErrorRepo{
		getResult:       GroupItem{Group: GroupInfo{GroupID: "g1"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:        true,
		isMemberResult:  true,
		removeMemberErr: errors.New("boom"),
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/members/remove", strings.NewReader(`{"group_id":"g1","user_id":"user_bob"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RemoveMember(rr, req)
	assertGroupCode(t, rr, httpx.CodeInternalUnavailable)
}
