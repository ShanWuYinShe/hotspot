package realtime

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeTimeout is the deadline applied before each WriteJSON call.
// A slow or malicious consumer that stalls the TCP connection will cause
// the write to fail, allowing the hub to drop the sink and continue
// delivering to other connections on the same channel.
const writeTimeout = 5 * time.Second

type socketSink struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *socketSink) Send(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return s.conn.WriteJSON(event)
}

func (s *socketSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Close()
}

// closeWithCode sends a standard WS close frame (spec api_spec_websocket.md
// 「Close Code 规范」) before closing the underlying connection, so clients
// can apply the specified reconnect semantics (4001 → refresh token & retry,
// 1008 → protocol violation, 1011 → backoff retry, 1000 → graceful).
func (s *socketSink) closeWithCode(code int, reason string) error {
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
