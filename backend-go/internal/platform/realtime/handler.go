package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/ratelimit"
)

type userPresenceAuthorizer interface {
	OpenTheDoor(context.Context, string) (string, bool, error)
}

type userRelationAuthorizer interface {
	GetSettings(context.Context, string) (bool, string, bool, error)
	Relationship(context.Context, string, string) (bool, bool, bool, error)
}

type groupMembershipAuthorizer interface {
	ListMembers(context.Context, string, int, string) ([]string, string, error)
}

type Handler struct {
	auth           *platformauth.Service
	users          userPresenceAuthorizer
	userRepo       userRelationAuthorizer
	groupRepo      groupMembershipAuthorizer
	typingThrottle sync.Map
}

func NewHandler(auth *platformauth.Service, userService userPresenceAuthorizer, userRepo userRelationAuthorizer, groupRepo groupMembershipAuthorizer) *Handler {
	return &Handler{auth: auth, users: userService, userRepo: userRepo, groupRepo: groupRepo}
}

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
		_ = conn.WriteJSON(Event{Type: EventTypeError, Payload: ErrorPayload{Code: 20000, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
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
		_ = conn.WriteJSON(Event{Type: EventTypeError, Payload: ErrorPayload{Code: 20000, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
		return
	}
	principalCtx := platformauth.WithPrincipal(ctx, principal)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))

	sink := &socketSink{conn: conn}
	if DefaultHub().RegisterUser(principal.UserID, principal.SessionID, sink) {
		DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypeUserOnline, Payload: UserOnlinePayload{UserID: principal.UserID, OnlineAt: time.Now().UTC().Format(time.RFC3339)}})
	}
	defer func() {
		if DefaultHub().UnregisterUser(principal.UserID, principal.SessionID, sink) {
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypeUserOffline, Payload: UserOfflinePayload{UserID: principal.UserID, OfflineAt: time.Now().UTC().Format(time.RFC3339)}})
		}
	}()

	connectionID := fmt.Sprintf("conn_%d", time.Now().UTC().UnixNano())
	payload := ConnectionEstablishedPayload{ConnectionID: connectionID, UserID: principal.UserID, ServerTime: time.Now().UTC().Format(time.RFC3339)}
	payload.Config.PingInterval = 30
	payload.Config.PingTimeout = 10
	payload.Config.MaxSubscriptions.User = 500
	payload.Config.MaxSubscriptions.Group = 100
	if err := sink.Send(Event{Type: EventTypeConnectionEstablished, Payload: payload}); err != nil {
		return
	}

	subs := newSubscriptions()
	for {
		var msg Event
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		switch msg.Type {
		case "Ping":
			_ = sink.Send(Event{Type: EventTypePong, Payload: PingPayload{}})
		case "GetSubscriptions":
			var payload GetSubscriptionsPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			snapshot := subs.snapshot()
			snapshot.ClientRequestID = payload.ClientRequestID
			_ = sink.Send(Event{Type: EventTypeSubscriptionsSnapshot, Payload: snapshot})
		case "SetSubscriptions":
			var payload SetSubscriptionsPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			if retryAfter, limited := ratelimit.AllowWebsocketSubscribe(r.Context(), principal.UserID, connectionID); limited {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: 20300, Message: "请求过于频繁", Severity: "recoverable", Retryable: true, RetryAfter: retryAfter}})
				continue
			}
			limit := 500
			if payload.Type == "group" {
				limit = 100
			}
			failed := []FailedSubscription{}
			if len(payload.IDs) > limit {
				for _, id := range payload.IDs[limit:] {
					failed = append(failed, FailedSubscription{SubscriptionID: id, Reason: "limit_exceeded"})
				}
				payload.IDs = append([]string(nil), payload.IDs[:limit]...)
			}
			ack, snapshot := h.applyAuthorizedSubscriptions(principalCtx, principal.UserID, payload, subs, sink)
			ack.FailedIDs = append(ack.FailedIDs, failed...)
			if len(ack.FailedIDs) > 0 {
				ack.Code = 20100
				if ack.Message == "" {
					ack.Message = "订阅部分失败"
				}
			}
			_ = sink.Send(Event{Type: EventTypeSubscribeAck, Payload: ack})
			snapshot.ClientRequestID = payload.ClientRequestID
			_ = sink.Send(Event{Type: EventTypeSubscriptionsSnapshot, Payload: snapshot})
		case "PrivateTyping":
			var payload PrivateTypingPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			if !h.canSendPrivateTyping(principalCtx, principal.UserID, payload.TargetUserID) {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: 20400, Message: "无权限", Severity: "warning", Retryable: false}})
				continue
			}
			typingKey := fmt.Sprintf("typing:%s:%s", principal.UserID, payload.TargetUserID)
			if last, ok := h.typingThrottle.Load(typingKey); ok {
				if time.Since(last.(time.Time)) < 3*time.Second {
					continue
				}
			}
			h.typingThrottle.Store(typingKey, time.Now())
			DefaultHub().BroadcastToUser(payload.TargetUserID, Event{Type: EventTypePrivateUserTyping, Payload: map[string]any{"user_id": principal.UserID, "target_user_id": payload.TargetUserID, "is_typing": payload.IsTyping}})
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypePrivateUserTyping, Payload: map[string]any{"user_id": principal.UserID, "target_user_id": payload.TargetUserID, "is_typing": payload.IsTyping}})
		case "GroupTyping":
			var payload GroupTypingPayload
			blob, _ := json.Marshal(msg.Payload)
			_ = json.Unmarshal(blob, &payload)
			if !h.isGroupMember(principalCtx, principal.UserID, payload.GroupID) {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: 20400, Message: "无权限", Severity: "warning", Retryable: false}})
				continue
			}
			typingKey := fmt.Sprintf("typing:%s:%s", principal.UserID, payload.GroupID)
			if last, ok := h.typingThrottle.Load(typingKey); ok {
				if time.Since(last.(time.Time)) < 3*time.Second {
					continue
				}
			}
			h.typingThrottle.Store(typingKey, time.Now())
			DefaultHub().BroadcastToGroup(payload.GroupID, Event{Type: EventTypeGroupUserTyping, Payload: map[string]any{"user_id": principal.UserID, "group_id": payload.GroupID, "is_typing": payload.IsTyping}})
		case "Auth":
			_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: 20000, Message: "重复鉴权", Severity: "warning", Retryable: false}})
		default:
			_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: 20000, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
		}
	}
}

func (h *Handler) applyAuthorizedSubscriptions(ctx context.Context, viewerUserID string, payload SetSubscriptionsPayload, subs *subscriptions, sink FanoutSink) (SubscribeAckPayload, SubscriptionsSnapshotPayload) {
	ack := SubscribeAckPayload{ClientRequestID: payload.ClientRequestID, Operation: payload.Operation, Type: payload.Type, IDs: payload.IDs, Code: 0}
	allowed := make([]string, 0, len(payload.IDs))
	for _, id := range payload.IDs {
		if payload.Operation == "unsubscribe" || h.isSubscriptionAllowed(ctx, viewerUserID, payload.Type, id) {
			allowed = append(allowed, id)
			continue
		}
		ack.FailedIDs = append(ack.FailedIDs, FailedSubscription{SubscriptionID: id, Reason: "denied"})
	}
	if len(ack.FailedIDs) > 0 {
		ack.Code = 20100
		ack.Message = "订阅权限不足"
	}
	subs.apply(payload.Operation, payload.Type, allowed)
	snapshot := subs.snapshot()
	DefaultHub().SetUserSubscriptions(snapshot.UserIDs, sink)
	DefaultHub().SetGroupSubscriptions(snapshot.GroupIDs, sink)
	return ack, snapshot
}

func (h *Handler) isSubscriptionAllowed(ctx context.Context, viewerUserID, kind, id string) bool {
	switch kind {
	case "user":
		return h.canSubscribeUser(ctx, viewerUserID, id)
	case "group":
		return h.isGroupMember(ctx, viewerUserID, id)
	default:
		return false
	}
}

func (h *Handler) canSubscribeUser(ctx context.Context, viewerUserID, targetUserID string) bool {
	if viewerUserID == targetUserID {
		return true
	}
	if h.users != nil {
		presenceStatus, ok, err := h.users.OpenTheDoor(ctx, targetUserID)
		if err == nil && ok {
			return presenceStatus != ""
		}
	}
	if h.userRepo == nil {
		return true
	}
	presenceVisible, _, ok, err := h.userRepo.GetSettings(ctx, targetUserID)
	if err != nil {
		return false
	}
	if !ok {
		return true
	}
	return presenceVisible
}

func (h *Handler) canSendPrivateTyping(ctx context.Context, viewerUserID, targetUserID string) bool {
	if viewerUserID == "" || targetUserID == "" || h.userRepo == nil {
		return false
	}
	_, allowPrivateChat, ok, err := h.userRepo.GetSettings(ctx, targetUserID)
	if err != nil {
		return false
	}
	if !ok {
		allowPrivateChat = "everyone"
	}
	isFollowing, isFollowedBy, isBlocked, err := h.userRepo.Relationship(ctx, viewerUserID, targetUserID)
	if err != nil {
		return false
	}
	if isBlocked {
		return false
	}
	switch allowPrivateChat {
	case "nobody":
		return false
	case "friends":
		return isFollowing && isFollowedBy
	default:
		return true
	}
}

func (h *Handler) isGroupMember(ctx context.Context, viewerUserID, groupID string) bool {
	if viewerUserID == "" || groupID == "" || h.groupRepo == nil {
		return false
	}
	cursor := ""
	for {
		members, nextCursor, err := h.groupRepo.ListMembers(ctx, groupID, 100, cursor)
		if err != nil {
			return false
		}
		for _, userID := range members {
			if userID == viewerUserID {
				return true
			}
		}
		if nextCursor == "" || nextCursor == cursor {
			return false
		}
		cursor = nextCursor
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
