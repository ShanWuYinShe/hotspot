package realtime

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeTimeout is the deadline applied to each queued write by the writer
// goroutine. A slow or malicious consumer that stalls the TCP connection
// fails its own writes without blocking broadcast initiators.
const writeTimeout = 5 * time.Second

// sendQueueSize bounds each connection's outbound buffer. Broadcast Send
// never blocks the caller: a stalled consumer's queue fills and further
// events are dropped (fail-fast for one connection instead of stalling the
// HTTP goroutine sending a group message to hundreds of members).
const sendQueueSize = 128

// errSendQueueFull reports a dropped event: the consumer is slower than the
// broadcast rate. Callers (hub deliver paths) intentionally ignore Send
// errors, matching the previous best-effort broadcast semantics.
var errSendQueueFull = errors.New("realtime: send queue full, event dropped")

type socketSink struct {
	conn *websocket.Conn
	// mu serializes raw conn writes between the writer goroutine and
	// closeWithCode (gorilla forbids concurrent writers).
	mu        sync.Mutex
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
	// onSubscriptionRevoked 由 handler 在 RegisterUser 之前注册（此后只读，
	// 无需加锁）：服务端强制摘除订阅时同步连接侧订阅快照，见 Hub.revokeGroupLocal。
	onSubscriptionRevoked func(kind, id string)
}

func newSocketSink(conn *websocket.Conn) *socketSink {
	s := &socketSink{
		conn: conn,
		send: make(chan []byte, sendQueueSize),
		done: make(chan struct{}),
	}
	go s.writeLoop()
	return s
}

// revokeSubscription 实现 subscriptionRevoker：把强制撤销转交给连接层的快照回调。
func (s *socketSink) revokeSubscription(kind, id string) {
	if s.onSubscriptionRevoked != nil {
		s.onSubscriptionRevoked(kind, id)
	}
}

// Send marshals the event and enqueues it for the writer goroutine. It never
// blocks: on a full queue or closed sink it returns an error immediately.
func (s *socketSink) Send(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	select {
	case s.send <- data:
		return nil
	case <-s.done:
		return errors.New("realtime: sink closed")
	default:
		return errSendQueueFull
	}
}

func (s *socketSink) writeLoop() {
	for {
		select {
		case <-s.done:
			return
		case data := <-s.send:
			s.mu.Lock()
			_ = s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			writeErr := s.conn.WriteMessage(websocket.TextMessage, data)
			s.mu.Unlock()
			if writeErr != nil {
				return
			}
		}
	}
}

func (s *socketSink) Close() error {
	s.closeOnce.Do(func() { close(s.done) })
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Close()
}

// closeWithCode sends a standard WS close frame (spec api_spec_websocket.md
// 「Close Code 规范」) before closing the underlying connection, so clients
// can apply the specified reconnect semantics (4001 → refresh token & retry,
// 1008 → protocol violation, 1011 → backoff retry, 1000 → graceful).
func (s *socketSink) closeWithCode(code int, reason string) error {
	// 与 Close 同步标记 sink 已关闭：writeLoop 随之退出、后续 Send 直接拒绝。
	// 不依赖「调用方随后必定调用 Close()」的隐式约定。
	s.closeOnce.Do(func() { close(s.done) })
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	err := s.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(writeTimeout))
	if err != nil {
		// 写失败时连接多半已不可用，直接关闭即可。
		return s.conn.Close()
	}
	return s.conn.Close()
}
