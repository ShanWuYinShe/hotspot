package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/modules/notify"
	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
)

type outboxSinkCapture struct {
	events []realtime.Event
}

func (s *outboxSinkCapture) Send(event realtime.Event) error {
	s.events = append(s.events, event)
	return nil
}
func (s *outboxSinkCapture) Close() error { return nil }

func TestSystemAnnouncementOutboxHandlerBroadcasts(t *testing.T) {
	hub := realtime.NewHub()
	sink := &outboxSinkCapture{}
	hub.RegisterUser("user_alice", "sess_1", sink)
	payload, err := json.Marshal(admin.AnnouncementBroadcastPayload{
		Item: systemmodule.SystemAnnouncementItem{
			AnnouncementID: "ann_1",
			Title:          "维护通知",
			Content:        "今晚维护",
			Severity:       systemmodule.AnnounceLevelWarning,
			CreatedAt:      "2026-11-07T10:00:00Z",
		},
		NotifyID: systemmodule.AnnouncementNotifyID("ann_1"),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	handler := systemAnnouncementOutboxHandler(hub)
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeSystemAnnouncement {
		t.Fatalf("unexpected broadcast: %#v", sink.events)
	}
	payloadOut, ok := sink.events[0].Payload.(notify.SystemAnnouncementPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", sink.events[0].Payload)
	}
	if payloadOut.NotifyID != systemmodule.AnnouncementNotifyID("ann_1") || payloadOut.Announcement.Title != "维护通知" || payloadOut.Announcement.Severity != string(systemmodule.AnnounceLevelWarning) {
		t.Fatalf("unexpected announcement payload: %#v", payloadOut)
	}
	if err := handler(context.Background(), []byte("{")); err == nil {
		t.Fatal("invalid payload must return error")
	}
}

// mediaServiceFromModules 从模块切片定位 media 模块（dispatcher 注册注销清理
// handler 的装配前提）：生产装配必须能找到；找不到时注册侧跳过并 Error 暴露。
func TestMediaServiceFromModulesLocatesMediaModule(t *testing.T) {
	cfg := config.Config{
		S3Endpoint:       "http://seaweedfs:8333",
		S3Bucket:         "hotspot",
		S3Region:         "us-east-1",
		S3AccessKey:      "minioadmin",
		S3SecretKey:      "minioadmin123",
		S3ForcePathStyle: true,
	}
	if svc := mediaServiceFromModules(BuildModules(cfg, persistence.NewFactory(cfg))); svc == nil {
		t.Fatal("BuildModules must contain the media module (account purge outbox handler depends on it)")
	}
	if svc := mediaServiceFromModules(nil); svc != nil {
		t.Fatalf("empty module slice must yield nil, got %v", svc)
	}
}

// mediaPurgeDeleterStub 记录注销对象清理 handler 的转发参数（bootstrap 只经
// mediaPurgeObjectDeleter 接口消费 media.Service）。
type mediaPurgeDeleterStub struct {
	calls  int
	userID string
	keys   []string
	err    error
}

func (s *mediaPurgeDeleterStub) DeletePurgeObjects(_ context.Context, userID string, keys []string) error {
	s.calls++
	s.userID = userID
	s.keys = keys
	return s.err
}

func TestMediaPurgeOutboxHandlerDeletesObjects(t *testing.T) {
	deleter := &mediaPurgeDeleterStub{}
	handler := mediaPurgeOutboxHandler(deleter)
	keys := []string{"published/user_1/a.jpg", "uploads/user_1/b.png"}
	payload, err := json.Marshal(media.PurgeObjectsPayload{UserID: "user_1", Keys: keys})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if deleter.calls != 1 || deleter.userID != "user_1" || !slices.Equal(deleter.keys, keys) {
		t.Fatalf("unexpected delete call: calls=%d user=%q keys=%v", deleter.calls, deleter.userID, deleter.keys)
	}

	// 空 keys 直接成功（幂等，不调用删除能力）。
	empty, err := json.Marshal(media.PurgeObjectsPayload{UserID: "user_1"})
	if err != nil {
		t.Fatalf("marshal empty payload: %v", err)
	}
	if err := handler(context.Background(), empty); err != nil {
		t.Fatalf("empty keys must succeed: %v", err)
	}
	if deleter.calls != 1 {
		t.Fatalf("empty keys must not call deleter, calls = %d", deleter.calls)
	}

	// 非法 JSON / 缺 user_id 返回错误：由 dispatcher 按重试/死信语义处理。
	if err := handler(context.Background(), []byte("{")); err == nil {
		t.Fatal("invalid payload must return error")
	}
	missingUser, err := json.Marshal(media.PurgeObjectsPayload{Keys: keys})
	if err != nil {
		t.Fatalf("marshal missing-user payload: %v", err)
	}
	if err := handler(context.Background(), missingUser); err == nil {
		t.Fatal("missing user_id must return error")
	}

	// 删除失败原样上抛，由 outbox 重试（已删对象 404 幂等）。
	boom := errors.New("s3 delete boom")
	failing := &mediaPurgeDeleterStub{err: boom}
	if err := mediaPurgeOutboxHandler(failing)(context.Background(), payload); !errors.Is(err, boom) {
		t.Fatalf("deleter error must propagate for retry, got %v", err)
	}
}

func TestAdminReportHandledOutboxHandlerBroadcasts(t *testing.T) {
	hub := realtime.NewHub()
	sink := &outboxSinkCapture{}
	hub.RegisterUser("user_reporter", "sess_report", sink)
	payload, err := json.Marshal(admin.ReportHandledBroadcastPayload{
		NotifyID:       "notif_report_1",
		ReporterUserID: "user_reporter",
		ReportID:       "report_1",
		TargetType:     "moment",
		TargetID:       "moment_9",
		Note:           "违规内容已处理",
		Resolved:       true,
		OccurredAt:     "2026-11-07T14:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	handler := adminReportHandledOutboxHandler(hub)
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeReportResolved {
		t.Fatalf("unexpected broadcast: %#v", sink.events)
	}
	wsPayload, ok := sink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type %T", sink.events[0].Payload)
	}
	if wsPayload["notify_id"] != "notif_report_1" || wsPayload["report_id"] != "report_1" || wsPayload["note"] != "违规内容已处理" || wsPayload["resolved_at"] != "2026-11-07T14:00:00Z" {
		t.Fatalf("unexpected ws payload: %#v", wsPayload)
	}
	if err := handler(context.Background(), []byte("{")); err == nil {
		t.Fatal("invalid payload must return error")
	}
}
