package groups

// 本文件覆盖 spec 对齐新增/修改的群邀请/申请通知逻辑:
// RevokeInvite 的 GroupInviteRevoked 事件与通知落库、RejectInvite 的
// GroupInviteRejected 事件与通知落库、邀请/申请惰性过期
// (GroupInviteExpired / GroupApplicationExpired)、CancelApplication 对管理员的
// GroupApplicationCancelled 广播、RejectApplication 对申请人的
// GroupApplicationRejected 广播、resolveUserInfo 与 applicationExpired。
// 断言依据 spec/api_spec_websocket.md 与 spec/api_spec_notify.md。

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// --- stubs ---

// sinkStub 捕获 hub 投递的事件。
type sinkStub struct{ events []realtime.Event }

func (s *sinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error                { return nil }

// captureGroupsRepo 内嵌 stubGroupsRepo 并额外捕获 RevokeInvite /
// CancelApplication 的调用(用于断言惰性过期路径)。
type captureGroupsRepo struct {
	*stubGroupsRepo
	revokeInviteCalls    []revokeInviteCall
	cancelApplicationIDs []string
}

type revokeInviteCall struct {
	groupID  string
	inviteID string
}

func (r *captureGroupsRepo) RevokeInvite(_ context.Context, groupID, inviteID string) error {
	r.revokeInviteCalls = append(r.revokeInviteCalls, revokeInviteCall{groupID, inviteID})
	return r.revokeInvErr
}

func (r *captureGroupsRepo) CancelApplication(_ context.Context, applicationID string) error {
	r.cancelApplicationIDs = append(r.cancelApplicationIDs, applicationID)
	return r.cancelErr
}

// notifyWriterStub 实现 groupNotificationWriter 并捕获本次 spec 对齐的调用。
type notifyWriterStub struct {
	applicationRejected []groupApplicationRejectedCall
}

type groupApplicationRejectedCall struct {
	targetUserID  string
	groupID       string
	groupName     string
	applicationID string
	rejectedBy    types.UserInfo
	reason        *string
}

func (w *notifyWriterStub) AppendGroupInviteReceived(context.Context, string, types.UserInfo, string, string, string, *string) (string, error) {
	return "notif_written", nil
}

func (w *notifyWriterStub) AppendGroupApplicationReceived(context.Context, string, types.UserInfo, string, string, string, *string) (string, error) {
	return "notif_written", nil
}

func (w *notifyWriterStub) AppendGroupApplicationAccepted(context.Context, string, string, string, string) (string, error) {
	return "notif_written", nil
}

func (w *notifyWriterStub) AppendGroupApplicationRejected(_ context.Context, targetUserID, groupID, groupName, applicationID string, reason *string) (string, error) {
	w.applicationRejected = append(w.applicationRejected, groupApplicationRejectedCall{targetUserID, groupID, groupName, applicationID, types.UserInfo{}, reason})
	return "notif_written", nil
}

// actorProviderStub 模拟 users 模块的 GetActorInfo。
type actorProviderStub struct {
	info actor.Info
	err  error
}

func (p actorProviderStub) GetActorInfo(context.Context, string) (actor.Info, error) {
	return p.info, p.err
}

// appConfigStub 模拟 app config 源(用于 applicationExpired 的 TTL 覆盖)。
type appConfigStub struct{ ttlHours int }

func (s appConfigStub) Get(context.Context) appconfig.AppConfig {
	return appconfig.AppConfig{Limits: appconfig.Limits{GroupApplicationTTLHours: s.ttlHours}}
}

// withTestHub 安装独立 hub,测试结束后恢复原默认 hub。
func withTestHub(t *testing.T) *realtime.Hub {
	t.Helper()
	previous := realtime.DefaultHub()
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	t.Cleanup(func() { realtime.SetDefaultHub(previous) })
	return hub
}

func registerSink(hub *realtime.Hub, userID string) *sinkStub {
	sink := &sinkStub{}
	hub.RegisterUser(userID, "sess_"+userID, sink)
	return sink
}

// --- RevokeInvite ---

func TestServiceRevokeInviteNotifiesInvitee(t *testing.T) {
	pending := ApplicationStatusPending
	reason := "no longer needed"
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		getResult:      ownerGroup(),
		getFound:       true,
		sentInvites:    []GroupSentInviteItem{{InviteID: "inv1", Status: pending, Invitee: UserInfo{UserID: "u2"}}},
		sentInvitesErr: nil,
	}}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	inviteeSink := registerSink(hub, "u2")

	err := svc.RevokeInvite(grpCtx("u1"), RevokeSentInviteRequest{GroupID: "g1", InviteID: "inv1", Reason: &reason})
	if err != nil {
		t.Fatalf("revoke invite: %v", err)
	}
	if len(repo.revokeInviteCalls) != 1 || repo.revokeInviteCalls[0] != (revokeInviteCall{groupID: "g1", inviteID: "inv1"}) {
		t.Fatalf("unexpected revoke calls: %#v", repo.revokeInviteCalls)
	}
	if len(inviteeSink.events) != 1 {
		t.Fatalf("expected 1 event for invitee, got %d", len(inviteeSink.events))
	}
	event := inviteeSink.events[0]
	if event.Type != realtime.EventTypeGroupInviteRevoked {
		t.Fatalf("expected GroupInviteRevoked, got %q", event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %#v", event.Payload)
	}
	for _, key := range []string{"invite_id", "group_id", "revoked_by", "revoked_at"} {
		if payload[key] == nil {
			t.Fatalf("expected payload key %q, got %#v", key, payload)
		}
	}
	if payload["invite_id"] != "inv1" || payload["reason"] != reason {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
}

func TestServiceRevokeInviteSkipsNotificationWhenInviteeUnknown(t *testing.T) {
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		getResult:   ownerGroup(),
		getFound:    true,
		sentInvites: nil, // 无匹配 pending 邀请 => 找不到 invitee
	}}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	withTestHub(t)

	if err := svc.RevokeInvite(grpCtx("u1"), RevokeSentInviteRequest{GroupID: "g1", InviteID: "inv1"}); err != nil {
		t.Fatalf("revoke invite: %v", err)
	}
	if len(repo.revokeInviteCalls) != 1 {
		t.Fatalf("expected revoke call regardless of invitee, got %#v", repo.revokeInviteCalls)
	}
}

// --- RejectInvite ---

func TestServiceRejectInviteNotifiesInviter(t *testing.T) {
	pending := ApplicationStatusPending
	reason := "not interested"
	repo := &stubGroupsRepo{
		myInvites: []GroupInviteItem{{
			InviteID: "inv1",
			Status:   pending,
			Group:    GroupInfo{GroupID: "g1", Name: "test"},
			Inviter:  UserInfo{UserID: "u2"},
		}},
	}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	inviterSink := registerSink(hub, "u2")

	err := svc.RejectInvite(grpCtx("u1"), RejectInviteRequest{InviteID: "inv1", Reason: &reason})
	if err != nil {
		t.Fatalf("reject invite: %v", err)
	}
	if len(inviterSink.events) != 1 {
		t.Fatalf("expected 1 event for inviter, got %d", len(inviterSink.events))
	}
	event := inviterSink.events[0]
	if event.Type != realtime.EventTypeGroupInviteRejected {
		t.Fatalf("expected GroupInviteRejected, got %q", event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %#v", event.Payload)
	}
	for _, key := range []string{"invite_id", "group_id", "rejected_by", "rejected_at"} {
		if payload[key] == nil {
			t.Fatalf("expected payload key %q, got %#v", key, payload)
		}
	}
	if payload["invite_id"] != "inv1" || payload["group_id"] != "g1" {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
	rejectedBy, ok := payload["rejected_by"].(types.UserInfo)
	if !ok || rejectedBy.UserID != "u1" {
		t.Fatalf("unexpected rejected_by in event: %#v", payload["rejected_by"])
	}
	if payload["reason"] != reason {
		t.Fatalf("expected reason %q in event payload, got %#v", reason, payload)
	}
}

func TestServiceRejectInviteOmitsReasonWhenEmpty(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &stubGroupsRepo{
		myInvites: []GroupInviteItem{{
			InviteID: "inv1",
			Status:   pending,
			Group:    GroupInfo{GroupID: "g1", Name: "test"},
			Inviter:  UserInfo{UserID: "u2"},
		}},
	}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	inviterSink := registerSink(hub, "u2")

	if err := svc.RejectInvite(grpCtx("u1"), RejectInviteRequest{InviteID: "inv1"}); err != nil {
		t.Fatalf("reject invite: %v", err)
	}
	if len(inviterSink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(inviterSink.events))
	}
	payload, ok := inviterSink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %#v", inviterSink.events[0].Payload)
	}
	if _, hasReason := payload["reason"]; hasReason {
		t.Fatalf("expected no reason in event payload, got %#v", payload)
	}
}

// --- 邀请惰性过期 ---

func TestServiceRejectInviteLazyExpiryRevokesAndPushesExpired(t *testing.T) {
	pending := ApplicationStatusPending
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		myInvites: []GroupInviteItem{{
			InviteID:  "inv1",
			Status:    pending,
			Group:     GroupInfo{GroupID: "g1", Name: "test"},
			Inviter:   UserInfo{UserID: "u2"},
			ExpiresAt: &past,
		}},
	}}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	inviteeSink := registerSink(hub, "u1")

	err := svc.RejectInvite(grpCtx("u1"), RejectInviteRequest{InviteID: "inv1"})
	if !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("expected ErrInviteExpired, got %v", err)
	}
	if len(repo.revokeInviteCalls) != 1 || repo.revokeInviteCalls[0].inviteID != "inv1" {
		t.Fatalf("expected lazy revoke call, got %#v", repo.revokeInviteCalls)
	}
	if len(inviteeSink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(inviteeSink.events))
	}
	event := inviteeSink.events[0]
	if event.Type != realtime.EventTypeGroupInviteExpired {
		t.Fatalf("expected GroupInviteExpired, got %q", event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %#v", event.Payload)
	}
	if payload["invite_id"] != "inv1" || payload["group_id"] != "g1" {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
	if payload["expired_at"] == nil || payload["expired_at"] == "" {
		t.Fatalf("expected expired_at, got %#v", payload)
	}
}

func TestInviteExpired(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)

	if !inviteExpired(GroupInviteItem{ExpiresAt: &past}) {
		t.Error("expected expired for past expires_at")
	}
	if inviteExpired(GroupInviteItem{ExpiresAt: &future}) {
		t.Error("expected not expired for future expires_at")
	}
	if inviteExpired(GroupInviteItem{ExpiresAt: nil}) {
		t.Error("expected not expired for nil expires_at")
	}
	bad := "not-a-time"
	if inviteExpired(GroupInviteItem{ExpiresAt: &bad}) {
		t.Error("expected not expired for malformed expires_at")
	}
}

// --- CancelApplication ---

func TestServiceCancelApplicationNotifiesAdmins(t *testing.T) {
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		getResult: GroupItem{Group: GroupInfo{GroupID: "g1", Name: "test"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}},
		getFound:  true,
		apps: []GroupApplicationItem{{
			ApplicationID: "app1",
			GroupID:       "g1",
			Applicant:     UserInfo{UserID: "u2"},
		}},
		members: []GroupMemberItem{
			{User: UserInfo{UserID: "u1"}, Role: GroupRoleOwner},
			{User: UserInfo{UserID: "u3"}, Role: GroupRoleAdmin},
			{User: UserInfo{UserID: "u4"}, Role: GroupRoleMember},
		},
	}}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	ownerSink := registerSink(hub, "u1")
	adminSink := registerSink(hub, "u3")
	memberSink := registerSink(hub, "u4")

	err := svc.CancelApplication(grpCtx("u2"), CancelApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if err != nil {
		t.Fatalf("cancel application: %v", err)
	}
	if len(repo.cancelApplicationIDs) != 1 || repo.cancelApplicationIDs[0] != "app1" {
		t.Fatalf("unexpected cancel calls: %#v", repo.cancelApplicationIDs)
	}
	// 事件只应送达管理员(owner/admin),普通成员不接收。
	if len(ownerSink.events) != 1 || ownerSink.events[0].Type != realtime.EventTypeGroupApplicationCancelled {
		t.Fatalf("unexpected owner events: %#v", ownerSink.events)
	}
	if len(adminSink.events) != 1 || adminSink.events[0].Type != realtime.EventTypeGroupApplicationCancelled {
		t.Fatalf("unexpected admin events: %#v", adminSink.events)
	}
	if len(memberSink.events) != 0 {
		t.Fatalf("expected no event for plain member, got %#v", memberSink.events)
	}
	payload, ok := ownerSink.events[0].Payload.(map[string]any)
	if !ok || payload["application_id"] != "app1" || payload["group_id"] != "g1" {
		t.Fatalf("unexpected owner payload: %#v", ownerSink.events[0].Payload)
	}
	if payload["cancelled_at"] == nil {
		t.Fatalf("unexpected owner payload fields: %#v", payload)
	}
	applicant, ok := payload["applicant"].(types.UserInfo)
	if !ok || applicant.UserID != "u2" {
		t.Fatalf("unexpected applicant in payload: %#v", payload["applicant"])
	}
}

func TestServiceCancelApplicationRejectsForeignApplication(t *testing.T) {
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		apps: []GroupApplicationItem{{ApplicationID: "app1", Applicant: UserInfo{UserID: "u9"}}},
	}}
	svc := NewServiceWithDeps(repo, &notifyWriterStub{}, nil)
	withTestHub(t)

	err := svc.CancelApplication(grpCtx("u2"), CancelApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if !errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
	if len(repo.cancelApplicationIDs) != 0 {
		t.Fatalf("expected no cancel call, got %#v", repo.cancelApplicationIDs)
	}
}

// --- RejectApplication ---

func TestServiceRejectApplicationNotifiesApplicant(t *testing.T) {
	pending := ApplicationStatusPending
	reason := "not suitable"
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		apps: []GroupApplicationItem{{
			ApplicationID: "app1",
			GroupID:       "g1",
			Status:        pending,
			Applicant:     UserInfo{UserID: "u3"},
		}},
	}}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	applicantSink := registerSink(hub, "u3")

	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1", Reason: &reason})
	if err != nil {
		t.Fatalf("reject application: %v", err)
	}
	if len(writer.applicationRejected) != 1 {
		t.Fatalf("expected 1 AppendGroupApplicationRejected call, got %d", len(writer.applicationRejected))
	}
	call := writer.applicationRejected[0]
	if call.targetUserID != "u3" || call.groupID != "g1" || call.applicationID != "app1" {
		t.Fatalf("unexpected rejected call: %#v", call)
	}
	// F7：审批者身份对称化——accepted/rejected 均不暴露审批者（spec api_spec_notify.md）。
	if call.rejectedBy.UserID != "" {
		t.Fatalf("expected empty rejected_by (F7), got %#v", call.rejectedBy)
	}
	if call.reason == nil || *call.reason != reason {
		t.Fatalf("expected reason %q, got %#v", reason, call.reason)
	}
	if len(applicantSink.events) != 1 {
		t.Fatalf("expected 1 event for applicant, got %d", len(applicantSink.events))
	}
	event := applicantSink.events[0]
	if event.Type != realtime.EventTypeGroupApplicationRejected {
		t.Fatalf("expected GroupApplicationRejected, got %q", event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %#v", event.Payload)
	}
	if payload["application_id"] != "app1" || payload["reason"] != reason {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
	// F7：事件负载不暴露审批者（accepted/rejected 对称）。
	if _, present := payload["rejected_by"]; present {
		t.Fatalf("did not expect rejected_by in payload (F7), got %#v", payload["rejected_by"])
	}
	group, ok := payload["group"].(map[string]any)
	if !ok || group["group_id"] != "g1" || group["name"] != "test" {
		t.Fatalf("unexpected group in payload: %#v", payload["group"])
	}
	if payload["notify_id"] != "notif_written" {
		t.Fatalf("expected notification_id embedded, got %#v", payload["notify_id"])
	}
}

// --- 申请惰性过期 ---

func TestServiceRejectApplicationLazyExpiryCancelsAndPushesExpired(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		apps: []GroupApplicationItem{{
			ApplicationID: "app1",
			GroupID:       "g1",
			Status:        pending,
			Applicant:     UserInfo{UserID: "u3"},
			CreatedAt:     "2020-01-01T00:00:00Z", // 远超默认 72h TTL
		}},
		members: []GroupMemberItem{
			{User: UserInfo{UserID: "u1"}, Role: GroupRoleOwner},
		},
	}}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	adminSink := registerSink(hub, "u1")

	err := svc.RejectApplication(grpCtx("u1"), RejectGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("expected ErrAlreadyProcessed, got %v", err)
	}
	if len(repo.cancelApplicationIDs) != 1 || repo.cancelApplicationIDs[0] != "app1" {
		t.Fatalf("expected lazy cancel call, got %#v", repo.cancelApplicationIDs)
	}
	if len(adminSink.events) != 1 || adminSink.events[0].Type != realtime.EventTypeGroupApplicationExpired {
		t.Fatalf("unexpected admin events: %#v", adminSink.events)
	}
	payload, ok := adminSink.events[0].Payload.(map[string]any)
	if !ok || payload["application_id"] != "app1" || payload["group_id"] != "g1" {
		t.Fatalf("unexpected expired payload: %#v", adminSink.events[0].Payload)
	}
	if payload["expired_at"] == nil {
		t.Fatalf("unexpected expired payload fields: %#v", payload)
	}
}

func TestServiceApproveApplicationLazyExpiryCancelsAndPushesExpired(t *testing.T) {
	pending := ApplicationStatusPending
	repo := &captureGroupsRepo{stubGroupsRepo: &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		apps: []GroupApplicationItem{{
			ApplicationID: "app1",
			GroupID:       "g1",
			Status:        pending,
			Applicant:     UserInfo{UserID: "u3"},
			CreatedAt:     "2020-01-01T00:00:00Z",
		}},
		members: []GroupMemberItem{
			{User: UserInfo{UserID: "u1"}, Role: GroupRoleOwner},
		},
	}}
	writer := &notifyWriterStub{}
	svc := NewServiceWithDeps(repo, writer, nil)
	hub := withTestHub(t)
	adminSink := registerSink(hub, "u1")

	err := svc.ApproveApplication(grpCtx("u1"), ApproveGroupApplicationRequest{GroupID: "g1", ApplicationID: "app1"})
	if !errors.Is(err, ErrAlreadyProcessed) {
		t.Fatalf("expected ErrAlreadyProcessed, got %v", err)
	}
	if len(repo.cancelApplicationIDs) != 1 || repo.cancelApplicationIDs[0] != "app1" {
		t.Fatalf("expected lazy cancel call, got %#v", repo.cancelApplicationIDs)
	}
	if len(adminSink.events) != 1 || adminSink.events[0].Type != realtime.EventTypeGroupApplicationExpired {
		t.Fatalf("unexpected admin events: %#v", adminSink.events)
	}
}

// --- applicationExpired ---

func TestApplicationExpired(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	past := time.Now().UTC().Add(-72 * time.Hour).Add(-time.Minute).Format(time.RFC3339)
	future := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)

	if !svc.applicationExpired(context.Background(), GroupApplicationItem{CreatedAt: past}) {
		t.Error("expected expired for application older than default 72h TTL")
	}
	if svc.applicationExpired(context.Background(), GroupApplicationItem{CreatedAt: future}) {
		t.Error("expected not expired for recent application")
	}
	if svc.applicationExpired(context.Background(), GroupApplicationItem{CreatedAt: "not-a-time"}) {
		t.Error("expected not expired for malformed created_at")
	}
}

func TestApplicationExpiredHonorsConfiguredTTL(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{})
	svc.SetAppConfig(appConfigStub{ttlHours: 1})
	overTTL := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	withinTTL := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)

	if !svc.applicationExpired(context.Background(), GroupApplicationItem{CreatedAt: overTTL}) {
		t.Error("expected expired with 1h TTL for 2h-old application")
	}
	if svc.applicationExpired(context.Background(), GroupApplicationItem{CreatedAt: withinTTL}) {
		t.Error("expected not expired with 1h TTL for 30m-old application")
	}

	// ttl <= 0 时回退默认 72h。
	svc.SetAppConfig(appConfigStub{ttlHours: 0})
	oneDayAgo := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	if svc.applicationExpired(context.Background(), GroupApplicationItem{CreatedAt: oneDayAgo}) {
		t.Error("expected not expired with default 72h TTL for 24h-old application")
	}
}

// --- resolveUserInfo ---

func TestResolveUserInfoUsesActorProvider(t *testing.T) {
	provider := actorProviderStub{info: actor.Info{UserID: "u42", Nickname: "Nick", AvatarURL: "https://example.com/avatar.png"}}
	svc := NewServiceWithDeps(&stubGroupsRepo{}, nil, provider)

	got := svc.resolveUserInfo(context.Background(), "u42")
	if got.UserID != "u42" || got.Nickname != "Nick" {
		t.Fatalf("unexpected user info: %#v", got)
	}
	if got.AvatarURL == nil || *got.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("expected avatar_url, got %#v", got)
	}
}

func TestResolveUserInfoFallsBackToUserID(t *testing.T) {
	svc := NewServiceWithRepo(&stubGroupsRepo{}) // users == nil
	got := svc.resolveUserInfo(context.Background(), "u42")
	if got.UserID != "u42" || got.Nickname != "u42" {
		t.Fatalf("expected fallback user_id/nickname, got %#v", got)
	}
	if got.AvatarURL != nil {
		t.Fatalf("expected no avatar_url in fallback, got %#v", got)
	}
}

func TestResolveUserInfoFallsBackOnProviderError(t *testing.T) {
	provider := actorProviderStub{err: errors.New("actor lookup failed")}
	svc := NewServiceWithDeps(&stubGroupsRepo{}, nil, provider)
	got := svc.resolveUserInfo(context.Background(), "u42")
	if got.UserID != "u42" || got.Nickname != "u42" {
		t.Fatalf("expected fallback on provider error, got %#v", got)
	}
}
