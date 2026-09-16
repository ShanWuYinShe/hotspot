package realtime

import (
	"context"
	"testing"
)

// 组合路径（审计 perf P1-4）：store 实现 recordPublisher 时，广播走单次往返
// 组合方法——publisher 不再被调用（发布已随脚本完成），本地 sink 收到带 seq 事件。
type combinedStoreStub struct {
	seq      int64
	fail     bool
	called   string
	channels []string
}

func (s *combinedStoreStub) Record(string, Event) int64      { return 0 }
func (s *combinedStoreStub) RecordGroup(string, Event) int64 { return 0 }
func (s *combinedStoreStub) Replay(string, []string, int64) []Event {
	return nil
}
func (s *combinedStoreStub) LastSeq() int64                { return 0 }
func (s *combinedStoreStub) SaveState(string, ResumeState) {}
func (s *combinedStoreStub) LoadState(string) (ResumeState, bool) {
	return ResumeState{}, false
}
func (s *combinedStoreStub) Close() error { return nil }

func (s *combinedStoreStub) RecordUserAndPublish(userID, channel string, event Event) (int64, bool) {
	s.called = "user:" + userID
	s.channels = append(s.channels, channel)
	if s.fail {
		return 0, false
	}
	s.seq++
	return s.seq, true
}

func (s *combinedStoreStub) RecordGroupAndPublish(groupID, channel string, event Event) (int64, bool) {
	s.called = "group:" + groupID
	s.channels = append(s.channels, channel)
	if s.fail {
		return 0, false
	}
	s.seq++
	return s.seq, true
}

type countingPublisher struct {
	calls    int
	channels []string
	err      error
}

func (p *countingPublisher) Publish(_ context.Context, channel string, _ Event) error {
	p.calls++
	p.channels = append(p.channels, channel)
	return p.err
}

func (p *countingPublisher) Close() error { return nil }

func TestBroadcastToUserCombinedSkipsPublish(t *testing.T) {
	store := &combinedStoreStub{}
	hub := NewHub()
	pub := &countingPublisher{}
	hub.SetResumeStore(store)
	hub.SetPublisher(pub)
	sink := &sinkStub{}
	hub.RegisterUser("u1", "s1", sink)

	hub.BroadcastToUser("u1", Event{Type: "test"})

	if store.called != "user:u1" {
		t.Fatalf("combined store not invoked: %q", store.called)
	}
	if pub.calls != 0 {
		t.Fatalf("publish must be skipped when combined path published, got %d calls", pub.calls)
	}
	if len(sink.events) != 1 || sink.events[0].ServerSeq != 1 {
		t.Fatalf("sink should receive event with seq, got %#v", sink.events)
	}
}

func TestBroadcastToGroupCombinedSkipsPublish(t *testing.T) {
	store := &combinedStoreStub{}
	hub := NewHub()
	pub := &countingPublisher{}
	hub.SetResumeStore(store)
	hub.SetPublisher(pub)
	sink := &sinkStub{}
	hub.SetGroupSubscriptions([]string{"g1"}, sink)

	hub.BroadcastToGroup("g1", Event{Type: "test"})

	if store.called != "group:g1" {
		t.Fatalf("combined store not invoked: %q", store.called)
	}
	if pub.calls != 0 {
		t.Fatalf("publish must be skipped, got %d calls", pub.calls)
	}
	if len(sink.events) != 1 || sink.events[0].ServerSeq != 1 {
		t.Fatalf("sink should receive event with seq, got %#v", sink.events)
	}
}

// 脚本失败（published=false, seq=0）：回退独立 publish，本地投递照常（降级语义
// 与既有 record 失败一致——事件无 seq，客户端全量重放兜底）。
func TestBroadcastCombinedFailureFallsBackToPublish(t *testing.T) {
	store := &combinedStoreStub{fail: true}
	hub := NewHub()
	pub := &countingPublisher{}
	hub.SetResumeStore(store)
	hub.SetPublisher(pub)
	sink := &sinkStub{}
	hub.RegisterUser("u1", "s1", sink)

	hub.BroadcastToUser("u1", Event{Type: "test"})

	if pub.calls != 1 || len(pub.channels) != 1 || pub.channels[0] != "user:u1" {
		t.Fatalf("fallback publish expected once on user:u1, got calls=%d channels=%v", pub.calls, pub.channels)
	}
	if len(sink.events) != 1 || sink.events[0].ServerSeq != 0 {
		t.Fatalf("degraded delivery should carry no seq, got %#v", sink.events)
	}
	if len(store.channels) != 1 || store.channels[0] != "user:u1" {
		t.Fatalf("combined store should still be attempted with right channel: %v", store.channels)
	}
}

// store 不实现 recordPublisher（如 NopResumeStore / 测试桩）：走既有两步路径。
func TestBroadcastLegacyPathWhenCombinedUnsupported(t *testing.T) {
	store := newFakeResumeStore()
	hub := NewHub()
	pub := &countingPublisher{}
	hub.SetResumeStore(store)
	hub.SetPublisher(pub)
	sink := &sinkStub{}
	hub.RegisterUser("u1", "s1", sink)

	hub.BroadcastToUser("u1", Event{Type: "test"})

	if pub.calls != 1 {
		t.Fatalf("legacy path must publish, got %d calls", pub.calls)
	}
	if len(sink.events) != 1 || sink.events[0].ServerSeq != 1 {
		t.Fatalf("legacy path should still enrich seq, got %#v", sink.events)
	}
	// 编译期防误加：fakeResumeStore 不应意外实现组合接口（否则本测试覆盖失真）
	var _ ResumeStore = store
	if _, ok := interface{}(store).(recordPublisher); ok {
		t.Fatal("fakeResumeStore should not accidentally implement recordPublisher")
	}
}
