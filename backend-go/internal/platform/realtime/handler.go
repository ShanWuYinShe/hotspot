package realtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idgen"
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
	BlockedEitherWay(context.Context, string, string) (callerBlocked, targetBlocked bool, err error)
}

type groupMembershipAuthorizer interface {
	ListMembers(context.Context, string, int, string) ([]string, string, error)
	IsMember(context.Context, string, string) (bool, error)
}

// banAuthorizer reports whether a user is currently banned platform-wide.
// Implemented by the admin module's repository via bootstrap wiring. Without
// it the realtime channel would accept connections from banned accounts —
// the HTTP-side ban middleware never runs on the WS handshake path.
type banAuthorizer interface {
	IsBanned(context.Context, string) (bool, error)
}

type Handler struct {
	auth           *platformauth.Service
	users          userPresenceAuthorizer
	userRepo       userRelationAuthorizer
	groupRepo      groupMembershipAuthorizer
	bans           banAuthorizer
	actorProvider  typingActorProvider
	typingThrottle sync.Map
	resumeStore    ResumeStore
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

// SetBanAuthorizer wires the platform-wide ban checker consulted at WS auth.
// Optional (nil = no ban check, dev/test default); the fail-open policy matches
// the HTTP RequiredAuth middleware so both paths degrade identically.
func (h *Handler) SetBanAuthorizer(b banAuthorizer) { h.bans = b }

// SetResumeStore 注入 resume 增量恢复存储（spec L156, L159）。nil 时退化为 NopResumeStore
// 语义：token 仍下发但 LoadState 恒空，客户端回退全量重放（spec L161 降级保证）。
func (h *Handler) SetResumeStore(s ResumeStore) { h.resumeStore = s }

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
				if t, ok := value.(time.Time); ok && t.Before(cutoff) {
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

// sendCloseFrame writes a standard WS close frame (spec「Close Code 规范」).
// The server then closes the connection; clients apply the code-specific
// reconnect semantics (4001 auth failure/session revoked → refresh & retry).
func sendCloseFrame(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(5*time.Second))
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
		sendCloseFrame(conn, 4001, "auth failed")
		return
	}
	var authPayload AuthPayload
	if err := decodePayload(authEvent.Payload, &authPayload); err != nil {
		observability.L().Warn("ws: auth failed, invalid payload", "error", err)
		_ = conn.WriteJSON(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "鉴权失败", Severity: "fatal", Retryable: false}})
		sendCloseFrame(conn, 4001, "auth failed")
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
		sendCloseFrame(conn, 4001, "auth failed")
		return
	}
	// 封禁检查（HTTP 中间件不覆盖 WS 握手路径）：被封禁账号不得建立实时连接，
	// 否则封禁仅剩 HTTP 生效、实时通道（订阅/typing/事件）完全绕过。
	// fail-open 与 RequiredAuth 同口径：ban checker 瞬时故障时不锁死全部连接。
	if h.bans != nil {
		if banned, err := h.bans.IsBanned(ctx, principal.UserID); err == nil && banned {
			observability.L().Warn("ws: auth rejected, account banned", "user_id", principal.UserID)
			_ = conn.WriteJSON(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSAccountBanned, Message: "账号已被封禁", Severity: "fatal", Retryable: false}})
			sendCloseFrame(conn, 4003, "account banned")
			return
		}
	}
	principalCtx := platformauth.WithPrincipal(ctx, principal)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))

	// idgen 随机后缀：UnixNano 在同纳秒并发/多实例下可重复，而 connectionID
	// 进入日志与 WS 订阅限流 key，重复会互相干扰。
	connectionID := "conn_" + idgen.MustWithPrefix("")
	resumeToken := generateResumeToken()
	observability.L().Info("ws: connected", "user_id", principal.UserID, "session_id", principal.SessionID, "connection_id", connectionID)

	// subs 提前声明：断线 defer 需保存订阅快照（spec L156「订阅状态由服务端恢复」），
	// resume 回放聚合也依赖断线时快照中的 GroupIDs。
	subs := newSubscriptions()
	sink := &socketSink{conn: conn}
	if DefaultHub().RegisterUser(principal.UserID, principal.SessionID, sink) {
		DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypeUserOnline, Payload: UserOnlinePayload{UserID: principal.UserID, OnlineAt: time.Now().UTC().Format(time.RFC3339)}})
	}
	defer func() {
		// resume 基线必须在 UnregisterUser **之前**捕获：注销后并发 Broadcast 对
		// 该用户 Record 的事件已无 sink 可投递，若注销后才取全局 LastSeq 写入快照，
		// 这些「已记录未送达」的事件会被高基线永久跳过（既没实时送达也不被补回）。
		// 先定格基线的代价是可能重放已送达事件——客户端按 ServerSeq 去重，可接受。
		var baselineSeq int64
		if h.resumeStore != nil {
			baselineSeq = h.resumeStore.LastSeq()
		}
		if DefaultHub().UnregisterUser(principal.UserID, principal.SessionID, sink) {
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypeUserOffline, Payload: UserOfflinePayload{UserID: principal.UserID, OfflineAt: time.Now().UTC().Format(time.RFC3339)}})
		}
		observability.L().Info("ws: disconnected", "user_id", principal.UserID, "session_id", principal.SessionID, "connection_id", connectionID)
		// resume：连接断开时保存 state（LastSeq + 订阅快照），作为重连续传起点与
		// 服务端订阅恢复依据（spec L156）。多端登录共享全局 seq 流，LastSeq 取全局。
		if h.resumeStore != nil {
			h.resumeStore.SaveState(resumeToken, ResumeState{UserID: principal.UserID, LastSeq: baselineSeq, Subs: subs.snapshot()})
		}
	}()

	// 全局事件流的最大 seq（连接建立基线）。store 为 nil（Redis 不可用）时
	// ServerSeq=0、ResumeToken 仍下发但 LoadState 恒空 → 客户端回退全量重放（spec L161）。
	currentLastSeq := int64(0)
	if h.resumeStore != nil {
		currentLastSeq = h.resumeStore.LastSeq()
	}
	payload := ConnectionEstablishedPayload{ConnectionID: connectionID, UserID: principal.UserID, ServerTime: time.Now().UTC().Format(time.RFC3339), ResumeToken: resumeToken, ServerSeq: currentLastSeq}
	payload.Config.PingInterval = 30
	payload.Config.PingTimeout = 10
	payload.Config.MaxSubscriptions.User = 500
	payload.Config.MaxSubscriptions.Group = 100
	if err := sink.Send(Event{Type: EventTypeConnectionEstablished, Payload: payload}); err != nil {
		return
	}

	// resume 续传（spec L156）：客户端携带的旧 token 有效且属同一 user 时——
	// 1. 服务端按断线时订阅快照恢复订阅（带权限校验：断线期间已退出的群不恢复、
	//    不聚合回放，防止越权接收）；
	// 2. 补拉断线期间事件：user 流 + 快照内各 group 流按全局 seq 归并回放（已带
	//    ServerSeq）。回放事件经 sink.Send 直达、不经 BroadcastToUser/Group，故不会
	//    重复 Record。前端重连后仍会 replaySubscriptions（set 幂等覆盖，本地订阅
	//    为权威），服务端恢复订阅保证两者一致前的事件送达。
	if h.resumeStore != nil && authPayload.ResumeToken != "" {
		if state, ok := h.resumeStore.LoadState(authPayload.ResumeToken); ok && state.UserID == principal.UserID {
			// 断线快照转 items（spec：SetSubscriptions 批量结构，单请求覆盖两类订阅）
			items := make([]SubscriptionItem, 0, 2)
			if len(state.Subs.GroupIDs) > 0 {
				items = append(items, SubscriptionItem{Type: "group", IDs: state.Subs.GroupIDs})
			}
			if len(state.Subs.UserIDs) > 0 {
				items = append(items, SubscriptionItem{Type: "user", IDs: state.Subs.UserIDs})
			}
			_, snap := h.applyAuthorizedSubscriptions(principalCtx, principal.UserID, SetSubscriptionsPayload{Operation: "set", Items: items}, subs, sink)
			for _, ev := range h.resumeStore.Replay(principal.UserID, snap.GroupIDs, state.LastSeq) {
				_ = sink.Send(ev)
			}
		}
	}

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
			// 按类型限额裁剪（user 500 / group 100，与 CE config 一致）；超限与
			// 未知类型计入 failed_ids（reason: limit_exceeded / invalid）。
			failed := []FailedSubscription{}
			items := make([]SubscriptionItem, 0, len(payload.Items))
			for _, item := range payload.Items {
				limit, ok := map[string]int{"user": 500, "group": 100}[item.Type]
				if !ok {
					for _, id := range item.IDs {
						failed = append(failed, FailedSubscription{Type: item.Type, SubscriptionID: id, Reason: "invalid"})
					}
					continue
				}
				if len(item.IDs) > limit {
					for _, id := range item.IDs[limit:] {
						failed = append(failed, FailedSubscription{Type: item.Type, SubscriptionID: id, Reason: "limit_exceeded"})
					}
					item.IDs = append([]string(nil), item.IDs[:limit]...)
				}
				items = append(items, item)
			}
			ack, snapshot := h.applyAuthorizedSubscriptions(principalCtx, principal.UserID, SetSubscriptionsPayload{ClientRequestID: payload.ClientRequestID, Operation: payload.Operation, Items: items}, subs, sink)
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
				if t, ok := last.(time.Time); ok && time.Since(t) < 3*time.Second {
					continue
				}
			}
			h.typingThrottle.Store(typingKey, time.Now())
			actorInfo := h.resolveTypingActor(principalCtx, principal.UserID)
			// typing 是瞬态事件：不写 resume 流（写流会稀释 seq 空间、膨胀
			// Stream，且重连回放陈旧 typing 指示是错误 UX）。
			DefaultHub().BroadcastToUserNoRecord(payload.TargetUserID, Event{Type: EventTypePrivateTyping, Payload: PrivateTypingPayload{TargetUserID: payload.TargetUserID, Actor: actorInfo, IsTyping: payload.IsTyping}})
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
				if t, ok := last.(time.Time); ok && time.Since(t) < 3*time.Second {
					continue
				}
			}
			h.typingThrottle.Store(typingKey, time.Now())
			DefaultHub().BroadcastToGroupNoRecord(payload.GroupID, Event{Type: EventTypeGroupTyping, Payload: GroupTypingPayload{GroupID: payload.GroupID, Actor: h.resolveTypingActor(principalCtx, principal.UserID), IsTyping: payload.IsTyping}})
		case "Auth":
			_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "重复鉴权", Severity: "warning", Retryable: false}})
		default:
			_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSProtocol, Message: "消息格式错误", Severity: "recoverable", Retryable: true, RetryAfter: 1}})
		}
	}
}

func (h *Handler) applyAuthorizedSubscriptions(ctx context.Context, viewerUserID string, payload SetSubscriptionsPayload, subs *subscriptions, sink FanoutSink) (SubscribeAckPayload, SubscriptionsSnapshotPayload) {
	ack := SubscribeAckPayload{ClientRequestID: payload.ClientRequestID, Operation: payload.Operation, Code: 0, Accepted: []SubscriptionItem{}}
	// allowedByType 汇总同类型多 item 的通过 ID（订阅项可按 type 分多条提交）
	allowedByType := map[string][]string{}
	for _, item := range payload.Items {
		allowed := make([]string, 0, len(item.IDs))
		for _, id := range item.IDs {
			if payload.Operation == "unsubscribe" || h.isSubscriptionAllowed(ctx, viewerUserID, item.Type, id) {
				allowed = append(allowed, id)
				continue
			}
			ack.FailedIDs = append(ack.FailedIDs, FailedSubscription{Type: item.Type, SubscriptionID: id, Reason: "denied"})
		}
		if len(allowed) > 0 {
			ack.Accepted = append(ack.Accepted, SubscriptionItem{Type: item.Type, IDs: allowed})
		}
		allowedByType[item.Type] = append(allowedByType[item.Type], allowed...)
	}
	if len(ack.FailedIDs) > 0 {
		ack.Code = 20100
		ack.Message = "订阅权限不足"
	}
	for _, item := range payload.Items {
		subs.apply(payload.Operation, item.Type, allowedByType[item.Type])
	}
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
	// 双向黑名单拦截：Relationship 的 isBlocked 只是 viewer 拉黑了对方，
	// 被对方拉黑的发送者同样不允许 typing（与 send 的 ensureCanSend 口径一致）。
	callerBlocked, targetBlocked, blockErr := h.userRepo.BlockedEitherWay(ctx, viewerUserID, targetUserID)
	if blockErr != nil || callerBlocked || targetBlocked {
		return false
	}
	isFollowing, isFollowedBy, _, err := h.userRepo.Relationship(ctx, viewerUserID, targetUserID)
	if err != nil {
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

// generateResumeToken 生成不可预测的 opaque resume 凭证（spec L155）。crypto/rand
// 保证不可猜测，避免攻击者伪造他人 token 续传他人事件流。
func generateResumeToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 失败时返回空 token：空值不进 LoadState 匹配（恒 miss → 客户端全量
		// 重放），比可预测的全零 token（可被猜测伪造）安全。
		return ""
	}
	return hex.EncodeToString(b)
}
