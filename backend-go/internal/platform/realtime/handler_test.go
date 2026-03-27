package realtime

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/ratelimit"
)

func wsURL(serverURL string) string { return "ws" + strings.TrimPrefix(serverURL, "http") + "/ws/v1" }

func TestRealtimeWebsocketLifecycle(t *testing.T) {
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession("user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	handler := NewHandler(authService)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	var established Event
	if err := conn.ReadJSON(&established); err != nil {
		t.Fatalf("read established: %v", err)
	}
	if established.Type != "ConnectionEstablished" {
		t.Fatalf("unexpected event: %#v", established)
	}

	if err := conn.WriteJSON(Event{Type: "Ping", Payload: PingPayload{}}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	var pong Event
	if err := conn.ReadJSON(&pong); err != nil {
		t.Fatalf("read pong: %v", err)
	}
	if pong.Type != "Pong" {
		t.Fatalf("unexpected pong: %#v", pong)
	}

	if err := conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req1", Operation: "set", Type: "user", IDs: []string{"user_bob"}}}); err != nil {
		t.Fatalf("write subscriptions: %v", err)
	}
	var ack Event
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Type != "SubscribeAck" {
		t.Fatalf("unexpected ack: %#v", ack)
	}
	var snapshot Event
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if snapshot.Type != "SubscriptionsSnapshot" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}

	otherResult, _ := authService.IssueSession("user_bob")
	otherConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial other websocket: %v", err)
	}
	defer otherConn.Close()
	_ = otherConn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: otherResult.AccessToken}})
	var establishedOther Event
	_ = otherConn.ReadJSON(&establishedOther)
	var online Event
	if err := conn.ReadJSON(&online); err != nil {
		t.Fatalf("read online event: %v", err)
	}
	if online.Type != "UserOnline" {
		t.Fatalf("unexpected online event: %#v", online)
	}

	_ = otherConn.WriteJSON(Event{Type: "PrivateTyping", Payload: PrivateTypingPayload{TargetUserID: "user_alice", IsTyping: true}})
	var typing Event
	if err := conn.ReadJSON(&typing); err != nil {
		t.Fatalf("read typing event: %v", err)
	}
	if typing.Type != "PrivateUserTyping" {
		t.Fatalf("unexpected typing event: %#v", typing)
	}
}

func TestRealtimeRequiresAuthFirst(t *testing.T) {
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	handler := NewHandler(authService)
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(Event{Type: "Ping", Payload: PingPayload{}}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	var event struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read error: %v", err)
	}
	if event.Type != "Error" || event.Payload.Severity != "fatal" {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestRealtimeSubscriptionLimit(t *testing.T) {
	ratelimit.SetSource(appconfig.NewStaticSource(config.Config{}))
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := NewHandler(authService)
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}})
	var established Event
	_ = conn.ReadJSON(&established)
	ids := make([]string, 501)
	for i := range ids {
		ids[i] = "u"
	}
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req2", Operation: "set", Type: "user", IDs: ids}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Type != "SubscribeAck" || ack.Payload.Code == 0 {
		payload, _ := json.Marshal(ack)
		t.Fatalf("unexpected ack: %s", payload)
	}
}

func TestRealtimeSubscriptionRateLimit(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(nil, func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.RateLimits.WSSubscribePerSecond = 1
		return cfg
	})
	ratelimit.SetSource(source)
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := NewHandler(authService)
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}})
	var established Event
	_ = conn.ReadJSON(&established)
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "conn-1", Operation: "set", Type: "user", IDs: []string{"user_bob"}}})
	var ack Event
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	var snapshot Event
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "conn-1", Operation: "set", Type: "user", IDs: []string{"user_charlie"}}})
	var limited struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&limited); err != nil {
		t.Fatalf("read limited event: %v", err)
	}
	if limited.Type != "Error" || limited.Payload.Code != 20300 || limited.Payload.RetryAfter <= 0 {
		payload, _ := json.Marshal(limited)
		t.Fatalf("unexpected limited event: %s", payload)
	}
}
