package notifications

import (
	"testing"

	"hotspot/backend-go/internal/platform/realtime"
)

type sinkStub struct{ events []realtime.Event }

func (s *sinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error                { return nil }

func TestPushSystemAnnouncementFansOut(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	sink := &sinkStub{}
	hub.RegisterUser("user_alice", "sess_1", sink)
	service := NewService()
	service.PushSystemAnnouncement("user_alice", "ann_1", "info")
	if len(sink.events) != 1 || sink.events[0].Type != "SystemAnnouncement" {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
}
