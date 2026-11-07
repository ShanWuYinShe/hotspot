package admin

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/outbox"
	"hotspot/backend-go/internal/platform/realtime"
)

// 举报裁决路径接入 outbox 后：通知行在裁决事务内落库（append 同步执行），
// WS 广播事件与裁决同事务入队，提交前不发推送；未装配 outbox 时保持原
// aftercommit 同步语义（既有测试覆盖）。
func TestHandleReportEnqueuesOutboxForReporterNotification(t *testing.T) {
	svc, repo, push := newHandleReportService(t)
	seedPendingReport(t, repo, "report_outbox_1", "user_reporter", "user", "user_target")
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()
	reporterSink := &adminSinkStub{}
	hub.RegisterUser("user_reporter", "sess_r", reporterSink)

	writer := &captureOutboxWriter{}
	svc.SetEventOutbox(writer)
	txCalls := 0
	svc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
		txCalls++
		return fn(ctx)
	})

	apiErr, err := svc.HandleReport(adminCtx("admin1"), JudgeAdminReportRequest{
		ReportID: "report_outbox_1",
		Action:   ReportActionResolve,
		Note:     "confirmed",
	})
	if err != nil || apiErr != nil {
		t.Fatalf("handle report: apiErr=%v err=%v", apiErr, err)
	}
	if txCalls != 1 {
		t.Fatalf("tx calls = %d, want 1", txCalls)
	}
	if len(push.resolvedCalls) != 1 {
		t.Fatalf("notification append must run inside decision tx, calls=%d", len(push.resolvedCalls))
	}
	if writer.calls != 1 || writer.topic != outbox.TopicAdminReportHandled {
		t.Fatalf("outbox enqueue = calls:%d topic:%q", writer.calls, writer.topic)
	}
	payload, ok := writer.payload.(ReportHandledBroadcastPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", writer.payload)
	}
	if payload.NotifyID != "notif_resolved_1" || payload.ReporterUserID != "user_reporter" || payload.ReportID != "report_outbox_1" || payload.TargetType != "user" || payload.TargetID != "user_target" || payload.Note != "confirmed" || !payload.Resolved || payload.OccurredAt == "" {
		t.Fatalf("unexpected outbox payload: %#v", payload)
	}
	if len(reporterSink.events) != 0 {
		t.Fatalf("outbox path must not broadcast synchronously: %+v", reporterSink.events)
	}
}
