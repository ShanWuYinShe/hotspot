package realtime

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type sinkStub struct{ events []Event }

func (s *sinkStub) Send(e Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error       { return nil }

type blockingSinkStub struct {
	mu      sync.Mutex
	started chan struct{}
	release chan struct{}
	events  []Event
}

func (s *blockingSinkStub) Send(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started != nil {
		select {
		case s.started <- struct{}{}:
		default:
		}
	}
	if s.release != nil {
		<-s.release
	}
	s.events = append(s.events, e)
	return nil
}

func (s *blockingSinkStub) Close() error { return nil }

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

func TestHubBroadcastToUserSubscribersAllowsConcurrentSendsOnSameSink(t *testing.T) {
	hub := NewHub()
	sink := &blockingSinkStub{started: make(chan struct{}, 2), release: make(chan struct{}, 2)}
	hub.SetUserSubscriptions([]string{"user_alice"}, sink)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		hub.BroadcastToUserSubscribers("user_alice", Event{Type: "UserOnline"})
	}()
	go func() {
		defer wg.Done()
		hub.BroadcastToUserSubscribers("user_alice", Event{Type: "PrivateTyping"})
	}()

	<-sink.started
	sink.release <- struct{}{}
	<-sink.started
	sink.release <- struct{}{}
	wg.Wait()

	if len(sink.events) != 2 {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
}

// countingSink records how many Send calls it received.
type countingSink struct {
	mu     sync.Mutex
	count  int
	events []Event
}

func (s *countingSink) Send(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
	s.events = append(s.events, e)
	return nil
}
func (s *countingSink) Close() error { return nil }

// failingPublisher is an EventPublisher whose Publish always returns an error,
// used to verify the Hub logs (rather than panics/swallows) cross-instance
// fan-out failures.
type failingPublisher struct{ calls int }

func (f *failingPublisher) Publish(context.Context, string, Event) error {
	f.calls++
	return errors.New("redis down")
}
func (f *failingPublisher) Close() error { return nil }

// TestHubPublish_LogsOnFailure verifies that when the configured publisher
// returns an error, the Hub does not panic and still invokes Publish (so the
// failure is observable via the warn log it emits).
func TestHubPublish_LogsOnFailure(t *testing.T) {
	hub := NewHub()
	pub := &failingPublisher{}
	hub.SetPublisher(pub)

	// BroadcastToUser triggers publish() for cross-instance fan-out. The local
	// subscriber still receives the event; the publish failure must not panic.
	sink := &sinkStub{}
	hub.RegisterUser("u1", "s1", sink)
	hub.BroadcastToUser("u1", Event{Type: "test"})

	if pub.calls == 0 {
		t.Fatal("expected Publish to be called for cross-instance fan-out")
	}
	if len(sink.events) != 1 {
		t.Fatalf("local delivery should still succeed, got %d events", len(sink.events))
	}
}

// TestHubDeliverAllLocalDropsTailBeyondCap verifies the drop-tail broadcast
// cap: when more than maxConnectionsPerBroadcast sinks are registered, only
// the first cap-many receive the event and the rest are silently dropped.
// This guards against fan-out storms swamping the goroutine pool.
func TestHubDeliverAllLocalDropsTailBeyondCap(t *testing.T) {
	hub := NewHub()
	// Register exactly cap+5 distinct sessions so total sinks exceed the cap.
	total := maxConnectionsPerBroadcast + 5
	sinks := make([]*countingSink, total)
	for i := 0; i < total; i++ {
		sinks[i] = &countingSink{}
		// Each session gets its own sink so we can count per-sink deliveries.
		hub.RegisterUser("user_"+itoa(i), "sess_"+itoa(i), sinks[i])
	}
	hub.deliverAllLocal(Event{Type: "SystemAnnouncement"})

	delivered := 0
	dropped := 0
	for _, s := range sinks {
		s.mu.Lock()
		if s.count > 0 {
			delivered++
		} else {
			dropped++
		}
		s.mu.Unlock()
	}
	if delivered != maxConnectionsPerBroadcast {
		t.Fatalf("expected exactly %d delivered, got %d", maxConnectionsPerBroadcast, delivered)
	}
	if dropped != 5 {
		t.Fatalf("expected 5 dropped tail sessions, got %d", dropped)
	}
}

// itoa is a tiny strconv.Itoa replacement so this test file doesn't pull
// in strconv just for index labeling.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [12]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
