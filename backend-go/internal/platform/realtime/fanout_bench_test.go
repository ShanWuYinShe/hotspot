package realtime

import (
	"encoding/json"
	"testing"

	"github.com/gorilla/websocket"
)

// benchPlainSink 只实现基础 FanoutSink（逐连接 json.Marshal 回退路径）。
type benchPlainSink struct{ sent int }

func (s *benchPlainSink) Send(Event) error { s.sent++; return nil }
func (s *benchPlainSink) Close() error     { return nil }

// benchPreparedSink 模拟 socketSink 的复用路径（SendPrepared）。
type benchPreparedSink struct{ sent int }

func (s *benchPreparedSink) Send(Event) error { s.sent++; return nil }
func (s *benchPreparedSink) SendPrepared(*websocket.PreparedMessage, int64) error {
	s.sent++
	return nil
}
func (s *benchPreparedSink) Close() error { return nil }

func benchEvent() Event {
	return Event{Type: EventTypeGroupMessageNew, Payload: map[string]any{
		"message_id": "msg_bench_0001",
		"group_id":   "group_bench_01",
		"sender":     map[string]any{"user_id": "user_bench_a", "nickname": "压测用户"},
		"type":       "text",
		"content":    "这是一条用于压测广播序列化开销的群消息内容",
		"sent_at":    "2026-01-01T00:00:00.000000Z",
	}}
}

// BenchmarkBroadcastSinksPrepared 测「一次 marshal + PreparedMessage 复用」
// 向 100 个连接广播的单事件成本（socketSink 生产路径）。
func BenchmarkBroadcastSinksPrepared(b *testing.B) {
	sinks := make([]FanoutSink, 100)
	for i := range sinks {
		sinks[i] = &benchPreparedSink{}
	}
	event := benchEvent()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcastSinks(sinks, event)
	}
}

// BenchmarkBroadcastSinksPlain 测回退路径（每个 sink 自行 Marshal）：
// 与 Prepared 对比可量化「每连接重复序列化」的代价，防止回退。
func BenchmarkBroadcastSinksPlain(b *testing.B) {
	sinks := make([]FanoutSink, 100)
	for i := range sinks {
		sinks[i] = &benchPlainSink{}
	}
	event := benchEvent()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 回退路径的真实成本：每个 sink 各自序列化同一事件。
		for _, sink := range sinks {
			data, _ := json.Marshal(event)
			_ = sink.Send(event)
			_ = data
		}
	}
}

// BenchmarkEventMarshal 单事件 JSON 序列化大小基线（广播 payload 体积回归）。
func BenchmarkEventMarshal(b *testing.B) {
	event := benchEvent()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(event); err != nil {
			b.Fatal(err)
		}
	}
}
