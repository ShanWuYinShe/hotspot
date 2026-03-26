package realtime

import "testing"

type sinkStub struct{ events []Event }

func (s *sinkStub) Send(e Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error       { return nil }

func TestHubBroadcasts(t *testing.T) {
	hub := NewHub()
	sink := &sinkStub{}
	hub.RegisterUser("user_alice", "sess_1", sink)
	hub.BroadcastToUser("user_alice", Event{Type: "PrivateMessageNew"})
	hub.BroadcastToUser("user_alice", Event{Type: "NewFollower"})
	if len(sink.events) != 2 {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
	groupSink := &sinkStub{}
	hub.SetGroupSubscriptions([]string{"group_1"}, groupSink)
	hub.BroadcastToGroup("group_1", Event{Type: "GroupMessageNew"})
	if len(groupSink.events) != 1 {
		t.Fatalf("unexpected group events: %#v", groupSink.events)
	}
	allUserSink := &sinkStub{}
	allSecondSink := &sinkStub{}
	hub.RegisterUser("user_alice", "sess_2", allUserSink)
	hub.RegisterUser("user_bob", "sess_3", allSecondSink)
	hub.BroadcastToAll(Event{Type: "SystemAnnouncement"})
	if len(allUserSink.events) != 1 || len(allSecondSink.events) != 1 {
		t.Fatalf("unexpected broadcast-all events: %#v %#v", allUserSink.events, allSecondSink.events)
	}
}
