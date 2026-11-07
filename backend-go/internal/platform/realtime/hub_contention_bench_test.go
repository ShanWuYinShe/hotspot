package realtime

import (
	"sync/atomic"
	"testing"
)

// BenchmarkHubBroadcastParallelDisjoint 测并发广播到不同 user 的伸缩性：
// 旧实现所有读路径共享同一把 RWMutex，广播之间虽可并发，但与注册/注销写锁
// 互相排斥；分片后不同 key 落到不同 shard，读写互不阻塞。
func BenchmarkHubBroadcastParallelDisjoint(b *testing.B) {
	hub := NewHub()
	const users = 64
	for i := 0; i < users; i++ {
		hub.RegisterUser(userKey(i), "sess_"+string(rune('a'+i%26)), &benchPreparedSink{})
	}
	event := benchEvent()
	var next atomic.Uint64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := next.Add(1) % users
			hub.BroadcastToUser(userKey(int(idx)), event)
		}
	})
}

// BenchmarkHubMixedBroadcastChurn 混合压测：一半操作广播到已存在用户，一半
// 注册/注销临时连接。单锁实现里连接抖动会阻塞全部广播；分片后广播只与同分片
// 的写竞争，伸缩性显著改善。
func BenchmarkHubMixedBroadcastChurn(b *testing.B) {
	hub := NewHub()
	const stableUsers = 128
	for i := 0; i < stableUsers; i++ {
		hub.RegisterUser(userKey(i), "stable_sess_"+userKey(i), &benchPreparedSink{})
	}
	event := benchEvent()
	var counter atomic.Uint64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := counter.Add(1)
			if n%2 == 0 {
				hub.BroadcastToUser(userKey(int(n%stableUsers)), event)
				continue
			}
			userID := "churn_user_" + userKey(int(n%512))
			sessionID := "churn_sess_" + userKey(int(n))
			sink := &benchPreparedSink{}
			hub.RegisterUser(userID, sessionID, sink)
			hub.UnregisterUser(userID, sessionID, sink)
		}
	})
}

// BenchmarkHubBroadcastToAll 全量广播（系统公告等）：每条事件 O(在线连接数)，
// 分片实现需要从各 shard 汇总 sink 快照；用于确认分片没有把全量广播改慢。
func BenchmarkHubBroadcastToAll(b *testing.B) {
	hub := NewHub()
	const connections = 512
	for i := 0; i < connections; i++ {
		hub.RegisterUser("all_user_"+userKey(i), "all_sess_"+userKey(i), &benchPreparedSink{})
	}
	event := benchEvent()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastToAll(event)
	}
}

func userKey(i int) string {
	const digits = "0123456789abcdef"
	if i < 16 {
		return string(digits[i])
	}
	return userKey(i/16) + string(digits[i%16])
}
