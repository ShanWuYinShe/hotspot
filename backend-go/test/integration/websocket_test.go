package integration

// WebSocket 全链路 E2E（审计第 46 轮 P0-1）：此前 test/integration 只有 HTTP 用例，
// WS 仅靠 realtime 包单测，握手/鉴权/订阅/心跳在真实 compose 环境下无回归覆盖。
// 本文件覆盖：连接→Auth→ConnectionEstablished→Ping/Pong→订阅 Ack/Snapshot，
// 以及「未鉴权先发消息」「token 非法」两条协议的 4001 关闭路径。

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type wsEvent struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	ServerSeq string          `json:"server_seq"`
}

type wsConnectionEstablished struct {
	ConnectionID string `json:"connection_id"`
	UserID       string `json:"user_id"`
	ServerTime   string `json:"server_time"`
	ResumeToken  string `json:"resume_token"`
	ServerSeq    string `json:"server_seq"`
	Config       struct {
		PingInterval int `json:"ping_interval"`
		PingTimeout  int `json:"ping_timeout"`
		MaxSubs      struct {
			User  int `json:"user"`
			Group int `json:"group"`
		} `json:"max_subscriptions"`
	} `json:"config"`
}

type wsSubscribeAck struct {
	ClientRequestID string `json:"client_request_id"`
	Operation       string `json:"operation"`
	Code            int    `json:"code"`
	Accepted        []struct {
		Type string   `json:"type"`
		IDs  []string `json:"ids"`
	} `json:"accepted"`
	FailedIDs []struct {
		Type           string `json:"type"`
		SubscriptionID string `json:"subscription_id"`
	} `json:"failed_ids"`
}

type wsSubscriptionsSnapshot struct {
	UserIDs  []string `json:"user_ids"`
	GroupIDs []string `json:"group_ids"`
}

type wsErrorPayload struct {
	Code     int    `json:"code"`
	Severity string `json:"severity"`
}

// wsEndpoint 由 BACKEND_URL 推导 WS 端点（ws://host:port/ws/v1）。
func wsEndpoint(t *testing.T) string {
	t.Helper()
	u, err := url.Parse(backendAddr)
	if err != nil {
		t.Fatalf("parse BACKEND_URL %q: %v", backendAddr, err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		t.Fatalf("unsupported BACKEND_URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/ws/v1"
	return u.String()
}

func dialWS(t *testing.T) *websocket.Conn {
	t.Helper()
	requireBackend(t)
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second, EnableCompression: true}
	conn, resp, err := dialer.Dial(wsEndpoint(t), nil)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("ws dial %s: %v (http_status=%d)", wsEndpoint(t), err, status)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func sendWSEvent(t *testing.T, conn *websocket.Conn, eventType string, payload any) {
	t.Helper()
	msg := map[string]any{"type": eventType, "payload": payload}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("ws write %s: %v", eventType, err)
	}
}

// readWSEvent 读取一条事件；超过超时时间未收到则失败。
func readWSEvent(t *testing.T, conn *websocket.Conn, timeout time.Duration) wsEvent {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var ev wsEvent
	if err := conn.ReadJSON(&ev); err != nil {
		t.Fatalf("ws read (timeout %s): %v", timeout, err)
	}
	return ev
}

// expectWSEvent 在读到的事件流中等待指定类型（跳过无关事件，最多 maxEvents 条），
// 返回该事件；用于订阅 Ack 等可能与其它事件交错下发的场景。
func expectWSEvent(t *testing.T, conn *websocket.Conn, want string, maxEvents int) wsEvent {
	t.Helper()
	for i := 0; i < maxEvents; i++ {
		ev := readWSEvent(t, conn, 5*time.Second)
		if ev.Type == want {
			return ev
		}
		t.Logf("ws: skip event %q while waiting for %q", ev.Type, want)
	}
	t.Fatalf("ws: did not receive event %q within %d events", want, maxEvents)
	return wsEvent{}
}

func TestWebSocket_ConnectAuthHeartbeatSubscriptions(t *testing.T) {
	requireBackend(t)
	// 本用例全程用显式 token，避免全局 accessToken 影响 guest 注册请求。
	prevToken := accessToken
	accessToken = ""
	defer func() { accessToken = prevToken }()
	userID, token := createGuestUser(t)

	conn := dialWS(t)
	sendWSEvent(t, conn, "Auth", map[string]any{
		"client_type":  "app",
		"access_token": token,
		"device_id":    uniqueDeviceID(t),
		"resume_token": nil,
	})

	ceEvent := expectWSEvent(t, conn, "ConnectionEstablished", 3)
	var ce wsConnectionEstablished
	if err := json.Unmarshal(ceEvent.Payload, &ce); err != nil {
		t.Fatalf("decode ConnectionEstablished: %v", err)
	}
	if ce.ConnectionID == "" || ce.UserID != userID {
		t.Fatalf("ConnectionEstablished fields: connection_id=%q user_id=%q want user_id=%q", ce.ConnectionID, ce.UserID, userID)
	}
	if _, err := time.Parse(time.RFC3339, ce.ServerTime); err != nil {
		t.Fatalf("server_time %q not RFC3339: %v", ce.ServerTime, err)
	}
	if _, err := strconv.ParseInt(ce.ServerSeq, 10, 64); err != nil {
		t.Fatalf("server_seq %q must be int64-as-string: %v", ce.ServerSeq, err)
	}
	if ce.Config.PingInterval != 30 || ce.Config.PingTimeout != 10 {
		t.Fatalf("heartbeat config = %+v, want ping_interval=30 ping_timeout=10", ce.Config)
	}
	if ce.Config.MaxSubs.User <= 0 || ce.Config.MaxSubs.Group <= 0 {
		t.Fatalf("max_subscriptions = %+v, want positive limits", ce.Config.MaxSubs)
	}

	// 心跳：Ping → Pong。
	sendWSEvent(t, conn, "Ping", map[string]any{})
	if ev := expectWSEvent(t, conn, "Pong", 3); ev.Type != "Pong" {
		t.Fatalf("expected Pong, got %q", ev.Type)
	}

	// 订阅自己（canSubscribeUser 允许 viewer==target）→ SubscribeAck accepted。
	sendWSEvent(t, conn, "SetSubscriptions", map[string]any{
		"operation":         "subscribe",
		"client_request_id": "e2e-subscribe-1",
		"items":             []map[string]any{{"type": "user", "ids": []string{userID}}},
	})
	ackEvent := expectWSEvent(t, conn, "SubscribeAck", 5)
	var ack wsSubscribeAck
	if err := json.Unmarshal(ackEvent.Payload, &ack); err != nil {
		t.Fatalf("decode SubscribeAck: %v", err)
	}
	if ack.Code != 0 || ack.Operation != "subscribe" || ack.ClientRequestID != "e2e-subscribe-1" {
		t.Fatalf("SubscribeAck mismatch: %+v", ack)
	}
	accepted := false
	for _, item := range ack.Accepted {
		if item.Type == "user" {
			for _, id := range item.IDs {
				if id == userID {
					accepted = true
				}
			}
		}
	}
	if !accepted {
		t.Fatalf("SubscribeAck accepted = %+v, want user %q", ack.Accepted, userID)
	}

	// 订阅快照应包含刚订阅的自己。
	sendWSEvent(t, conn, "GetSubscriptions", map[string]any{})
	snapEvent := expectWSEvent(t, conn, "SubscriptionsSnapshot", 5)
	var snap wsSubscriptionsSnapshot
	if err := json.Unmarshal(snapEvent.Payload, &snap); err != nil {
		t.Fatalf("decode SubscriptionsSnapshot: %v", err)
	}
	found := false
	for _, id := range snap.UserIDs {
		if id == userID {
			found = true
		}
	}
	if !found {
		t.Fatalf("SubscriptionsSnapshot user_ids = %v, want %q", snap.UserIDs, userID)
	}

	// 正常关闭：发送 close 帧后服务端应关闭连接。
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "e2e done"))
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected server to close the connection after client close frame")
	}
}

func TestWebSocket_AuthFailureClosesWith4001(t *testing.T) {
	requireBackend(t)

	t.Run("message_before_auth", func(t *testing.T) {
		conn := dialWS(t)
		// 5 秒内首帧必须是 Auth；先发 Ping 触发协议错误。
		sendWSEvent(t, conn, "Ping", map[string]any{})
		ev := readWSEvent(t, conn, 5*time.Second)
		if ev.Type != "Error" {
			t.Fatalf("expected Error before close, got %q", ev.Type)
		}
		var payload wsErrorPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			t.Fatalf("decode Error payload: %v", err)
		}
		if payload.Code != 20000 || payload.Severity != "fatal" {
			t.Fatalf("Error payload = %+v, want code=20000 severity=fatal", payload)
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, 4001) {
			t.Fatalf("expected close code 4001, got %v", err)
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		conn := dialWS(t)
		sendWSEvent(t, conn, "Auth", map[string]any{
			"client_type":  "app",
			"access_token": "invalid-token-for-e2e",
			"device_id":    uniqueDeviceID(t),
		})
		ev := readWSEvent(t, conn, 5*time.Second)
		if ev.Type != "Error" {
			t.Fatalf("expected Error for invalid token, got %q", ev.Type)
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, 4001) {
			t.Fatalf("expected close code 4001, got %v", err)
		}
	})
}
