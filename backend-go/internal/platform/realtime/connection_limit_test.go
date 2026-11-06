package realtime

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
)

func TestHandlerConnectionLimitAcquireRelease(t *testing.T) {
	h := &Handler{}
	h.SetMaxConnections(2)
	first := h.tryAcquireConnection()
	second := h.tryAcquireConnection()
	if !first || !second {
		t.Fatal("first two connections must be accepted")
	}
	if h.tryAcquireConnection() {
		t.Fatal("third connection must be rejected at limit=2")
	}
	if got := h.activeConnections.Load(); got != 2 {
		t.Fatalf("active connections = %d, want 2", got)
	}
	h.releaseConnection()
	if !h.tryAcquireConnection() {
		t.Fatal("connection slot must be reusable after release")
	}
	if got := h.activeConnections.Load(); got != 2 {
		t.Fatalf("active connections after reuse = %d, want 2", got)
	}
}

func TestHandlerConnectionLimitUnlimitedByDefault(t *testing.T) {
	h := &Handler{}
	for i := 0; i < 100; i++ {
		if !h.tryAcquireConnection() {
			t.Fatalf("unlimited handler rejected connection %d", i+1)
		}
	}
}

// 达到全局上限的第二个连接：收到 fatal Error(20300) 后以 close 4002 断开；
// 关闭首个连接释放槽位后，新连接可正常建立。
func TestHandlerGlobalConnectionLimitRejectsExcess(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	session, err := authService.IssueSession(context.Background(), "user_limit")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	handler := newRealtimeHandlerForTest(authService)
	handler.SetMaxConnections(1)
	server := httptest.NewServer(handler)
	defer server.Close()

	first := dialLimitWS(t, server.URL, session.AccessToken)
	defer first.Close()
	if event := readLimitEvent(t, first); event.Type != EventTypeConnectionEstablished {
		t.Fatalf("first connection: %#v", event)
	}

	second := dialLimitWS(t, server.URL, session.AccessToken)
	defer second.Close()
	rejected := readLimitEvent(t, second)
	if rejected.Type != EventTypeError {
		t.Fatalf("expected Error event, got %#v", rejected)
	}
	payload, ok := rejected.Payload.(map[string]any)
	if !ok || int(payload["code"].(float64)) != int(CodeWSRateLimit) {
		t.Fatalf("expected code %d, got %#v", CodeWSRateLimit, rejected.Payload)
	}
	if _, _, err := second.ReadMessage(); err == nil {
		t.Fatal("expected close after rejection")
	} else if closeErr, ok := err.(*websocket.CloseError); !ok || closeErr.Code != 4002 {
		t.Fatalf("expected close 4002, got %v", err)
	}

	first.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		third := dialLimitWS(t, server.URL, session.AccessToken)
		_ = third.SetReadDeadline(time.Now().Add(2 * time.Second))
		var event Event
		readErr := third.ReadJSON(&event)
		third.Close()
		if readErr == nil && event.Type == EventTypeConnectionEstablished {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("slot was not released after first connection closed: %v", readErr)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func dialLimitWS(t *testing.T, serverURL, token string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(serverURL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: token}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	return conn
}

func readLimitEvent(t *testing.T, conn *websocket.Conn) Event {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var event Event
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read event: %v", err)
	}
	return event
}

func TestHubCollectorCountsRejectedConnections(t *testing.T) {
	hub := NewHub()
	hub.NoteConnectionRejected()
	hub.NoteConnectionRejected()
	registry := prometheus.NewRegistry()
	registry.MustRegister(NewHubCollector(hub))
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "realtime_connections_rejected_total" {
			continue
		}
		if got := family.GetMetric()[0].GetCounter().GetValue(); got != 2 {
			t.Fatalf("rejected metric = %v, want 2", got)
		}
		return
	}
	t.Fatal("realtime_connections_rejected_total metric missing")
}
