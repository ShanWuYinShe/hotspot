package realtime

import "github.com/gorilla/websocket"

type socketSink struct{ conn *websocket.Conn }

func (s *socketSink) Send(event Event) error { return s.conn.WriteJSON(event) }
func (s *socketSink) Close() error           { return s.conn.Close() }
