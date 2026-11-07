package realtime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRedisBridgeMultiInstanceBroadcastBurst 双实例（miniredis）广播压测：
// 8 个用户各 100 条并发跨实例事件，全部恰好投递一次。用于守住：
//   - 分片 Hub 快照 + RedisBridge receiveLoop 在并发发布下不丢/不重复；
//   - 跨实例用户事件路由按 user channel 正确落到目标实例连接。
func TestRedisBridgeMultiInstanceBroadcastBurst(t *testing.T) {
	pair := newBridgePair(t)
	pair.hubA.SetPublisher(pair.bridgeA)
	pair.hubB.SetPublisher(pair.bridgeB)
	const users = 8
	const perUser = 1000
	sinks := make([]*closingSink, users)
	for i := 0; i < users; i++ {
		sinks[i] = &closingSink{}
		pair.hubB.RegisterUser(fmt.Sprintf("burst_user_%d", i), fmt.Sprintf("burst_sess_%d", i), sinks[i])
	}
	waitSubscriptionReady(t, pair, "burst_user_0", sinks[0])

	var published atomic.Int64
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < users; i++ {
		userID := fmt.Sprintf("burst_user_%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < perUser; n++ {
				pair.hubA.BroadcastToUser(userID, Event{Type: EventTypeNewFollower, Payload: map[string]any{"n": n}})
				published.Add(1)
			}
		}()
	}
	wg.Wait()
	waitFor(t, 10*time.Second, "all burst events delivered", func() bool {
		total := 0
		for _, sink := range sinks {
			total += sink.count()
		}
		return total == users*perUser
	})
	elapsed := time.Since(start)
	// 等待一个短窗口确认没有额外重复投递。
	time.Sleep(100 * time.Millisecond)
	for i, sink := range sinks {
		if got := sink.count(); got != perUser {
			t.Fatalf("user %d received %d events, want %d", i, got, perUser)
		}
	}
	t.Logf("cross-instance burst: %d events in %s (%.0f events/s)", published.Load(), elapsed, float64(published.Load())/elapsed.Seconds())
}
