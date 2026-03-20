package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Handler struct{ auth *platformauth.Service }

func NewHandler(auth *platformauth.Service) *Handler { return &Handler{auth: auth} }

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var authEvent Event
	if err := conn.ReadJSON(&authEvent); err != nil || authEvent.Type != "Auth" {
		_ = conn.WriteJSON(Event{Type: "Error", Payload: ErrorPayload{Code: 20000, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
		return
	}
	var authPayload AuthPayload
	blob, _ := json.Marshal(authEvent.Payload)
	_ = json.Unmarshal(blob, &authPayload)
	ctx := platformauth.WithPlatform(r.Context(), detectPlatform(r, authPayload.ClientType))
	accessCookie := readCookie(r, platformauth.AccessTokenCookieName)
	authorization := r.Header.Get("Authorization")
	if authPayload.AccessToken != "" {
		authorization = "Bearer " + authPayload.AccessToken
	}
	principal, _, ok := h.auth.AuthenticateRequest(ctx, authorization, accessCookie)
	if !ok {
		_ = conn.WriteJSON(Event{Type: "Error", Payload: ErrorPayload{Code: 20000, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	sink := &socketSink{conn: conn}
	if DefaultHub().RegisterUser(principal.UserID, principal.SessionID, sink) {
		DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: "UserOnline", Payload: UserOnlinePayload{UserID: principal.UserID, OnlineAt: time.Now().UTC().Format(time.RFC3339)}})
	}
	defer func() {
		if DefaultHub().UnregisterUser(principal.UserID, principal.SessionID, sink) {
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: "UserOffline", Payload: UserOfflinePayload{UserID: principal.UserID, OfflineAt: time.Now().UTC().Format(time.RFC3339)}})
		}
	}()

	payload := ConnectionEstablishedPayload{ConnectionID: fmt.Sprintf("conn_%d", time.Now().UTC().UnixNano()), UserID: principal.UserID, ServerTime: time.Now().UTC().Format(time.RFC3339)}
	payload.Config.PingInterval = 30
	payload.Config.PingTimeout = 10
	payload.Config.MaxSubscriptions.User = 500
	payload.Config.MaxSubscriptions.Group = 100
	if err := conn.WriteJSON(Event{Type: "ConnectionEstablished", Payload: payload}); err != nil {
		return
	}

	subs := newSubscriptions()
	for {
		var msg Event
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		switch msg.Type {
		case "Ping":
			_ = conn.WriteJSON(Event{Type: "Pong", Payload: PingPayload{}})
		case "GetSubscriptions":
			var payload GetSubscriptionsPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			snapshot := subs.snapshot()
			snapshot.ClientRequestID = payload.ClientRequestID
			_ = conn.WriteJSON(Event{Type: "SubscriptionsSnapshot", Payload: snapshot})
		case "SetSubscriptions":
			var payload SetSubscriptionsPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			limit := 500
			if payload.Type == "group" {
				limit = 100
			}
			if len(payload.IDs) > limit {
				_ = conn.WriteJSON(Event{Type: "SubscribeAck", Payload: SubscribeAckPayload{ClientRequestID: payload.ClientRequestID, Operation: payload.Operation, Type: payload.Type, IDs: payload.IDs, Code: 20100, Message: "订阅数量超出限制", FailedIDs: []FailedSubscription{{SubscriptionID: payload.IDs[limit], Reason: "limit_exceeded"}}}})
				continue
			}
			subs.apply(payload.Operation, payload.Type, payload.IDs)
			snapshot := subs.snapshot()
			DefaultHub().SetUserSubscriptions(snapshot.UserIDs, sink)
			DefaultHub().SetGroupSubscriptions(snapshot.GroupIDs, sink)
			_ = conn.WriteJSON(Event{Type: "SubscribeAck", Payload: SubscribeAckPayload{ClientRequestID: payload.ClientRequestID, Operation: payload.Operation, Type: payload.Type, IDs: payload.IDs, Code: 0}})
			snapshot.ClientRequestID = payload.ClientRequestID
			_ = conn.WriteJSON(Event{Type: "SubscriptionsSnapshot", Payload: snapshot})
		case "PrivateTyping":
			var payload PrivateTypingPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			DefaultHub().BroadcastToUser(payload.TargetUserID, Event{Type: "PrivateUserTyping", Payload: map[string]any{"user_id": principal.UserID, "target_user_id": payload.TargetUserID, "is_typing": payload.IsTyping}})
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: "PrivateUserTyping", Payload: map[string]any{"user_id": principal.UserID, "target_user_id": payload.TargetUserID, "is_typing": payload.IsTyping}})
		case "GroupTyping":
			var payload GroupTypingPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			DefaultHub().BroadcastToGroup(payload.GroupID, Event{Type: "GroupUserTyping", Payload: map[string]any{"user_id": principal.UserID, "group_id": payload.GroupID, "is_typing": payload.IsTyping}})
		case "Auth":
			_ = conn.WriteJSON(Event{Type: "Error", Payload: ErrorPayload{Code: 20000, Message: "重复鉴权", Severity: "warning", Retryable: false}})
		default:
			_ = conn.WriteJSON(Event{Type: "Error", Payload: ErrorPayload{Code: 20000, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
		}
	}
}

func detectPlatform(r *http.Request, clientType string) string {
	if clientType != "" {
		return clientType
	}
	if readCookie(r, platformauth.AccessTokenCookieName) != "" {
		return "web"
	}
	if r.Header.Get("Authorization") != "" {
		return "app"
	}
	return "anonymous"
}
func readCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
