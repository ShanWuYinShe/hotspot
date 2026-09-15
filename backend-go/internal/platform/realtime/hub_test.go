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

// TestHubReverseIndexIsolatesSinks 验证反向索引语义（审计 perf P3-5）：
// 多 sink 各自登记订阅时，Set*Subscriptions 全量重建与 UnregisterUser 清理
// 只影响自己登记的 keys，其他 sink 的订阅与广播不受影响。
func TestHubReverseIndexIsolatesSinks(t *testing.T) {
	hub := NewHub()
	sinkA := &sinkStub{}
	sinkB := &sinkStub{}

	hub.SetUserSubscriptions([]string{"user_1", "user_2"}, sinkA)
	hub.SetGroupSubscriptions([]string{"group_1"}, sinkA)
	hub.SetUserSubscriptions([]string{"user_2", "user_3"}, sinkB)
	hub.SetGroupSubscriptions([]string{"group_1", "group_2"}, sinkB)

	// sinkA 全量重建 user 订阅：user_1 移除、user_4 新增，user_2 双方共存。
	hub.SetUserSubscriptions([]string{"user_2", "user_4"}, sinkA)

	// user_1 只被 sinkA 订阅过：重建后双方都不该收到。
	hub.BroadcastToUserSubscribers("user_1", Event{Type: "T"})
	if len(sinkA.events) != 0 || len(sinkB.events) != 0 {
		t.Fatalf("重建后 user_1 广播不应到达任何 sink, A=%#v B=%#v", sinkA.events, sinkB.events)
	}
	// user_2 由 A、B 同时订阅：广播双方都应收到（重建不误伤 B 的订阅）。
	hub.BroadcastToUserSubscribers("user_2", Event{Type: "T"})
	if len(sinkA.events) != 1 || len(sinkB.events) != 1 {
		t.Fatalf("user_2 广播应同时到达 A、B, A=%#v B=%#v", sinkA.events, sinkB.events)
	}

	// 注销 sinkB：其 user/group 订阅全部清理，sinkA 的不受影响。
	hub.UnregisterUser("user_b", "sess_b", sinkB)
	sinkA.events = nil
	sinkB.events = nil
	hub.BroadcastToUserSubscribers("user_2", Event{Type: "T"})
	hub.BroadcastToUserSubscribers("user_3", Event{Type: "T"})
	hub.BroadcastToGroup("group_1", Event{Type: "T"})
	hub.BroadcastToGroup("group_2", Event{Type: "T"})
	if len(sinkB.events) != 0 {
		t.Fatalf("注销后 sinkB 不应收到任何广播, got %#v", sinkB.events)
	}
	// sinkA：user_2 (user 订阅) + group_1 (group 订阅) 各一条；user_3/group_2 不订阅。
	if len(sinkA.events) != 2 {
		t.Fatalf("sinkA 应恰好收到 user_2 与 group_1 两条, got %#v", sinkA.events)
	}
}

// revocableSink 模拟 socketSink 的回调接线：撤销回调把连接侧订阅快照更新后
// 全量重建（对齐 handler 的 onSubscriptionRevoked + applyAuthorizedSubscriptions
// 重建路径），用于验证「撤销后重建不复活」的闭环。
type revocableSink struct {
	sinkStub
	onRevoke func(kind, id string)
}

func (s *revocableSink) revokeSubscription(kind, id string) {
	if s.onRevoke != nil {
		s.onRevoke(kind, id)
	}
}

// TestHubRevokeGroupSubscription 验证成员资格变更后的订阅强制摘除（安全修复）：
// 被移出成员保持连接也不再收到群消息，其他群订阅与其他成员不受影响；
// 连接侧快照经回调同步后，下一次全量重建不会把已撤销订阅加回 byGroup。
func TestHubRevokeGroupSubscription(t *testing.T) {
	hub := NewHub()
	sink := &revocableSink{}
	hub.RegisterUser("user_alice", "sess_1", sink)
	// 连接侧订阅快照（handler 的 subs 状态）+ 初始注册。
	snapshot := map[string]bool{"group_1": true, "group_2": true}
	hub.SetGroupSubscriptions([]string{"group_1", "group_2"}, sink)
	// 模拟 handler：撤销回调同步快照并全量重建。
	sink.onRevoke = func(kind, id string) {
		delete(snapshot, id)
		ids := make([]string, 0, len(snapshot))
		for gid := range snapshot {
			ids = append(ids, gid)
		}
		hub.SetGroupSubscriptions(ids, sink)
	}
	otherSink := &sinkStub{}
	hub.RegisterUser("user_bob", "sess_2", otherSink)
	hub.SetGroupSubscriptions([]string{"group_1"}, otherSink)

	hub.RevokeGroupSubscription("user_alice", "group_1")

	sink.events = nil
	otherSink.events = nil
	hub.BroadcastToGroup("group_1", Event{Type: "GroupMessageNew"})
	// group_2 订阅仍在（只摘除目标群）。
	hub.BroadcastToGroup("group_2", Event{Type: "GroupMessageNew"})
	if len(sink.events) != 1 {
		t.Fatalf("撤销后 alice 只应收到 group_2 一条, got %#v", sink.events)
	}
	if len(otherSink.events) != 1 {
		t.Fatalf("其他成员 bob 的 group_1 订阅不应受影响, got %#v", otherSink.events)
	}

	// 模拟客户端下一次 SetSubscriptions：按连接侧快照全量重建（已不含 group_1），
	// 群消息仍不会送达 alice——验证快照同步闭环。
	hub.BroadcastToGroup("group_1", Event{Type: "GroupMessageNew"})
	if len(sink.events) != 1 {
		t.Fatalf("后续广播不得再送达被移出成员, got %#v", sink.events)
	}
}

// TestHubRevokeGroupSubscriptionWithoutCallback 验证未实现回调的 sink（测试桩）
// 也至少被摘除 byGroup 投递——注册表清理不依赖回调存在。
func TestHubRevokeGroupSubscriptionWithoutCallback(t *testing.T) {
	hub := NewHub()
	sink := &sinkStub{}
	hub.RegisterUser("user_alice", "sess_1", sink)
	hub.SetGroupSubscriptions([]string{"group_1"}, sink)
	hub.RevokeGroupSubscription("user_alice", "group_1")
	hub.BroadcastToGroup("group_1", Event{Type: "GroupMessageNew"})
	if len(sink.events) != 0 {
		t.Fatalf("撤销后不应再收到群广播, got %#v", sink.events)
	}
}

// TestHubDropGroup 验证群解散后全部订阅者摘除，其他群的订阅不受影响。
func TestHubDropGroup(t *testing.T) {
	hub := NewHub()
	sinkA := &sinkStub{}
	sinkB := &sinkStub{}
	hub.RegisterUser("user_a", "sess_a", sinkA)
	hub.RegisterUser("user_b", "sess_b", sinkB)
	hub.SetGroupSubscriptions([]string{"group_1", "group_2"}, sinkA)
	hub.SetGroupSubscriptions([]string{"group_1"}, sinkB)

	hub.DropGroup("group_1")

	sinkA.events = nil
	sinkB.events = nil
	hub.BroadcastToGroup("group_1", Event{Type: "T"})
	hub.BroadcastToGroup("group_2", Event{Type: "T"})
	if len(sinkA.events) != 1 || len(sinkB.events) != 0 {
		t.Fatalf("解散后 group_1 全员摘除、group_2 保留, A=%#v B=%#v", sinkA.events, sinkB.events)
	}
}

// TestBridgeDispatchRevokeChannels 验证跨实例控制通道：revoke:/groupdrop: 只清
// 注册表，不向 sink 投递事件（对端实例不应产生客户端可见输出）。
func TestBridgeDispatchRevokeChannels(t *testing.T) {
	hub := NewHub()
	sink := &revocableSink{}
	hub.RegisterUser("user_alice", "sess_1", sink)
	hub.SetGroupSubscriptions([]string{"group_1", "group_2"}, sink)
	revoked := map[string]string{}
	sink.onRevoke = func(kind, id string) { revoked[kind+"/"+id] = id }

	bridge := &RedisBridge{hub: hub}
	bridge.dispatchLocal("revoke:user_alice", Event{Type: EventTypeSubscriptionRevoked, Payload: map[string]any{"group_id": "group_1"}})
	bridge.dispatchLocal("groupdrop:group_2", Event{Type: EventTypeSubscriptionRevoked, Payload: map[string]any{"group_id": "group_2"}})

	if _, ok := revoked["group/group_1"]; !ok {
		t.Fatalf("revoke: 通道应触发回调, got %#v", revoked)
	}
	if _, ok := revoked["group/group_2"]; !ok {
		t.Fatalf("groupdrop: 通道应触发回调, got %#v", revoked)
	}
	// 全部订阅已清：后续广播不可达。
	hub.BroadcastToGroup("group_1", Event{Type: "T"})
	hub.BroadcastToGroup("group_2", Event{Type: "T"})
	if len(sink.events) != 0 {
		t.Fatalf("控制通道处理后不应有广播可达, got %#v", sink.events)
	}
}
