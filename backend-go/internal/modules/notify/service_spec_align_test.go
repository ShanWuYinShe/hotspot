package notify

// 本文件覆盖 spec 对齐新增/修改的 Append*/PublishNearbyNewMoment/SetNearbyNotifyProvider 逻辑以及通知类型白名单。
// 断言依据 spec/api_spec_notify.md 的 NotifyPayload 联合类型结构。

import (
	"context"
	"errors"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// nearbyNotifyProviderStub 模拟用户“附近新瞬间”通知开关。
type nearbyNotifyProviderStub struct {
	enabled bool
	err     error
}

func (p nearbyNotifyProviderStub) IsNearbyMomentNotifyEnabled(context.Context, string) (bool, error) {
	return p.enabled, p.err
}

// withTestHub 安装独立 hub 并在测试结束后恢复原默认 hub,
// 返回 hub 与空 sink(调用方按需 RegisterUser)。
func withTestHub(t *testing.T) *sinkStub {
	t.Helper()
	previous := realtime.DefaultHub()
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	t.Cleanup(func() { realtime.SetDefaultHub(previous) })
	return &sinkStub{}
}

func requireSingleAppend(t *testing.T, repo *notifyListCaptureRepo, wantUser string) NotifyItem {
	t.Helper()
	if repo.appendUser != wantUser || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: user=%q appended=%d", repo.appendUser, len(repo.appended))
	}
	return repo.appended[0]
}

// --- AppendGroupApplicationRejected ---

func TestAppendGroupApplicationRejectedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	reason := "not suitable"
	notifyID, err := service.AppendGroupApplicationRejected(context.Background(), "user_bob", "group_1", "Hot Group", "app_1", &reason)
	if err != nil {
		t.Fatalf("append group application rejected: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_bob")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "group_application_rejected" || item.Payload["type"] != "group_application_rejected" {
		t.Fatalf("unexpected type: %#v", item)
	}
	// F7: 与 accepted 对称，不暴露审批者身份——payload 必须不含 rejected_by。
	if _, exists := item.Payload["rejected_by"]; exists {
		t.Fatalf("rejected_by must not be exposed (F7): %#v", item.Payload["rejected_by"])
	}
	if item.Payload["reason"] != reason {
		t.Fatalf("expected reason=%q, got %#v", reason, item.Payload["reason"])
	}
}

// --- AppendGroupApplicationReceived ---

func TestAppendGroupApplicationReceivedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	id, err := service.AppendGroupApplicationReceived(context.Background(), "user_admin", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "group_1", "Hot Group", "app_1", nil)
	if err != nil {
		t.Fatalf("append group application received: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty notify_id")
	}
	item := requireSingleAppend(t, repo, "user_admin")
	if item.NotifyID != id {
		t.Fatalf("expected item.NotifyID=%q, got %q", id, item.NotifyID)
	}
	if item.Type != "group_application_received" || item.Payload["type"] != "group_application_received" {
		t.Fatalf("unexpected type: %#v", item)
	}
	if item.Payload["group_id"] != "group_1" || item.Payload["group_name"] != "Hot Group" || item.Payload["application_id"] != "app_1" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
	applicant, ok := item.Payload["applicant"].(types.UserInfo)
	if !ok || applicant.UserID != "user_bob" {
		t.Fatalf("unexpected applicant payload: %#v", item.Payload["applicant"])
	}
}

// --- AppendReportResolved / AppendReportDismissed ---

func TestAppendReportResolvedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendReportResolved(context.Background(), "user_reporter", "report_1", "moment", "moment_9", "违规内容已处理")
	if err != nil {
		t.Fatalf("append report resolved: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_reporter")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "report_resolved" || item.Payload["type"] != "report_resolved" {
		t.Fatalf("unexpected type: %#v", item)
	}
	if item.Payload["report_id"] != "report_1" || item.Payload["target_type"] != "moment" || item.Payload["target_id"] != "moment_9" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
	if item.Payload["note"] != "违规内容已处理" {
		t.Fatalf("expected note, got %#v", item.Payload["note"])
	}
}

func TestAppendReportResolvedOmitsNoteWhenEmpty(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendReportResolved(context.Background(), "user_reporter", "report_1", "comment", "comment_2", ""); err != nil {
		t.Fatalf("append report resolved: %v", err)
	}
	item := requireSingleAppend(t, repo, "user_reporter")
	if _, hasNote := item.Payload["note"]; hasNote {
		t.Fatalf("expected no note, got %#v", item.Payload)
	}
	if item.Payload["target_type"] != "comment" {
		t.Fatalf("unexpected target_type: %#v", item.Payload)
	}
}

func TestAppendReportDismissedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendReportDismissed(context.Background(), "user_reporter", "report_2", "user", "user_x", "证据不足")
	if err != nil {
		t.Fatalf("append report dismissed: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_reporter")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "report_dismissed" || item.Payload["type"] != "report_dismissed" {
		t.Fatalf("unexpected type: %#v", item)
	}
	if item.Payload["report_id"] != "report_2" || item.Payload["target_type"] != "user" || item.Payload["target_id"] != "user_x" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
	if item.Payload["note"] != "证据不足" {
		t.Fatalf("expected note, got %#v", item.Payload["note"])
	}
}

func TestAppendReportDismissedOmitsNoteWhenEmpty(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendReportDismissed(context.Background(), "user_reporter", "report_2", "moment", "moment_1", ""); err != nil {
		t.Fatalf("append report dismissed: %v", err)
	}
	item := requireSingleAppend(t, repo, "user_reporter")
	if _, hasNote := item.Payload["note"]; hasNote {
		t.Fatalf("expected no note, got %#v", item.Payload)
	}
}

// --- AppendCommentReply ---

func TestAppendCommentReplyBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendCommentReply(context.Background(), "user_alice",
		types.UserInfo{UserID: "user_bob", Nickname: "Bob"},
		"moment_1", types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"},
		"comment_1", "parent preview", "reply_1", "reply preview")
	if err != nil {
		t.Fatalf("append comment reply: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_alice")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "comment_reply" || item.Payload["type"] != "comment_reply" {
		t.Fatalf("unexpected type: %#v", item)
	}
	checks := map[string]string{
		"moment_id":               "moment_1",
		"comment_id":              "comment_1",
		"replied_comment_preview": "parent preview",
		"reply_id":                "reply_1",
		"reply_content_preview":   "reply preview",
	}
	for key, want := range checks {
		if item.Payload[key] != want {
			t.Fatalf("expected payload[%q]=%q, got %#v", key, want, item.Payload[key])
		}
	}
	if _, ok := item.Payload["moment_summary"].(types.MomentSummary); !ok {
		t.Fatalf("expected moment_summary struct, got %#v", item.Payload["moment_summary"])
	}
	if _, ok := item.Payload["actor"].(types.UserInfo); !ok {
		t.Fatalf("expected actor struct, got %#v", item.Payload["actor"])
	}
}

// --- AppendNewFollower / AppendMutualFollow / AppendMomentCommented / ---
// --- AppendMentioned / AppendSystemAnnouncement (notify_id) ---

// TestAppendNewFollowerBuildsNotifyID verifies the generated
// notify_id is returned and persisted as NotifyItem.NotifyID
// (embedded as the first field of the pushed NewFollower payload, spec
// api_spec_websocket.md).
func TestAppendNewFollowerBuildsNotifyID(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendNewFollower(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "moment", "moment_123")
	if err != nil {
		t.Fatalf("append new follower: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_alice")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "new_follower" || item.Payload["type"] != "new_follower" {
		t.Fatalf("unexpected type: %#v", item)
	}
}

func TestAppendMutualFollowBuildsNotifyID(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendMutualFollow(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"})
	if err != nil {
		t.Fatalf("append mutual follow: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_alice")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "mutual_follow" || item.Payload["type"] != "mutual_follow" {
		t.Fatalf("unexpected type: %#v", item)
	}
}

func TestAppendMomentCommentedBuildsNotifyID(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendMomentCommented(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "moment_1", types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"}, "comment_1", "preview")
	if err != nil {
		t.Fatalf("append moment commented: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_alice")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "moment_commented" || item.Payload["type"] != "moment_commented" {
		t.Fatalf("unexpected type: %#v", item)
	}
}

func TestAppendMentionedBuildsNotifyID(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendMentioned(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "comment", "moment_1", types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"}, "comment_1", "preview")
	if err != nil {
		t.Fatalf("append mentioned: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	item := requireSingleAppend(t, repo, "user_alice")
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "mentioned" || item.Payload["type"] != "mentioned" {
		t.Fatalf("unexpected type: %#v", item)
	}
}

func TestAppendSystemAnnouncementBuildsNotifyID(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnounceLevelInfo, CreatedAt: "2026-03-26T00:00:00Z"}
	notifyID, err := service.AppendSystemAnnouncement(context.Background(), []string{"user_alice", "user_bob"}, payload)
	if err != nil {
		t.Fatalf("append system announcement: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	if len(repo.appendMany) != 2 || len(repo.appended) != 1 {
		t.Fatalf("unexpected append-many capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "system_announcement" || item.Payload["type"] != "system_announcement" || item.Payload["announcement_id"] != "ann_1" {
		t.Fatalf("unexpected system notify item: %#v", item)
	}
}

// --- PublishNearbyNewMoment + SetNearbyNotifyProvider ---

func TestPublishNearbyNewMomentNoProviderAppendsAndPushes(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo) // provider 未设置 => 默认推送
	sink := withTestHub(t)
	hub := realtime.DefaultHub()
	hub.RegisterUser("user_nearby", "sess_1", sink)

	err := service.PublishNearbyNewMoment(context.Background(), "user_nearby", "user_actor", "moment_42", 350, types.UserInfo{UserID: "user_actor", Nickname: "Actor"}, types.MomentSummary{MomentID: "moment_42"})
	if err != nil {
		t.Fatalf("publish nearby new moment: %v", err)
	}
	item := requireSingleAppend(t, repo, "user_nearby")
	if item.Type != "nearby_new_moment" || item.Payload["type"] != "nearby_new_moment" {
		t.Fatalf("unexpected type: %#v", item)
	}
	if item.Payload["moment_id"] != "moment_42" || item.Payload["distance"] != 350 {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeNearbyNewMoment {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
	eventPayload, ok := sink.events[0].Payload.(map[string]any)
	if !ok || eventPayload["moment_id"] != "moment_42" || eventPayload["distance"] != 350 {
		t.Fatalf("unexpected event payload: %#v", sink.events[0].Payload)
	}
	// WS payload 携带与落库记录一致的 notify_id（spec api_spec_websocket.md，
	// 客户端去重 + 对齐 HTTP 侧通知）。
	if eventPayload["notify_id"] != item.NotifyID || item.NotifyID == "" {
		t.Fatalf("expected event notify_id to match persisted record, event=%v item=%#v", eventPayload["notify_id"], item)
	}
}

func TestPublishNearbyNewMomentEnabledProviderAppendsAndPushes(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	service.SetNearbyNotifyProvider(nearbyNotifyProviderStub{enabled: true})
	sink := withTestHub(t)
	realtime.DefaultHub().RegisterUser("user_nearby", "sess_1", sink)

	if err := service.PublishNearbyNewMoment(context.Background(), "user_nearby", "user_actor", "moment_42", 120, types.UserInfo{UserID: "user_actor"}, types.MomentSummary{MomentID: "moment_42"}); err != nil {
		t.Fatalf("publish nearby new moment: %v", err)
	}
	if len(repo.appended) != 1 {
		t.Fatalf("expected 1 append, got %d", len(repo.appended))
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeNearbyNewMoment {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
}

func TestPublishNearbyNewMomentDisabledProviderSkips(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	service.SetNearbyNotifyProvider(nearbyNotifyProviderStub{enabled: false})
	sink := withTestHub(t)
	realtime.DefaultHub().RegisterUser("user_nearby", "sess_1", sink)

	if err := service.PublishNearbyNewMoment(context.Background(), "user_nearby", "user_actor", "moment_42", 120, types.UserInfo{UserID: "user_actor"}, types.MomentSummary{MomentID: "moment_42"}); err != nil {
		t.Fatalf("publish nearby new moment: %v", err)
	}
	if len(repo.appended) != 0 {
		t.Fatalf("expected no append when disabled, got %d", len(repo.appended))
	}
	if len(sink.events) != 0 {
		t.Fatalf("expected no events when disabled, got %#v", sink.events)
	}
}

func TestPublishNearbyNewMomentProviderErrorPropagates(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	service.SetNearbyNotifyProvider(nearbyNotifyProviderStub{err: errors.New("settings unavailable")})

	err := service.PublishNearbyNewMoment(context.Background(), "user_nearby", "user_actor", "moment_42", 120, types.UserInfo{UserID: "user_actor"}, types.MomentSummary{MomentID: "moment_42"})
	if err == nil {
		t.Fatal("expected provider error, got nil")
	}
	if len(repo.appended) != 0 {
		t.Fatalf("expected no append on provider error, got %d", len(repo.appended))
	}
}

// --- 通知类型白名单(normalizeNotifyType) ---

func TestNormalizeNotifyTypeAcceptsAllSpecTypes(t *testing.T) {
	// 与 spec/api_spec_types.md NotifyType union 完全一致（15 种）。
	legal := []string{
		"new_follower",
		"mutual_follow",
		"moment_commented",
		"comment_reply",
		"mentioned",
		"group_mentioned",
		"reaction_added",
		"group_invite_received",
		"group_application_received",
		"group_application_accepted",
		"group_application_rejected",
		"report_resolved",
		"report_dismissed",
		"system_announcement",
		"nearby_new_moment",
	}
	if len(legal) != 15 {
		t.Fatalf("expected 15 legal types, got %d", len(legal))
	}
	for _, typ := range legal {
		if got := normalizeNotifyType(typ); got != typ {
			t.Fatalf("normalizeNotifyType(%q) = %q, want %q", typ, got, typ)
		}
	}
}

func TestNormalizeNotifyTypeTrimsAndHandlesEmpty(t *testing.T) {
	if got := normalizeNotifyType("  new_follower  "); got != "new_follower" {
		t.Fatalf("expected trimmed type, got %q", got)
	}
	if got := normalizeNotifyType(""); got != "" {
		t.Fatalf("expected empty for empty input, got %q", got)
	}
}

func TestNormalizeNotifyTypeRejectsIllegalTypes(t *testing.T) {
	for _, typ := range []string{"moment_liked", "comment_liked", "garbage", "new follower", "system", "report"} {
		if got := normalizeNotifyType(typ); got != "" {
			t.Fatalf("normalizeNotifyType(%q) = %q, want empty", typ, got)
		}
	}
}
