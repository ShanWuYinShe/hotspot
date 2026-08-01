package admin

import (
	"context"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/realtime"
)

// capturePushStub 记录 AppendReportResolved/AppendReportDismissed 的调用参数,
// 用于断言举报处理后的落库通知(与 stubPushService 的布尔型不同,保留参数)。
type capturePushStub struct {
	resolvedCalls []reportNotifyCall
	dismissCalls  []reportNotifyCall
}

type reportNotifyCall struct {
	reporterUserID string
	reportID       string
	targetType     string
	targetID       string
	note           string
}

func (s *capturePushStub) AppendSystemAnnouncement(_ context.Context, _ []string, _ systemmodule.SystemAnnouncementItem) (string, error) {
	return "notif_ann_1", nil
}

func (s *capturePushStub) PushSystemAnnouncement(_ systemmodule.SystemAnnouncementItem, _ string) {}

func (s *capturePushStub) AppendReportResolved(_ context.Context, reporterUserID, reportID, targetType, targetID, note string) (string, error) {
	s.resolvedCalls = append(s.resolvedCalls, reportNotifyCall{reporterUserID, reportID, targetType, targetID, note})
	return "notif_resolved_1", nil
}

func (s *capturePushStub) AppendReportDismissed(_ context.Context, reporterUserID, reportID, targetType, targetID, note string) (string, error) {
	s.dismissCalls = append(s.dismissCalls, reportNotifyCall{reporterUserID, reportID, targetType, targetID, note})
	return "notif_dismissed_1", nil
}

// adminSinkStub 捕获 BroadcastToUser 投递,用于 hub 注入断言。
type adminSinkStub struct{ events []realtime.Event }

func (s *adminSinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *adminSinkStub) Close() error                { return nil }

// seedPendingReport 向内存仓库注入一条 pending 举报。
func seedPendingReport(t *testing.T, store *InMemoryRepository, reportID, reporterID, targetType, targetID string) {
	t.Helper()
	store.reports[reportID] = reportHandleRecord{
		Item: AdminReportItem{
			ReportID:   reportID,
			TargetType: ReportTargetType(targetType),
			TargetID:   targetID,
			Reason:     "spam",
			Status:     ReportStatusPending,
			CreatedAt:  "2026-04-14T00:00:00Z",
			Reporter:   UserInfo{UserID: reporterID, Nickname: "Reporter"},
		},
		TargetUser: UserInfo{UserID: targetID, Nickname: "Target"},
	}
}

func newHandleReportService(t *testing.T) (*Service, *InMemoryRepository, *capturePushStub) {
	t.Helper()
	repo := NewInMemoryRepository()
	provider := &stubProvider{config: defaultConfig()}
	push := &capturePushStub{}
	mods := NewInMemoryModerationStore()
	svc := NewService(repo, provider, mods, push)
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	return svc, repo, push
}

func TestHandleReportResolvePushesAndNotifiesReporter(t *testing.T) {
	svc, repo, push := newHandleReportService(t)
	seedPendingReport(t, repo, "report_1", "user_reporter", "user", "user_target")

	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()
	reporterSink := &adminSinkStub{}
	hub.RegisterUser("user_reporter", "sess_r", reporterSink)

	apiErr, err := svc.HandleReport(adminCtx("admin1"), AdminReportHandleRequest{
		ReportID: "report_1",
		Action:   ReportActionResolve,
		Note:     "confirmed violation",
	})
	if err != nil || apiErr != nil {
		t.Fatalf("handle report: apiErr=%v err=%v", apiErr, err)
	}

	// 落库通知:举报人收到 report_resolved,参数正确
	if len(push.resolvedCalls) != 1 {
		t.Fatalf("expected 1 resolved notify call, got %d", len(push.resolvedCalls))
	}
	call := push.resolvedCalls[0]
	if call.reporterUserID != "user_reporter" || call.reportID != "report_1" || call.targetType != "user" || call.targetID != "user_target" || call.note != "confirmed violation" {
		t.Fatalf("unexpected resolved notify call: %+v", call)
	}
	if len(push.dismissCalls) != 0 {
		t.Fatalf("unexpected dismiss notify call: %+v", push.dismissCalls)
	}

	// WS 事件:ReportResolved 送达举报人,payload 含 notification_id/report_id/note/resolved_at
	if len(reporterSink.events) != 1 {
		t.Fatalf("expected 1 WS event to reporter, got %d: %+v", len(reporterSink.events), reporterSink.events)
	}
	ev := reporterSink.events[0]
	if ev.Type != realtime.EventTypeReportResolved {
		t.Fatalf("expected ReportResolved event, got %s", ev.Type)
	}
	payload, ok := ev.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", ev.Payload)
	}
	if payload["notification_id"] != "notif_resolved_1" || payload["report_id"] != "report_1" || payload["target_type"] != "user" || payload["target_id"] != "user_target" || payload["note"] != "confirmed violation" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if _, ok := payload["resolved_at"]; !ok {
		t.Fatalf("missing resolved_at: %+v", payload)
	}
}

func TestHandleReportDismissPushesAndNotifiesReporter(t *testing.T) {
	svc, repo, push := newHandleReportService(t)
	seedPendingReport(t, repo, "report_2", "user_reporter", "moment", "moment_9")

	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()
	reporterSink := &adminSinkStub{}
	hub.RegisterUser("user_reporter", "sess_r", reporterSink)

	apiErr, err := svc.HandleReport(adminCtx("admin1"), AdminReportHandleRequest{
		ReportID: "report_2",
		Action:   ReportActionDismiss,
	})
	if err != nil || apiErr != nil {
		t.Fatalf("handle report: apiErr=%v err=%v", apiErr, err)
	}

	if len(push.dismissCalls) != 1 {
		t.Fatalf("expected 1 dismiss notify call, got %d", len(push.dismissCalls))
	}
	call := push.dismissCalls[0]
	if call.reporterUserID != "user_reporter" || call.reportID != "report_2" || call.targetType != "moment" || call.targetID != "moment_9" || call.note != "" {
		t.Fatalf("unexpected dismiss notify call: %+v", call)
	}

	// note 为空时 payload 不包含 note 键(契约:可选字段省略)
	if len(reporterSink.events) != 1 {
		t.Fatalf("expected 1 WS event, got %d", len(reporterSink.events))
	}
	ev := reporterSink.events[0]
	if ev.Type != realtime.EventTypeReportDismissed {
		t.Fatalf("expected ReportDismissed event, got %s", ev.Type)
	}
	payload := ev.Payload.(map[string]any)
	if payload["notification_id"] != "notif_dismissed_1" || payload["report_id"] != "report_2" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if _, ok := payload["note"]; ok {
		t.Fatalf("note should be omitted when empty: %+v", payload)
	}
	if _, ok := payload["dismissed_at"]; !ok {
		t.Fatalf("missing dismissed_at: %+v", payload)
	}
}

func TestHandleReportAlreadyTerminalDoesNotNotify(t *testing.T) {
	svc, repo, push := newHandleReportService(t)
	seedPendingReport(t, repo, "report_3", "user_reporter", "comment", "comment_1")
	// 先处理一次进入终态
	if apiErr, err := svc.HandleReport(adminCtx("admin1"), AdminReportHandleRequest{ReportID: "report_3", Action: ReportActionDismiss}); err != nil || apiErr != nil {
		t.Fatalf("first handle: apiErr=%v err=%v", apiErr, err)
	}
	// 重复处理:10400 且不再产生通知/推送
	apiErr, err := svc.HandleReport(adminCtx("admin1"), AdminReportHandleRequest{ReportID: "report_3", Action: ReportActionDismiss})
	if err != nil {
		t.Fatalf("second handle error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10400 {
		t.Fatalf("expected 10400 on terminal report, got %+v", apiErr)
	}
	if len(push.dismissCalls) != 1 {
		t.Fatalf("expected only 1 dismiss notify call, got %d", len(push.dismissCalls))
	}
}
