package privatemessages

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/realtime"
)

// TestRecallMessageByIDAsAdminBroadcasts：举报裁决按 message_id 撤回私聊消息，
// 向双方广播 PrivateMessageRecalled；已撤回/不存在返回 (false, nil)。
func TestRecallMessageByIDAsAdminBroadcasts(t *testing.T) {
	hub := realtime.NewHub()
	previousHub := realtime.DefaultHub()
	realtime.SetDefaultHub(hub)
	t.Cleanup(func() { realtime.SetDefaultHub(previousHub) })

	repo := NewInMemoryRepository()
	service := NewServiceWithDeps(repo, nil)
	ctx := context.Background()
	senderSink := &eventSink{}
	receiverSink := &eventSink{}
	hub.RegisterUser("user_a", "sess_a", senderSink)
	hub.RegisterUser("user_b", "sess_b", receiverSink)

	sent, err := repo.Send(ctx, "user_a", "user_b", SendPrivateMessageRequest{TargetUserID: "user_b", Type: "text", Content: "bad"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	hit, err := service.RecallMessageByIDAsAdmin(ctx, "admin_1", sent.MessageID)
	if err != nil || !hit {
		t.Fatalf("recall as admin = %v err=%v, want hit", hit, err)
	}
	for _, sink := range []*eventSink{senderSink, receiverSink} {
		if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypePrivateMessageRecalled {
			t.Fatalf("expected one recall event per participant, got %#v", sink.events)
		}
	}
	hit, err = service.RecallMessageByIDAsAdmin(ctx, "admin_1", sent.MessageID)
	if err != nil || hit {
		t.Fatalf("repeat recall = %v err=%v, want false/nil (idempotent)", hit, err)
	}
	hit, err = service.RecallMessageByIDAsAdmin(ctx, "admin_1", "message_missing")
	if err != nil || hit {
		t.Fatalf("missing message = %v err=%v, want false/nil", hit, err)
	}
}
