package admin

import (
	"context"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/outbox"
)

type captureOutboxWriter struct {
	topic   string
	payload any
	calls   int
}

func (w *captureOutboxWriter) Enqueue(_ context.Context, topic string, payload any) error {
	w.calls++
	w.topic = topic
	w.payload = payload
	return nil
}

// 公告创建 + WS 广播 outbox 入队必须在同一事务内执行；注入 outbox 后不再
// 走同步广播（避免事务外副作用）。
func TestCreateAnnouncementEnqueuesOutboxInTransaction(t *testing.T) {
	announcements := &recordingAnnouncementStore{inner: NewInMemoryModerationStore()}
	store := NewPersistenceModerationStore(nil, announcements, nil)
	push := &stubPushService{}
	svc := NewService(NewInMemoryRepository(), &stubProvider{config: defaultConfig()}, store, push)
	txCalls := 0
	svc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
		txCalls++
		return fn(ctx)
	})
	writer := &captureOutboxWriter{}
	svc.SetAnnouncementOutbox(writer)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "admin", SessionID: "sess_admin"})

	created, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "维护通知", Content: "今晚维护", Severity: AnnounceLevelInfo, SendPush: true})
	if err != nil || apiErr != nil {
		t.Fatalf("CreateAnnouncement err=%v apiErr=%+v", err, apiErr)
	}
	if txCalls != 1 {
		t.Fatalf("tx calls = %d, want 1", txCalls)
	}
	if writer.calls != 1 || writer.topic != outbox.TopicSystemAnnouncement {
		t.Fatalf("outbox enqueue = calls:%d topic:%q", writer.calls, writer.topic)
	}
	payload, ok := writer.payload.(AnnouncementBroadcastPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", writer.payload)
	}
	if payload.Item.AnnouncementID != created.AnnouncementID || payload.NotifyID != systemmodule.AnnouncementNotifyID(created.AnnouncementID) {
		t.Fatalf("unexpected outbox payload: %#v", payload)
	}
	if push.pushed {
		t.Fatalf("outbox path must not broadcast synchronously")
	}
}
