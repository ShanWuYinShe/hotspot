package realtime

import (
	"sync"

	"github.com/gorilla/websocket"
)

type socketSink struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *socketSink) Send(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteJSON(event)
}

func (s *socketSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Close()
}
