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
