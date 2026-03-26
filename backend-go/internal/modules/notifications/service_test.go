package notifications

import (
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/realtime"
)

type sinkStub struct{ events []realtime.Event }

func (s *sinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error                { return nil }

func TestPushSystemAnnouncementFansOut(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	aliceSink := &sinkStub{}
	bobSink := &sinkStub{}
	hub.RegisterUser("user_alice", "sess_1", aliceSink)
	hub.RegisterUser("user_bob", "sess_2", bobSink)
	service := NewService()
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnouncementSeverityInfo, CreatedAt: "2026-03-26T00:00:00Z"}
	service.PushSystemAnnouncement(payload)
	if len(aliceSink.events) != 1 || len(bobSink.events) != 1 {
		t.Fatalf("unexpected events: %#v %#v", aliceSink.events, bobSink.events)
	}
	if aliceSink.events[0].Type != "SystemAnnouncement" || bobSink.events[0].Type != "SystemAnnouncement" {
		t.Fatalf("unexpected event types: %#v %#v", aliceSink.events, bobSink.events)
	}
	alicePayload, ok := aliceSink.events[0].Payload.(systemmodule.SystemAnnouncementItem)
	if !ok || alicePayload.AnnouncementID != "ann_1" || alicePayload.Title != "维护通知" {
		t.Fatalf("unexpected alice payload: %#v", aliceSink.events[0].Payload)
	}
}
