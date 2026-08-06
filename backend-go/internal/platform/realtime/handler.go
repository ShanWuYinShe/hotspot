package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/worker"

	"github.com/gorilla/websocket"
)

// userPresenceAuthorizer 查询目标用户详情（在场状态）。OpenTheDoor 为刻意
// 命名（「查户口」= 取详情），与 moments 的瞬间详情接口同语义，禁止改名。
type userPresenceAuthorizer interface {
	OpenTheDoor(context.Context, string) (string, bool, error)
}

// typingActorProvider resolves nickname/avatar for the typing actor (UserInfo
// in the pushed typing payloads). Implemented by the users module Service via
// bootstrap wiring; nil falls back to {user_id, nickname=user_id}.
type typingActorProvider interface {
	GetActorInfo(context.Context, string) (actor.Info, error)
}

type userRelationAuthorizer interface {
	GetSettings(context.Context, string) (bool, string, bool, error)
	Relationship(context.Context, string, string) (bool, bool, bool, error)
}

type groupMembershipAuthorizer interface {
	ListMembers(context.Context, string, int, string) ([]string, string, error)
	IsMember(context.Context, string, string) (bool, error)
}

type Handler struct {
	auth           *platformauth.Service
	users          userPresenceAuthorizer
	userRepo       userRelationAuthorizer
	groupRepo      groupMembershipAuthorizer
	actorProvider  typingActorProvider
	typingThrottle sync.Map
	stop           context.CancelFunc
	wg             sync.WaitGroup
}

// NewHandler creates a realtime WS handler. The typing-throttle sweeper is
// bound to ctx: when ctx is cancelled (or Close is called) the sweeper exits.
// Callers that embed the handler for the process lifetime — e.g. bootstrap in
// production — should pass the app's root context so shutdown stops the
// goroutine. Tests that build a handler per test can pass context.Background()
// and rely on Close (invoked via server.Close cleanup) to stop the sweeper.
func NewHandler(auth *platformauth.Service, userService userPresenceAuthorizer, userRepo userRelationAuthorizer, groupRepo groupMembershipAuthorizer) *Handler {
	return NewHandlerWithContext(context.Background(), auth, userService, userRepo, groupRepo)
}

// NewHandlerWithContext is the context-aware constructor. Prefer this from
// bootstrap so the sweeper is tied to the application lifetime.
func NewHandlerWithContext(ctx context.Context, auth *platformauth.Service, userService userPresenceAuthorizer, userRepo userRelationAuthorizer, groupRepo groupMembershipAuthorizer) *Handler {
	sweepCtx, cancel := context.WithCancel(ctx)
	h := &Handler{auth: auth, users: userService, userRepo: userRepo, groupRepo: groupRepo, stop: cancel}
	h.wg.Add(1)
	worker.Go("realtime.typing_sweeper", func() { h.sweepTypingThrottle(sweepCtx) })
	return h
}

// Close stops the background typing-throttle sweeper and waits for it to exit.
// Safe to call multiple times.
func (h *Handler) Close() {
	if h.stop != nil {
		h.stop()
	}
	h.wg.Wait()
}

// SetActorProvider wires the user-info resolver used to build the actor
// (UserInfo) field of pushed typing payloads. Optional: nil falls back to
// {user_id, nickname=user_id}.
func (h *Handler) SetActorProvider(p typingActorProvider) { h.actorProvider = p }

// resolveTypingActor builds the actor UserInfo map for the typing event.
func (h *Handler) resolveTypingActor(ctx context.Context, userID string) map[string]any {
	if h.actorProvider != nil {
		if info, err := h.actorProvider.GetActorInfo(ctx, userID); err == nil {
			actorInfo := map[string]any{"user_id": info.UserID, "nickname": info.Nickname}
			if info.AvatarURL != "" {
				actorInfo["avatar_url"] = info.AvatarURL
			}
			return actorInfo
		}
	}
	return map[string]any{"user_id": userID, "nickname": userID}
}

// sweepTypingThrottle periodically drops stale typing-throttle entries so the
// sync.Map doesn't grow unboundedly with each unique (user,target) pair over
// the process lifetime. The throttle window is 3s; entries older than a minute
// are dead. It exits when ctx is cancelled.
func (h *Handler) sweepTypingThrottle(ctx context.Context) {
	defer h.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-time.Minute)
			h.typingThrottle.Range(func(key, value any) bool {
				if value.(time.Time).Before(cutoff) {
					h.typingThrottle.Delete(key)
				}
				return true
			})
		}
	}
}

// upgrader validates the WebSocket Origin against the request Host. WS auth is
// cookie-based, so a permissive CheckOrigin would enable cross-site WebSocket
// hijacking (a malicious page opens a WS to the legit server riding the
// victim's cookies). Same-origin is the safe default for browser clients;
// native clients (HarmonyOS) don't send a browser-style Origin header.
var upgrader = websocket.Upgrader{CheckOrigin: sameOrigin}

// sameOrigin allows the upgrade when the Origin header is absent (non-browser
// client) or matches the request's own scheme+host. It blocks cross-origin
// browser WebSocket connections, which is the cookie-theft vector for CSWSH.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // native/non-browser client
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == r.Host
}

// decodePayload converts the generic `Event.Payload` (decoded as
// map[string]any by ReadJSON) into a concrete payload struct. It reports an
// error instead of silently continuing with a zero-value payload.
func decodePayload(payload any, out any) error {
	blob, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(blob, out)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Prevent memory-exhaustion DoS: the largest legitimate message is a
	// SetSubscriptions with up to 500 user/group IDs (~64 KB).
	conn.SetReadLimit(65536)

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var authEvent Event
	if err := conn.ReadJSON(&authEvent); err != nil || authEvent.Type != "Auth" {
		observability.L().Warn("ws: auth failed, invalid auth message", "error", err)
		_ = conn.WriteJSON(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
		return
	}
	var authPayload AuthPayload
	if err := decodePayload(authEvent.Payload, &authPayload); err != nil {
		observability.L().Warn("ws: auth failed, invalid payload", "error", err)
		_ = conn.WriteJSON(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
		return
	}
	ctx := platformauth.WithPlatform(r.Context(), detectPlatform(r, authPayload.ClientType))
	accessCookie := readCookie(r, platformauth.AccessTokenCookieName)
	authorization := r.Header.Get("Authorization")
	if authPayload.AccessToken != "" {
		authorization = "Bearer " + authPayload.AccessToken
	}
	principal, _, ok := h.auth.AuthenticateRequest(ctx, authorization, accessCookie)
	if !ok {
		observability.L().Warn("ws: auth failed, token invalid")
		_ = conn.WriteJSON(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
		return
	}
	principalCtx := platformauth.WithPrincipal(ctx, principal)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))

	connectionID := fmt.Sprintf("conn_%d", time.Now().UTC().UnixNano())
	observability.L().Info("ws: connected", "user_id", principal.UserID, "session_id", principal.SessionID, "connection_id", connectionID)

	sink := &socketSink{conn: conn}
	if DefaultHub().RegisterUser(principal.UserID, principal.SessionID, sink) {
		DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypeUserOnline, Payload: UserOnlinePayload{UserID: principal.UserID, OnlineAt: time.Now().UTC().Format(time.RFC3339)}})
	}
	defer func() {
		if DefaultHub().UnregisterUser(principal.UserID, principal.SessionID, sink) {
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypeUserOffline, Payload: UserOfflinePayload{UserID: principal.UserID, OfflineAt: time.Now().UTC().Format(time.RFC3339)}})
		}
		observability.L().Info("ws: disconnected", "user_id", principal.UserID, "session_id", principal.SessionID, "connection_id", connectionID)
	}()

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
			observability.L().Debug("ws: read error, closing", "user_id", principal.UserID, "connection_id", connectionID, "error", err)
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		switch msg.Type {
		case "Ping":
			_ = sink.Send(Event{Type: EventTypePong, Payload: PingPayload{}})
		case "GetSubscriptions":
			var payload GetSubscriptionsPayload
			if err := decodePayload(msg.Payload, &payload); err != nil {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
				continue
			}
			snapshot := subs.snapshot()
			snapshot.ClientRequestID = payload.ClientRequestID
			_ = sink.Send(Event{Type: EventTypeSubscriptionsSnapshot, Payload: snapshot})
		case "SetSubscriptions":
			var payload SetSubscriptionsPayload
			if err := decodePayload(msg.Payload, &payload); err != nil {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
				continue
			}
			if retryAfter, limited := ratelimit.AllowWebsocketSubscribe(r.Context(), principal.UserID, connectionID); limited {
				observability.L().Warn("ws: subscribe rate limited", "user_id", principal.UserID, "connection_id", connectionID, "retry_after", retryAfter)
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSRateLimit, Message: "请求过于频繁", Severity: "recoverable", Retryable: true, RetryAfter: retryAfter}})
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
		case "SendPrivateTyping":
			var payload PrivateTypingSendPayload
			if err := decodePayload(msg.Payload, &payload); err != nil {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
				continue
			}
			if !h.canSendPrivateTyping(principalCtx, principal.UserID, payload.TargetUserID) {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSPermission, Message: "无权限", Severity: "warning", Retryable: false}})
				continue
			}
			typingKey := fmt.Sprintf("typing:%s:%s", principal.UserID, payload.TargetUserID)
			if last, ok := h.typingThrottle.Load(typingKey); ok {
				if time.Since(last.(time.Time)) < 3*time.Second {
					continue
				}
			}
			h.typingThrottle.Store(typingKey, time.Now())
			actorInfo := h.resolveTypingActor(principalCtx, principal.UserID)
			DefaultHub().BroadcastToUser(payload.TargetUserID, Event{Type: EventTypePrivateTyping, Payload: PrivateTypingPayload{TargetUserID: payload.TargetUserID, Actor: actorInfo, IsTyping: payload.IsTyping}})
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypePrivateTyping, Payload: PrivateTypingPayload{TargetUserID: payload.TargetUserID, Actor: actorInfo, IsTyping: payload.IsTyping}})
		case "SendGroupTyping":
			var payload GroupTypingSendPayload
			if err := decodePayload(msg.Payload, &payload); err != nil {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
				continue
			}
			if !h.isGroupMember(principalCtx, principal.UserID, payload.GroupID) {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSPermission, Message: "无权限", Severity: "warning", Retryable: false}})
				continue
			}
			typingKey := fmt.Sprintf("typing:%s:%s", principal.UserID, payload.GroupID)
			if last, ok := h.typingThrottle.Load(typingKey); ok {
				if time.Since(last.(time.Time)) < 3*time.Second {
					continue
				}
			}
			h.typingThrottle.Store(typingKey, time.Now())
			DefaultHub().BroadcastToGroup(payload.GroupID, Event{Type: EventTypeGroupTyping, Payload: GroupTypingPayload{GroupID: payload.GroupID, Actor: h.resolveTypingActor(principalCtx, principal.UserID), IsTyping: payload.IsTyping}})
		case "Auth":
			_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "重复鉴权", Severity: "warning", Retryable: false}})
		default:
			_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
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
	// Single indexed lookup. The previous implementation paged through the
	// entire member list (100 per page) to answer a yes/no question, which on a
	// 10k-member group meant ~100 DB round trips per WS message.
	ok, err := h.groupRepo.IsMember(ctx, groupID, viewerUserID)
	if err != nil {
		return false
	}
	return ok
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
