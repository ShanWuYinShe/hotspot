package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"hotspot/backend-go/internal/platform/observability"
)

// writeTimeout is the deadline applied to each queued write by the writer
// goroutine. A slow or malicious consumer that stalls the TCP connection
// fails its own writes without blocking broadcast initiators.
const writeTimeout = 5 * time.Second

// sendQueueSize bounds each connection's outbound buffer. Broadcast Send
// never blocks the caller: a stalled consumer's queue fills and further
// events are dropped (fail-fast for one connection instead of stalling the
// HTTP goroutine sending a group message to hundreds of members).
//
// Resume replay is the exception: it must not drop events silently, so the
// replay path uses SendBlocking with a caller deadline instead.
const sendQueueSize = 128

// errSendQueueFull reports a dropped event: the consumer is slower than the
// broadcast rate. Callers (hub deliver paths) intentionally ignore Send
// errors, matching the previous best-effort broadcast semantics.
var errSendQueueFull = errors.New("realtime: send queue full, event dropped")

// errSinkClosed reports an event that could not be queued because the
// connection sink is already closed.
var errSinkClosed = errors.New("realtime: sink closed")

// wsDroppedEventsTotal 让队列满丢事件可观测（此前 errSendQueueFull 被广播方
// 静默吞掉，弱网/慢消费者堆积完全不可见）。
var wsDroppedEventsTotal = promauto.With(observability.Registry()).NewCounter(
	prometheus.CounterOpts{
		Name: "realtime_ws_dropped_events_total",
		Help: "WS 出站队列满被丢弃的事件总数（按连接慢消费累计）",
	},
)

// outboundMessage 是写队列元素：普通事件携带已序列化 JSON；广播热路径携带
// 跨连接复用的 *websocket.PreparedMessage（wire 表示与压缩结果只准备一次）。
// seq 用于在真正写成功后推进连接的 resume 基线（未写成功的事件重连后仍需回放）。
type outboundMessage struct {
	data     []byte
	prepared *websocket.PreparedMessage
	seq      int64
}

type socketSink struct {
	conn *websocket.Conn
	// mu serializes raw conn writes between the writer goroutine and
	// closeWithCode (gorilla forbids concurrent writers).
	mu        sync.Mutex
	send      chan outboundMessage
	done      chan struct{}
	closeOnce sync.Once
	// lastWrittenSeq 记录真正写入 TCP 的最大 ServerSeq（writer goroutine 单写、
	// 断连收尾单读）。resume 基线取该值而不是全局 LastSeq：全局 seq 会被其它
	// 用户的事件推进，用它做基线会永久跳过本用户「已记录未送达」的事件。
	lastWrittenSeq atomic.Int64
	// onSubscriptionRevoked 由 handler 在 RegisterUser 之前注册（此后只读，
	// 无需加锁）：服务端强制摘除订阅时同步连接侧订阅快照，见 Hub.revokeGroupLocal。
	onSubscriptionRevoked func(kind, id string)
}

func newSocketSink(conn *websocket.Conn) *socketSink {
	s := &socketSink{
		conn: conn,
		send: make(chan outboundMessage, sendQueueSize),
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
	return s.enqueue(outboundMessage{data: data, seq: event.ServerSeq})
}

// SendPrepared enqueues a message whose JSON and compressed wire form were
// prepared once for many connections (fanout fast path). The sink never
// mutates the prepared message; gorilla caches the wire bytes per
// (server, compress, level) so WritePreparedMessage avoids a per-connection
// json.Marshal + permessage-deflate compression.
func (s *socketSink) SendPrepared(pm *websocket.PreparedMessage, seq int64) error {
	return s.enqueue(outboundMessage{prepared: pm, seq: seq})
}

// SendBlocking enqueues event, waiting for queue space until ctx is done. It
// exists for resume replay, which must not silently drop events when the
// backlog exceeds the queue size: dropping the newest events there would lose
// exactly the messages the client reconnected to recover.
func (s *socketSink) SendBlocking(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := outboundMessage{data: data, seq: event.ServerSeq}
	select {
	case s.send <- msg:
		return nil
	case <-s.done:
		return errSinkClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *socketSink) enqueue(msg outboundMessage) error {
	select {
	case s.send <- msg:
		return nil
	case <-s.done:
		return errSinkClosed
	default:
		wsDroppedEventsTotal.Inc()
		return errSendQueueFull
	}
}

// LastWrittenSeq returns the highest ServerSeq actually written to the
// connection. Used by the handler when persisting resume state: anything
// enqueued but not written will be replayed on reconnect.
func (s *socketSink) LastWrittenSeq() int64 { return s.lastWrittenSeq.Load() }

func (s *socketSink) writeLoop() {
	for {
		select {
		case <-s.done:
			return
		case msg := <-s.send:
			s.mu.Lock()
			_ = s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			var writeErr error
			if msg.prepared != nil {
				writeErr = s.conn.WritePreparedMessage(msg.prepared)
			} else {
				writeErr = s.conn.WriteMessage(websocket.TextMessage, msg.data)
			}
			s.mu.Unlock()
			if writeErr != nil {
				// 写失败即死连接：必须关闭 conn（并标记 done），否则本 goroutine
				// 退出后 sink 滞留 Hub 注册表继续吃广播，直到读侧恰好报错才被清理。
				// conn.Close 会让 handler 读循环立即失败，走统一断连清理路径。
				observability.L().Warn("realtime: sink write failed, closing connection", "error", writeErr)
				_ = s.Close()
				return
			}
			s.markWritten(msg.seq)
		}
	}
}

// markWritten 单调推进 lastWrittenSeq（仅 writer goroutine 调用，读侧用 CAS 兜底，
// 防止未来多处调用破坏单调性）。
func (s *socketSink) markWritten(seq int64) {
	if seq <= 0 {
		return
	}
	for {
		last := s.lastWrittenSeq.Load()
		if seq <= last || s.lastWrittenSeq.CompareAndSwap(last, seq) {
			return
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
