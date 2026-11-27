package realtime

// payload_spec_test.go — WS 协议控制事件的 JSON 形状契约（spec
// api_spec_websocket.md）。恒定 nullable / 恒在数组字段必须显式下发 null/[]，
// 不得因 omitempty 缺键。

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"

	"github.com/gorilla/websocket"
)

// TestSubscribeAckPayloadJSONShape：成功订阅的 ack 必须携带 `message: null`
// 与 `failed_ids: []`（spec：message 为 string | null，failed_ids 空数组 =
// 全部成功），而不是缺键或 null 数组。
func TestSubscribeAckPayloadJSONShape(t *testing.T) {
	raw, err := json.Marshal(SubscribeAckPayload{
		ClientRequestID: "req-1",
		Operation:       "set",
		Accepted:        []SubscriptionItem{},
		Code:            0,
		FailedIDs:       []FailedSubscription{},
	})
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal ack: %v", err)
	}
	for _, key := range []string{"client_request_id", "operation", "accepted", "code", "message", "failed_ids"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("ack missing key %q: %s", key, raw)
		}
	}
	if string(fields["message"]) != "null" {
		t.Fatalf("message must be null on success, got %s", raw)
	}
	if string(fields["failed_ids"]) != "[]" {
		t.Fatalf("failed_ids must be [] on success, got %s", raw)
	}
}

// TestRealtimeSubscribeAckSuccessCarriesConstantNullableFields 走真实 WS 握手：
// 订阅自身通道（无需其他用户数据）成功时，帧里必须能读到 message=null 与
// failed_ids=[] 两个键。
func TestRealtimeSubscribeAckSuccessCarriesConstantNullableFields(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	handler := newRealtimeHandlerForTest(authService)
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	if ce := readEventWithTimeout(t, conn); ce.Type != EventTypeConnectionEstablished {
		t.Fatalf("want ConnectionEstablished, got %s", ce.Type)
	}
	if err := conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{
		ClientRequestID: "req-ok",
		Operation:       "set",
		Items:           []SubscriptionItem{{Type: "user", IDs: []string{"user_alice"}}},
	}}); err != nil {
		t.Fatalf("write subscriptions: %v", err)
	}
	ack := readEventWithTimeout(t, conn)
	if ack.Type != EventTypeSubscribeAck {
		t.Fatalf("want SubscribeAck, got %s", ack.Type)
	}
	payload, ok := ack.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", ack.Payload)
	}
	if payload["code"] != float64(0) {
		t.Fatalf("want success code 0, got %#v", payload)
	}
	messageValue, hasMessage := payload["message"]
	if !hasMessage || messageValue != nil {
		t.Fatalf("message key must be present with null, got %#v", payload)
	}
	failedIDs, hasFailed := payload["failed_ids"]
	failedSlice, isSlice := failedIDs.([]any)
	if !hasFailed || !isSlice || len(failedSlice) != 0 {
		t.Fatalf("failed_ids key must be present as [], got %#v", payload)
	}
}

// TestConnectionEstablishedAlwaysCarriesResumeTokenKey：spec 定义
// resume_token 为恒在 string（客户端据此决定是否尝试增量恢复）。此前字段带
// omitempty，Redis 不可用时整键消失、客户端读到 undefined（契约审计 B09）。
func TestConnectionEstablishedAlwaysCarriesResumeTokenKey(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	handler := newRealtimeHandlerForTest(authService)
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	ce := readEventWithTimeout(t, conn)
	if ce.Type != EventTypeConnectionEstablished {
		t.Fatalf("want ConnectionEstablished, got %s", ce.Type)
	}
	payload, ok := ce.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", ce.Payload)
	}
	token, hasToken := payload["resume_token"]
	if !hasToken {
		t.Fatalf("resume_token key must always be present: %#v", payload)
	}
	if _, isString := token.(string); !isString {
		t.Fatalf("resume_token must be a JSON string, got %#v", token)
	}
}
