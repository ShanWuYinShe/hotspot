package realtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestSocketSinkDeliversQueuedEvents sink 经队列投递：Send 只入队不阻塞，
// writer goroutine 按序写出，客户端收到完整 JSON。
func TestSocketSinkDeliversQueuedEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		sink := newSocketSink(conn)
		defer sink.Close()
		for i := 0; i < 5; i++ {
			if err := sink.Send(Event{Type: "ping"}); err != nil {
				return
			}
		}
		// 等客户端读完再关连接，避免测试结束竞态。
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	for i := 0; i < 5; i++ {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		var ev Event
		if err := json.Unmarshal(data, &ev); err != nil || ev.Type != "ping" {
			t.Fatalf("message %d: unexpected payload %s err=%v", i, data, err)
		}
	}
}

// TestSocketSinkSendNeverBlocksOnStalledConsumer 核心回归：消费端停读时
// Send 仍立即返回（满队丢弃），广播发起方永不被慢连接卡住。
func TestSocketSinkSendNeverBlocksOnStalledConsumer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		sink := newSocketSink(conn)
		defer sink.Close()
		done := make(chan time.Duration, 1)
		go func() {
			start := time.Now()
			// 64KB payload：总量远超内核发送缓冲 + 128 深队列，必然触发满队丢弃；
			// 若 Send 同步直写，第一次写满缓冲就会卡住犯规。
			big := strings.Repeat("x", 1<<16)
			for i := 0; i < sendQueueSize+50; i++ {
				_ = sink.Send(Event{Type: "spam", Payload: big})
			}
			done <- time.Since(start)
		}()
		select {
		case elapsed := <-done:
			if elapsed > 5*time.Second {
				t.Errorf("Send blocked on stalled consumer: %v", elapsed)
			}
		case <-time.After(10 * time.Second):
			t.Error("Send did not return")
		}
	}))
	defer server.Close()

	// 只建连从不读：模拟停滞的慢消费端。
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	time.Sleep(300 * time.Millisecond)
}
