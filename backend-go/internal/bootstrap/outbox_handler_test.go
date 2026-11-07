package bootstrap

import (
	"context"
	"encoding/json"
	"testing"

	"hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/notify"
	systemmodule "hotspot/backend-go/internal/modules/system"
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
