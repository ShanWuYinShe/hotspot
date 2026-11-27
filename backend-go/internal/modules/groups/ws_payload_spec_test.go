package groups

// ws_payload_spec_test.go — groups 模块 WS payload 与 HTTP 申请项的 JSON 形状
// 契约（spec api_spec_websocket.md / api_spec_group.md）。恒定 nullable 字段
// 必须显式下发 null，必填 UserInfo 不得缺键。

import (
	"encoding/json"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// assertPayloadJSONKeys 断言 payload 序列化后的键集合与期望完全一致（防止
// 多字段 / 缺字段），并返回键到原始 JSON 值的映射供取值断言。
func assertPayloadJSONKeys(t *testing.T, payload any, wantKeys ...string) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("payload is not a JSON object: %v (%s)", err, raw)
	}
	if len(fields) != len(wantKeys) {
		t.Fatalf("payload keys = %v, want %v (raw: %s)", payloadKeyList(fields), wantKeys, raw)
	}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Fatalf("payload missing key %q (raw: %s)", key, raw)
		}
	}
	return fields
}

func payloadKeyList(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	return keys
}

func TestGroupMemberRoleChangedPayloadJSONShape(t *testing.T) {
	fields := assertPayloadJSONKeys(t, GroupMemberRoleChangedPayload{
		GroupID:   "g1",
		User:      types.UserInfo{UserID: "u2", Nickname: "成员"},
		OldRole:   string(GroupRoleMember),
		NewRole:   string(GroupRoleAdmin),
		ChangedBy: types.UserInfo{UserID: "u1", Nickname: "群主"},
		ChangedAt: "2026-11-28T00:00:00Z",
	}, "group_id", "user", "old_role", "new_role", "changed_by", "changed_at")
	if string(fields["old_role"]) != `"member"` || string(fields["new_role"]) != `"admin"` {
		t.Fatalf("unexpected roles: %s / %s", fields["old_role"], fields["new_role"])
	}
}

func TestGroupMemberLeftPayloadJSONShape(t *testing.T) {
	fields := assertPayloadJSONKeys(t, GroupMemberLeftPayload{
		GroupID:     "g1",
		User:        types.UserInfo{UserID: "u2", Nickname: "成员"},
		LeftAt:      "2026-11-28T00:00:00Z",
		MemberCount: 3,
	}, "group_id", "user", "left_at", "member_count")
	if string(fields["member_count"]) != "3" {
		t.Fatalf("unexpected member_count: %s", fields["member_count"])
	}
}

func TestGroupMemberMutedPayloadJSONShape(t *testing.T) {
	fields := assertPayloadJSONKeys(t, GroupMemberMutedPayload{
		GroupID:         "g1",
		User:            types.UserInfo{UserID: "u2", Nickname: "成员"},
		MutedBy:         types.UserInfo{UserID: "u1", Nickname: "群主"},
		MutedUntil:      "2026-11-28T00:30:00Z",
		DurationMinutes: 30,
		MutedAt:         "2026-11-28T00:00:00Z",
	}, "group_id", "user", "muted_by", "muted_until", "duration_minutes", "muted_at")
	if string(fields["muted_until"]) != `"2026-11-28T00:30:00Z"` {
		t.Fatalf("unexpected muted_until: %s", fields["muted_until"])
	}
}

func TestGroupMemberRemovedPayloadJSONShape(t *testing.T) {
	fields := assertPayloadJSONKeys(t, GroupMemberRemovedPayload{
		GroupID:     "g1",
		User:        types.UserInfo{UserID: "u2", Nickname: "成员"},
		RemovedBy:   types.UserInfo{UserID: "u1", Nickname: "群主"},
		Reason:      "kicked",
		RemovedAt:   "2026-11-28T00:00:00Z",
		MemberCount: 2,
	}, "group_id", "user", "removed_by", "reason", "removed_at", "member_count")
	if string(fields["user"]) == "null" || string(fields["removed_by"]) == "null" {
		t.Fatalf("user/removed_by must be UserInfo objects: %s / %s", fields["user"], fields["removed_by"])
	}
}

func TestGroupApplicationCancelledPayloadJSONShape(t *testing.T) {
	fields := assertPayloadJSONKeys(t, GroupApplicationCancelledPayload{
		ApplicationID: "app1",
		GroupID:       "g1",
		Applicant:     types.UserInfo{UserID: "u2", Nickname: "申请人"},
		CancelledAt:   "2026-11-28T00:00:00Z",
		Reason:        nil,
	}, "application_id", "group_id", "applicant", "cancelled_at", "reason")
	if string(fields["reason"]) != "null" {
		t.Fatalf("reason must be null when absent, got %s", fields["reason"])
	}
}

// groupPayloadFixture 是「群主 + 一名直加入成员」的实时广播夹具：groupSink
// 订阅群频道、memberSink 是被操作用户的定向通道。
type groupPayloadFixture struct {
	groupID    string
	repo       *InMemoryRepository
	service    *Service
	groupSink  *groupRealtimeSink
	memberSink *groupRealtimeSink
}

func newGroupPayloadFixture(t *testing.T) *groupPayloadFixture {
	t.Helper()
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	created, err := service.Create(grpCtx("u_owner"), CreateGroupRequest{Name: "payload-group", Location: Location{LocationName: "somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	groupID := created.Group.GroupID
	if _, err := service.JoinGroup(grpCtx("u_member"), EnterGroupRequest{GroupID: groupID}); err != nil {
		t.Fatalf("join group: %v", err)
	}
	groupSink := &groupRealtimeSink{}
	hub.SetGroupSubscriptions([]string{groupID}, groupSink)
	memberSink := &groupRealtimeSink{}
	if !hub.RegisterUser("u_member", "sess_member", memberSink) {
		t.Fatal("register member sink failed")
	}
	return &groupPayloadFixture{groupID: groupID, repo: repo, service: service, groupSink: groupSink, memberSink: memberSink}
}

func TestSetMemberRoleBroadcastsOldAndNewRole(t *testing.T) {
	fixture := newGroupPayloadFixture(t)
	if err := fixture.service.SetMemberRole(grpCtx("u_owner"), SetMemberRoleRequest{GroupID: fixture.groupID, UserID: "u_member", Role: GroupRoleAdmin}); err != nil {
		t.Fatalf("set member role: %v", err)
	}
	if len(fixture.groupSink.events) != 1 {
		t.Fatalf("expected 1 group event, got %#v", fixture.groupSink.events)
	}
	payload, ok := fixture.groupSink.events[0].Payload.(GroupMemberRoleChangedPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", fixture.groupSink.events[0].Payload)
	}
	if payload.OldRole != string(GroupRoleMember) || payload.NewRole != string(GroupRoleAdmin) {
		t.Fatalf("unexpected old/new role: %#v", payload)
	}
	if len(fixture.memberSink.events) != 1 {
		t.Fatalf("target user must also receive the event: %#v", fixture.memberSink.events)
	}
}

func TestTransferOwnershipBroadcastsPreviousAndNewRole(t *testing.T) {
	fixture := newGroupPayloadFixture(t)
	if err := fixture.service.SetMemberRole(grpCtx("u_owner"), SetMemberRoleRequest{GroupID: fixture.groupID, UserID: "u_member", Role: GroupRoleAdmin}); err != nil {
		t.Fatalf("promote member to admin: %v", err)
	}
	fixture.groupSink.events = nil
	if err := fixture.service.TransferOwnership(grpCtx("u_owner"), TransferOwnershipRequest{GroupID: fixture.groupID, NewOwnerID: "u_member"}); err != nil {
		t.Fatalf("transfer ownership: %v", err)
	}
	if len(fixture.groupSink.events) != 1 {
		t.Fatalf("expected 1 group event, got %#v", fixture.groupSink.events)
	}
	payload, ok := fixture.groupSink.events[0].Payload.(GroupMemberRoleChangedPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", fixture.groupSink.events[0].Payload)
	}
	if payload.OldRole != string(GroupRoleAdmin) || payload.NewRole != string(GroupRoleOwner) {
		t.Fatalf("unexpected old/new role on transfer: %#v", payload)
	}
	if payload.User.UserID != "u_member" || payload.ChangedBy.UserID != "u_owner" {
		t.Fatalf("unexpected user/changed_by: %#v", payload)
	}
}

func TestMuteMemberBroadcastsMutedUntilMatchingViewerState(t *testing.T) {
	fixture := newGroupPayloadFixture(t)
	if err := fixture.service.MuteMember(grpCtx("u_owner"), MuteMemberRequest{GroupID: fixture.groupID, UserID: "u_member", DurationMinutes: 30}); err != nil {
		t.Fatalf("mute member: %v", err)
	}
	if len(fixture.groupSink.events) != 1 {
		t.Fatalf("expected 1 group event, got %#v", fixture.groupSink.events)
	}
	payload, ok := fixture.groupSink.events[0].Payload.(GroupMemberMutedPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", fixture.groupSink.events[0].Payload)
	}
	if payload.MutedUntil == "" {
		t.Fatal("muted_until must be present in payload")
	}
	if _, err := time.Parse(time.RFC3339, payload.MutedUntil); err != nil {
		t.Fatalf("muted_until is not RFC3339: %q (%v)", payload.MutedUntil, err)
	}
	item, ok, err := fixture.repo.Get(grpCtx("u_member"), fixture.groupID)
	if err != nil || !ok || item.ViewerState == nil || item.ViewerState.MutedUntil == nil {
		t.Fatalf("member viewer state missing muted_until: %#v err=%v", item.ViewerState, err)
	}
	if *item.ViewerState.MutedUntil != payload.MutedUntil {
		t.Fatalf("WS muted_until (%s) must equal HTTP viewer_state.muted_until (%s)", payload.MutedUntil, *item.ViewerState.MutedUntil)
	}
}

func TestLeaveGroupBroadcastsRemainingMemberCount(t *testing.T) {
	fixture := newGroupPayloadFixture(t)
	if err := fixture.service.LeaveGroup(grpCtx("u_member"), LeaveGroupRequest{GroupID: fixture.groupID}); err != nil {
		t.Fatalf("leave group: %v", err)
	}
	if len(fixture.groupSink.events) != 1 {
		t.Fatalf("expected 1 group event, got %#v", fixture.groupSink.events)
	}
	payload, ok := fixture.groupSink.events[0].Payload.(GroupMemberLeftPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", fixture.groupSink.events[0].Payload)
	}
	if payload.MemberCount != 1 {
		t.Fatalf("remaining member_count = %d, want 1", payload.MemberCount)
	}
	if payload.User.UserID != "u_member" {
		t.Fatalf("unexpected user: %#v", payload.User)
	}
}

func TestRemoveMemberBroadcastsUserAndOperatorOnBothChannels(t *testing.T) {
	fixture := newGroupPayloadFixture(t)
	if err := fixture.service.RemoveMember(grpCtx("u_owner"), RemoveMemberRequest{GroupID: fixture.groupID, UserID: "u_member"}); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if len(fixture.groupSink.events) != 1 || len(fixture.memberSink.events) != 1 {
		t.Fatalf("both channels must receive the event: group=%#v user=%#v", fixture.groupSink.events, fixture.memberSink.events)
	}
	for name, events := range map[string][]realtime.Event{"group": fixture.groupSink.events, "user": fixture.memberSink.events} {
		payload, ok := events[0].Payload.(GroupMemberRemovedPayload)
		if !ok {
			t.Fatalf("%s channel payload type = %T", name, events[0].Payload)
		}
		if payload.User.UserID != "u_member" || payload.RemovedBy.UserID != "u_owner" {
			t.Fatalf("%s channel must carry user and operator: %#v", name, payload)
		}
	}
}

func TestListApplicationsFillsPendingExpiresAtFromTTL(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	created, err := service.Create(grpCtx("u_owner"), CreateGroupRequest{Name: "approval-group", RequiresApproval: boolPtr(true), Location: Location{LocationName: "somewhere"}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	groupID := created.Group.GroupID
	if _, err := service.JoinGroup(grpCtx("u_applicant"), EnterGroupRequest{GroupID: groupID, Message: types.StrPtr("please")}); err != nil {
		t.Fatalf("apply to join: %v", err)
	}
	items, _, err := service.ListApplications(grpCtx("u_owner"), GroupApplicationsRequest{GroupID: groupID})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 application, got %#v", items)
	}
	item := items[0]
	if item.ExpiresAt == nil {
		t.Fatalf("pending application must carry expires_at: %#v", item)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	wantExpiresAt := createdAt.Add(72 * time.Hour).UTC().Format(time.RFC3339)
	if *item.ExpiresAt != wantExpiresAt {
		t.Fatalf("expires_at = %s, want created_at+72h = %s", *item.ExpiresAt, wantExpiresAt)
	}

	all := ApplicationStatus("all")
	if err := service.ApproveApplication(grpCtx("u_owner"), ApproveGroupApplicationRequest{GroupID: groupID, ApplicationID: item.ApplicationID}); err != nil {
		t.Fatalf("approve application: %v", err)
	}
	handled, _, err := service.ListApplications(grpCtx("u_owner"), GroupApplicationsRequest{GroupID: groupID, Status: &all})
	if err != nil {
		t.Fatalf("list handled applications: %v", err)
	}
	if len(handled) != 1 {
		t.Fatalf("want 1 handled application, got %#v", handled)
	}
	if handled[0].Status == ApplicationStatusPending {
		t.Fatalf("application must leave pending after approve: %#v", handled[0])
	}
	if handled[0].ExpiresAt != nil {
		t.Fatalf("handled application expires_at must be null per spec: %#v", handled[0])
	}
}
