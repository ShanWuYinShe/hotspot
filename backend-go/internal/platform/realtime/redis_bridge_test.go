package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// RedisBridge 是多实例部署的跨实例 fanout 生产路径，此前只有信封解码测试。
// 本文件用 miniredis 起两条独立连接模拟两个实例，锁定：
//   - 各频道（user/usersub/group/all/session）跨实例投递；
//   - 本实例 origin 跳过（不得二次投递）；
//   - revoke/groupdrop 控制通道在远端摘除订阅；
//   - 非法信封/未知频道不打断接收循环；
//   - Close 幂等且关闭后 Publish 变为 no-op。
type closingSink struct {
	mu     sync.Mutex
	events []Event
	closed bool
}

func (s *closingSink) Send(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}
func (s *closingSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}
func (s *closingSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}
func (s *closingSink) snapshot() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}
func (s *closingSink) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
}

type bridgePair struct {
	rdbA, rdbB       *redis.Client
	bridgeA, bridgeB *RedisBridge
	hubA, hubB       *Hub
}

func newBridgePair(t *testing.T) *bridgePair {
	t.Helper()
	mr := miniredis.RunT(t)
	rdbA := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rdbB := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rdbA.Close()
		_ = rdbB.Close()
	})
	hubA, hubB := NewHub(), NewHub()
	bridgeA, err := NewRedisBridge(rdbA, "instance-a", hubA)
	if err != nil {
		t.Fatalf("bridge A: %v", err)
	}
	bridgeB, err := NewRedisBridge(rdbB, "instance-b", hubB)
	if err != nil {
		t.Fatalf("bridge B: %v", err)
	}
	t.Cleanup(func() {
		_ = bridgeA.Close()
		_ = bridgeB.Close()
	})
	return &bridgePair{rdbA: rdbA, rdbB: rdbB, bridgeA: bridgeA, bridgeB: bridgeB, hubA: hubA, hubB: hubB}
}

// waitFor 轮询断言，避免依赖 Pub/Sub 订阅建立的具体延迟。
func waitFor(t *testing.T, timeout time.Duration, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", desc)
}

// waitSubscriptionReady 通过重复发布心跳事件直到目标实例收到，确认 PSubscribe
// 已生效（建立后不会再退化）。
func waitSubscriptionReady(t *testing.T, pair *bridgePair, userID string, sink *closingSink) {
	t.Helper()
	waitFor(t, 3*time.Second, "cross-instance subscription ready", func() bool {
		_ = pair.bridgeA.Publish(context.Background(), redisChannelUser+userID, Event{Type: EventTypeNewFollower})
		return sink.count() > 0
	})
	// barrier 是循环发布的，停止发布后等管道内剩余消息全部投递完，再清零计数，
	// 否则后续「恰好一次」断言会把 barrier 的尾巴算进去。
	time.Sleep(200 * time.Millisecond)
	sink.reset()
}

func TestRedisBridgeDeliversAllChannelsCrossInstance(t *testing.T) {
	pair := newBridgePair(t)
	hog := &closingSink{}
	pair.hubB.RegisterUser("u1", "sess1", hog)
	pair.hubB.SetUserSubscriptions([]string{"watched"}, hog)
	pair.hubB.SetGroupSubscriptions([]string{"g1"}, hog)
	waitSubscriptionReady(t, pair, "u1", hog)

	// 本实例 sink：bridgeA 发布的消息 origin=instance-a，不得回到 hubA。
	local := &closingSink{}
	pair.hubA.RegisterUser("u1", "sess-local", local)

	publishAndExpect := func(channel string, event Event) Event {
		t.Helper()
		before := hog.count()
		if err := pair.bridgeA.Publish(context.Background(), channel, event); err != nil {
			t.Fatalf("publish %s: %v", channel, err)
		}
		waitFor(t, 2*time.Second, "delivery on "+channel, func() bool { return hog.count() > before })
		time.Sleep(50 * time.Millisecond) // 同频道重复投递窗口
		if got := hog.count() - before; got != 1 {
			t.Fatalf("%s delivered %d times, want 1", channel, got)
		}
		return hog.snapshot()[hog.count()-1]
	}

	if got := publishAndExpect(redisChannelUser+"u1", Event{Type: EventTypeNewFollower, Payload: map[string]any{"n": 1}}); got.Type != EventTypeNewFollower {
		t.Fatalf("user channel event = %#v", got)
	}
	if got := publishAndExpect(redisChannelUserSub+"watched", Event{Type: EventTypeUserOnline}); got.Type != EventTypeUserOnline {
		t.Fatalf("usersub channel event = %#v", got)
	}
	if got := publishAndExpect(redisChannelGroup+"g1", Event{Type: EventTypeGroupMessageNew}); got.Type != EventTypeGroupMessageNew {
		t.Fatalf("group channel event = %#v", got)
	}
	if got := publishAndExpect(redisChannelAll, Event{Type: EventTypeSystemAnnouncement}); got.Type != EventTypeSystemAnnouncement {
		t.Fatalf("all channel event = %#v", got)
	}

	// session 频道：向指定会话下发并关闭连接。
	sessionSink := &closingSink{}
	pair.hubB.RegisterUser("u2", "sess2", sessionSink)
	if err := pair.bridgeA.Publish(context.Background(), redisChannelSession+"sess2", Event{Type: EventTypeSessionRevoked}); err != nil {
		t.Fatalf("publish session: %v", err)
	}
	waitFor(t, 2*time.Second, "session close delivery", func() bool { return sessionSink.count() == 1 })
	sessionSink.mu.Lock()
	closed := sessionSink.closed
	sessionSink.mu.Unlock()
	if !closed {
		t.Fatal("session channel must close the sink")
	}

	// origin 跳过：本实例 sink 不应收到 bridgeA 自己发布的事件。
	if local.count() != 0 {
		t.Fatalf("origin instance received its own events: %d", local.count())
	}
}

func TestRedisBridgeRevokeAndGroupDropLocal(t *testing.T) {
	pair := newBridgePair(t)
	sink := &closingSink{}
	pair.hubB.RegisterUser("u1", "sess1", sink)
	pair.hubB.SetGroupSubscriptions([]string{"g1"}, sink)
	waitSubscriptionReady(t, pair, "u1", sink)

	// 正向对照组：撤销前群消息可投递。
	pair.hubB.BroadcastToGroup("g1", Event{Type: EventTypeGroupMessageNew})
	if sink.count() != 1 {
		t.Fatalf("precondition: group delivery count = %d", sink.count())
	}
	sink.reset()

	// 跨实例 revoke：payload 经 JSON 往返为 map，远端据 group_id 摘除订阅。
	if err := pair.bridgeA.Publish(context.Background(), redisChannelRevoke+"u1",
		Event{Type: EventTypeSubscriptionRevoked, Payload: map[string]any{"group_id": "g1"}}); err != nil {
		t.Fatalf("publish revoke: %v", err)
	}
	// 可观测判据：继续向 g1 广播，直到某次广播不再送达（订阅已摘除）。
	waitFor(t, 2*time.Second, "revoke applied", func() bool {
		before := sink.count()
		pair.hubB.BroadcastToGroup("g1", Event{Type: EventTypeGroupMessageNew})
		time.Sleep(10 * time.Millisecond)
		return sink.count() == before
	})
	sink.reset()
	pair.hubB.BroadcastToGroup("g1", Event{Type: EventTypeGroupMessageNew})
	time.Sleep(100 * time.Millisecond)
	if sink.count() != 0 {
		t.Fatalf("revoked subscription still receives group events: %d", sink.count())
	}

	// groupdrop：整群解散后所有订阅者摘除。
	sink2 := &closingSink{}
	pair.hubB.SetGroupSubscriptions([]string{"g2"}, sink2)
	pair.hubB.BroadcastToGroup("g2", Event{Type: EventTypeGroupMessageNew})
	if sink2.count() != 1 {
		t.Fatalf("precondition: g2 delivery count = %d", sink2.count())
	}
	sink2.reset()
	if err := pair.bridgeA.Publish(context.Background(), redisChannelGroupDrop+"g2", Event{Type: EventTypeSubscriptionRevoked}); err != nil {
		t.Fatalf("publish groupdrop: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	pair.hubB.BroadcastToGroup("g2", Event{Type: EventTypeGroupMessageNew})
	time.Sleep(50 * time.Millisecond)
	if sink2.count() != 0 {
		t.Fatalf("dropped group still receives events: %d", sink2.count())
	}
}

func TestRedisBridgeIgnoresMalformedAndUnknownMessages(t *testing.T) {
	pair := newBridgePair(t)
	sink := &closingSink{}
	pair.hubB.RegisterUser("u1", "sess1", sink)
	waitSubscriptionReady(t, pair, "u1", sink)

	ctx := context.Background()
	// 非法 JSON、缺 origin 的合法 JSON、未知频道：均不得打断接收循环。
	if err := pair.rdbA.Publish(ctx, redisChannelPrefix+"user:u1", "not-json").Err(); err != nil {
		t.Fatalf("publish malformed: %v", err)
	}
	if err := pair.rdbA.Publish(ctx, redisChannelPrefix+"nonsense", `{"o":"other","e":{"Type":"X"}}`).Err(); err != nil {
		t.Fatalf("publish unknown channel: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	// 后续合法事件仍能投递（循环存活）。
	before := sink.count()
	if err := pair.bridgeA.Publish(ctx, redisChannelUser+"u1", Event{Type: EventTypeNewFollower}); err != nil {
		t.Fatalf("publish valid: %v", err)
	}
	waitFor(t, 2*time.Second, "valid event after malformed", func() bool { return sink.count() > before })
}

func TestRedisBridgeCloseIsIdempotentAndStopsPublishing(t *testing.T) {
	pair := newBridgePair(t)
	if err := pair.bridgeA.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := pair.bridgeA.Close(); err != nil {
		t.Fatalf("second close must be no-op: %v", err)
	}
	if err := pair.bridgeA.Publish(context.Background(), redisChannelUser+"u1", Event{Type: EventTypeNewFollower}); err != nil {
		t.Fatalf("publish after close must be nil, got %v", err)
	}
	// 关闭后的 bridgeA 不再消费消息；从「第三个实例」原始发布，bridgeB 仍应投递。
	sink := &closingSink{}
	pair.hubB.RegisterUser("u1", "sess1", sink)
	payload, err := json.Marshal(redisEnvelope{Origin: "instance-c", Event: Event{Type: EventTypeNewFollower}})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if err := pair.rdbB.Publish(context.Background(), redisChannelPrefix+redisChannelUser+"u1", payload).Err(); err != nil {
		t.Fatalf("publish from third instance: %v", err)
	}
	waitFor(t, 2*time.Second, "bridgeB still delivers after bridgeA closed", func() bool { return sink.count() > 0 })
}
