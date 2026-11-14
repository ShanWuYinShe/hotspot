package realtime

import (
	"sync"
	"testing"
	"time"
)

// evictRecordingSink 记录 Close 次数与收到的广播，用于验证单用户连接上限
// 触发的逐出路径确实关闭了连接（审计 A 平台层 P1-1：旧实现只从注册表摘除不 Close，
// 连接成僵尸——读循环永不退出、fd/goroutine 泄漏，客户端也收不到断线信号）。
type evictRecordingSink struct {
	mu      sync.Mutex
	events  []Event
	closes  int
	onClose func()
}

func (s *evictRecordingSink) Send(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *evictRecordingSink) Close() error {
	s.mu.Lock()
	s.closes++
	onClose := s.onClose
	s.mu.Unlock()
	if onClose != nil {
		onClose()
	}
	return nil
}

func (s *evictRecordingSink) closeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closes
}

func (s *evictRecordingSink) eventCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

// TestHubRegisterUserEvictsOldestConnection 验证第 maxConnectionsPerUser+1 条
// 连接注册时：最旧连接被 Close 恰好一次且从注册表/反向索引摘除，其余连接与
// 新连接不受影响；被逐出连接不再收到广播。
func TestHubRegisterUserEvictsOldestConnection(t *testing.T) {
	hub := NewHub()
	sinks := make([]*evictRecordingSink, maxConnectionsPerUser)
	for i := range sinks {
		sinks[i] = &evictRecordingSink{}
		hub.RegisterUser("user_evict", "sess_"+itoa(i), sinks[i])
	}
	// 未达上限：任何连接都不应被关闭。
	for i, s := range sinks {
		if got := s.closeCount(); got != 0 {
			t.Fatalf("未达上限时第 %d 条连接不应被关闭, got %d", i+1, got)
		}
	}

	extra := &evictRecordingSink{}
	hub.RegisterUser("user_evict", "sess_extra", extra)

	if got := sinks[0].closeCount(); got != 1 {
		t.Fatalf("注册第 %d 条连接应逐出并 Close 最旧连接恰好一次, got %d", maxConnectionsPerUser+1, got)
	}
	for i := 1; i < len(sinks); i++ {
		if got := sinks[i].closeCount(); got != 0 {
			t.Fatalf("第 %d 条连接不应被关闭, got %d", i+1, got)
		}
	}
	if got := extra.closeCount(); got != 0 {
		t.Fatalf("新连接不应被关闭, got %d", got)
	}

	// 注册表/反向索引不再含被逐出连接，总量维持在单用户上限。
	if _, ok := hub.sessionBySink.Load(sinks[0]); ok {
		t.Fatal("被逐出连接应从 sessionBySink 移除")
	}
	if got := len(hub.snapshotSessions()); got != maxConnectionsPerUser {
		t.Fatalf("在线 sink 数 = %d, want %d", got, maxConnectionsPerUser)
	}
	if got := hub.sessionCount.Load(); got != maxConnectionsPerUser {
		t.Fatalf("sessionCount = %d, want %d", got, maxConnectionsPerUser)
	}

	// 被逐出连接不得再收到广播，其余连接与新连接照常收到。
	hub.BroadcastToUser("user_evict", Event{Type: "T"})
	if got := sinks[0].eventCount(); got != 0 {
		t.Fatalf("被逐出连接不应再收到广播, got %d", got)
	}
	for i := 1; i < len(sinks); i++ {
		if got := sinks[i].eventCount(); got != 1 {
			t.Fatalf("第 %d 条连接应收到广播, got %d", i+1, got)
		}
	}
	if got := extra.eventCount(); got != 1 {
		t.Fatalf("新连接应收到广播, got %d", got)
	}
}

// TestHubRegisterUserClosesEvictedSinkOutsideShardLock 锁定「关闭被逐出连接必须
// 在全部 shard 锁之外」的约束：socketSink.Close 会做网络 Close 并最多等待写
// 超时（5s），持锁关闭会阻塞同 shard 的注册与广播。这里让被逐出连接的 Close
// 反向遍历 Hub 注册表（需要取同一 shard 的读锁）；若实现持锁关闭，该调用会
// 永久阻塞，由超时兜底失败。
func TestHubRegisterUserClosesEvictedSinkOutsideShardLock(t *testing.T) {
	hub := NewHub()
	sinks := make([]*evictRecordingSink, maxConnectionsPerUser)
	for i := range sinks {
		sinks[i] = &evictRecordingSink{}
		hub.RegisterUser("user_lock", "sess_"+itoa(i), sinks[i])
	}
	sinks[0].onClose = func() { _ = len(hub.snapshotSessions()) }

	done := make(chan struct{})
	go func() {
		defer close(done)
		hub.RegisterUser("user_lock", "sess_extra", &evictRecordingSink{})
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RegisterUser 在持有 shard 锁时关闭了被逐出连接（Close 内的读锁被阻塞）")
	}
	if got := sinks[0].closeCount(); got != 1 {
		t.Fatalf("被逐出连接应被 Close 恰好一次, got %d", got)
	}
}
