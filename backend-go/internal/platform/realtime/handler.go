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

// PresenceSetting 是订阅鉴权批量路径的目标用户 presence 摘要（bootstrap 适配层
// 从 users.UserSettings 投影，避免 realtime 依赖完整 settings 类型）。
type PresenceSetting struct {
	PresenceVisible bool
	Found           bool
}

// settingsBatchReader 能力探测接口（可选）：支持批量取目标用户 settings 时，
// 订阅鉴权走批量路径——单条订阅消息最坏 500 个 user ID，逐条 GetSettings 会把
// 连接读循环阻塞成串行 DB 往返（Ping 无法回应 → 客户端超时断连 → 重连风暴）。
// 未实现时回退逐条（内存 repo/测试）。
type settingsBatchReader interface {
	GetSettingsBatch(ctx context.Context, userIDs []string) (map[string]PresenceSetting, error)
}

// membershipBatchReader 能力探测接口（可选）：批量判定 viewer 是否为各群成员，
// 与 settingsBatchReader 同动机。未实现时回退逐条 IsMember。
type membershipBatchReader interface {
	AreMembers(ctx context.Context, groupIDs []string, userID string) (map[string]bool, error)
}

// banAuthorizer reports whether a user is currently banned platform-wide.
// Implemented by the admin module's repository via bootstrap wiring. Without
// it the realtime channel would accept connections from banned accounts —
// the HTTP-side ban middleware never runs on the WS handshake path.
type banAuthorizer interface {
	IsBanned(context.Context, string) (bool, error)
}

// BanDetails 携带封禁详情（spec api_spec_websocket.md AccountBannedPayload）。
type BanDetails struct {
	Reason       string
	DurationDays int
	BannedAt     time.Time
	BannedUntil  *time.Time // nil = 永久封禁
}

// banDetailProvider 可选提供封禁详情用于组装 AccountBanned payload；bootstrap
// 侧由 admin repo（GetUserBanState）适配。未接线时以零值详情降级（reason 兜底
// 文案，banned_until=null 表示未知/永久）。
type banDetailProvider interface {
	BanDetails(ctx context.Context, userID string) (BanDetails, bool, error)
}

// presenceWriter persists runtime presence transitions to the users table
// (online on the first connection of a user, offline when the last one
// closes). Implemented by the users module repository via bootstrap wiring;
// nil (dev/test default) skips the DB write — the UserOnline/UserOffline
// broadcast events below are unaffected either way.
type presenceWriter interface {
	SetPresence(ctx context.Context, userID string, online bool) error
}

type Handler struct {
	auth           *platformauth.Service
	userRepo       userRelationAuthorizer
	groupRepo      groupMembershipAuthorizer
	bans           banAuthorizer
	presence       presenceWriter
	actorProvider  typingActorProvider
	typingThrottle sync.Map
	actorCache     sync.Map // userID → actorCacheEntry（typing actor 短缓存，审计 perf P2-2）
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
func NewHandler(auth *platformauth.Service, userRepo userRelationAuthorizer, groupRepo groupMembershipAuthorizer) *Handler {
	return NewHandlerWithContext(context.Background(), auth, userRepo, groupRepo)
}

// NewHandlerWithContext is the context-aware constructor. Prefer this from
// bootstrap so the sweeper is tied to the application lifetime.
func NewHandlerWithContext(ctx context.Context, auth *platformauth.Service, userRepo userRelationAuthorizer, groupRepo groupMembershipAuthorizer) *Handler {
	sweepCtx, cancel := context.WithCancel(ctx)
	h := &Handler{auth: auth, userRepo: userRepo, groupRepo: groupRepo, stop: cancel}
	h.wg.Add(1)
	// 注意用 worker.Go 而非 GoRestartable：sweepTypingThrottle 自带 defer
	// h.wg.Done() 与 Close 的 wg.Wait() 配对，panic 重启会造成二次 Done
	// （负计数 panic）并破坏 Wait 语义。纯内存 throttle 清理无重启价值。
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

// banPayload 组装 AccountBanned 事件负载：优先取封禁详情（原因/时长/起止），
// 详情未接线时以兜底文案降级（banned_until=null 表示永久或未知）。
func (h *Handler) banPayload(ctx context.Context, userID string) AccountBannedPayload {
	if dp, ok := h.bans.(banDetailProvider); ok {
		if detail, found, err := dp.BanDetails(ctx, userID); err == nil && found {
			payload := AccountBannedPayload{
				Reason:       detail.Reason,
				DurationDays: detail.DurationDays,
			}
			if !detail.BannedAt.IsZero() {
				payload.BannedAt = detail.BannedAt.UTC().Format(time.RFC3339)
			} else {
				payload.BannedAt = time.Now().UTC().Format(time.RFC3339)
			}
			if detail.BannedUntil != nil {
				until := detail.BannedUntil.UTC().Format(time.RFC3339)
				payload.BannedUntil = &until
			}
			return payload
		}
	}
	return AccountBannedPayload{Reason: "账号已被封禁", BannedAt: time.Now().UTC().Format(time.RFC3339)}
}

// SetPresenceWriter wires the runtime presence persister (users table).
// Optional (nil = no DB write, dev/test default); write failures only log and
// never interrupt the connection lifecycle.
func (h *Handler) SetPresenceWriter(p presenceWriter) { h.presence = p }

// SetResumeStore 注入 resume 增量恢复存储（spec L156, L159）。nil 时退化为 NopResumeStore
// 语义：token 仍下发但 LoadState 恒空，客户端回退全量重放（spec L161 降级保证）。
func (h *Handler) SetResumeStore(s ResumeStore) { h.resumeStore = s }

// markPresence 落库 presence 变更。断连 defer 执行时请求 ctx（r.Context()）
// 已被取消，故统一用独立超时 ctx；写失败仅记日志，不影响连接生命周期。
func (h *Handler) markPresence(userID string, online bool) {
	if h.presence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.presence.SetPresence(ctx, userID, online); err != nil {
		observability.L().Warn("ws: presence db write failed", "user_id", userID, "online", online, "error", err)
	}
}

// typingActorCacheTTL 是 typing actor 信息的进程内缓存窗口（审计 perf P2-2
// 收口）：节流前置后每个 (user,target) 对最多每 3s 打一次 GetActorInfo DB 查询，
// 群聊内多对并发时仍是可观的热路径 DB 压力。昵称/头像变更延迟 ≤TTL 对 typing
// 指示无感知影响。
const typingActorCacheTTL = 30 * time.Second

type actorCacheEntry struct {
	actor map[string]any
	at    time.Time
}

// resolveTypingActor builds the actor UserInfo map for the typing event.
// 进程内短缓存（30s）：同 user 的连续 typing 帧不再逐帧打 GetActorInfo。
// DB 错误不缓存（瞬态故障下一帧重试）；「用户不存在」的兜底结果照常缓存
// （稳定状态）。
func (h *Handler) resolveTypingActor(ctx context.Context, userID string) map[string]any {
	if cached, ok := h.actorCache.Load(userID); ok {
		if entry, ok := cached.(actorCacheEntry); ok && time.Since(entry.at) < typingActorCacheTTL {
			return entry.actor
		}
	}
	actor, cacheable := h.lookupTypingActor(ctx, userID)
	if cacheable {
		h.actorCache.Store(userID, actorCacheEntry{actor: actor, at: time.Now()})
	}
	return actor
}

// lookupTypingActor 查询 typing actor；cacheable=false 表示瞬态故障（不缓存，
// 下一帧重试），true 表示稳定结果（含「用户不存在」兜底，可缓存）。
func (h *Handler) lookupTypingActor(ctx context.Context, userID string) (actor map[string]any, cacheable bool) {
	if h.actorProvider != nil {
		info, err := h.actorProvider.GetActorInfo(ctx, userID)
		if err != nil {
			return map[string]any{"user_id": userID, "nickname": userID}, false
		}
		actor = map[string]any{"user_id": info.UserID, "nickname": info.Nickname}
		if info.AvatarURL != "" {
			actor["avatar_url"] = info.AvatarURL
		}
		return actor, true
	}
	return map[string]any{"user_id": userID, "nickname": userID}, true
}

// sweepTypingThrottle periodically drops stale typing-throttle entries so the
// sync.Map doesn't grow unboundedly with each unique (user,target) pair over
// the process lifetime. The throttle window is 3s; entries older than a minute
// are dead. It exits when ctx is cancelled. actorCache（TTL 30s）同周期清扫：
// 超过 2 分钟未刷新的条目已必然过期。
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
			actorCutoff := time.Now().Add(-2 * typingActorCacheTTL)
			h.actorCache.Range(func(key, value any) bool {
				if entry, ok := value.(actorCacheEntry); ok && entry.at.Before(actorCutoff) {
					h.actorCache.Delete(key)
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
// EnableCompression 开启 permessage-deflate（审计 perf P3-2，gorilla v1.4+）：
// WS 帧以 JSON 为主、重复字段多，压缩显著省带宽；代价是每连接常驻一份压缩
// 上下文（客户端协商 context takeover 时），大连接量下以内存换流量。
var upgrader = websocket.Upgrader{CheckOrigin: sameOrigin, EnableCompression: true}

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
	// fail-open 与 RequiredAuth 同口径：ban checker 瞬时故障时不锁死全部连接，
	// 但故障必须记日志（P2-3：安全降级不可静默）。
	if h.bans != nil {
		banned, err := h.bans.IsBanned(ctx, principal.UserID)
		if err != nil {
			observability.L().Warn("ws: ban check failed, failing open",
				"user_id", principal.UserID, "error", err)
		} else if banned {
			observability.L().Warn("ws: auth rejected, account banned", "user_id", principal.UserID)
			// spec：推送 AccountBanned 事件后以 close 4001 断开（客户端据此
			// 停止自动重连并展示封禁页；登录将得到 10201）。
			_ = conn.WriteJSON(Event{Type: EventTypeAccountBanned, Payload: h.banPayload(ctx, principal.UserID)})
			sendCloseFrame(conn, 4001, "account banned")
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
	sink := newSocketSink(conn)
	// 服务端强制摘除订阅（踢出/封禁/退群/群解散）时同步连接侧快照：快照是
	// SetSubscriptions 全量重建 Hub 注册表的依据，不清会把已撤销订阅加回去。
	// 必须在 RegisterUser 之前赋值（此后经 Hub 读，无其他写入者）。
	sink.onSubscriptionRevoked = func(kind, id string) {
		subs.apply("unsubscribe", kind, []string{id})
	}
	if DefaultHub().RegisterUser(principal.UserID, principal.SessionID, sink) {
		// 首条连接：presence 落库 online（groups/search 的 presence_status 统计依据）。
		h.markPresence(principal.UserID, true)
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
			// 最后一条连接断开：presence 落库 offline（多连接共存时中间断开不写，
			// 由 Hub 的「连接数归零」判定保证语义）。
			h.markPresence(principal.UserID, false)
			DefaultHub().BroadcastToUserSubscribers(principal.UserID, Event{Type: EventTypeUserOffline, Payload: UserOfflinePayload{UserID: principal.UserID, OfflineAt: time.Now().UTC().Format(time.RFC3339)}})
		}
		// 停掉该连接的 writer goroutine（幂等；conn.Close 另有外层 defer 兜底）。
		_ = sink.Close()
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
	payload.Config.MaxSubscriptions.User = maxUserSubscriptionsPerConnection
	payload.Config.MaxSubscriptions.Group = maxGroupSubscriptionsPerConnection
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
			// 跨 item 的总量上限在 applyAuthorizedSubscriptions 汇总侧兜底。
			failed := []FailedSubscription{}
			items := make([]SubscriptionItem, 0, len(payload.Items))
			for _, item := range payload.Items {
				limit, ok := subscriptionTypeLimit(item.Type)
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
			// 节流前置（审计 perf P2-2）：窗口内帧直接丢弃、不再打 DB 鉴权。
			// 语义不变——节流时间戳只在通过 DB 校验后更新，故 3s 窗口内的首帧
			// 仍会走完整 DB 校验，被节流帧与鉴权失败帧一样静默（不广播、不回错）。
			typingKey := fmt.Sprintf("typing:%s:%s", principal.UserID, payload.TargetUserID)
			if last, ok := h.typingThrottle.Load(typingKey); ok {
				if t, ok := last.(time.Time); ok && time.Since(t) < 3*time.Second {
					continue
				}
			}
			if !h.canSendPrivateTyping(principalCtx, principal.UserID, payload.TargetUserID) {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSDenied, Message: "无权限", Severity: "warning", Retryable: false}})
				continue
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
			// 节流前置（审计 perf P2-2），语义同 SendPrivateTyping。
			typingKey := fmt.Sprintf("typing:%s:%s", principal.UserID, payload.GroupID)
			if last, ok := h.typingThrottle.Load(typingKey); ok {
				if t, ok := last.(time.Time); ok && time.Since(t) < 3*time.Second {
					continue
				}
			}
			if !h.isGroupMember(principalCtx, principal.UserID, payload.GroupID) {
				_ = sink.Send(Event{Type: EventTypeError, Payload: ErrorPayload{Code: CodeWSDenied, Message: "无权限", Severity: "warning", Retryable: false}})
				continue
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
	// allowedByType 汇总同类型多 item 的通过 ID（订阅项可按 type 分多条提交）。
	// 鉴权结果按 type 预计算（批量优先）：user 走 settingsBatchReader 一次往返、
	// group 走 membershipBatchReader 一次往返；能力未接线或批量查询失败时回退
	// 逐条（语义不变，仅慢）。
	presenceAllowed := h.presenceAllowedBatch(ctx, viewerUserID, payload)
	groupAllowed := h.groupAllowedBatch(ctx, viewerUserID, payload)
	// 任一类型批量不可用即整体回退逐条（两种类型并存且其一失败时，语义仍与
	// 旧实现完全一致）。
	batchMode := payload.Operation != "unsubscribe" && presenceAllowed != nil && groupAllowed != nil
	allowedByType := map[string][]string{}
	seenByType := map[string]map[string]bool{}
	for _, item := range payload.Items {
		allowed := make([]string, 0, len(item.IDs))
		for _, id := range item.IDs {
			permitted := payload.Operation == "unsubscribe"
			if !permitted {
				if batchMode {
					permitted = h.subscriptionAllowed(item.Type, id, presenceAllowed, groupAllowed)
				} else {
					permitted = h.isSubscriptionAllowed(ctx, viewerUserID, item.Type, id)
				}
			}
			if permitted {
				allowed = append(allowed, id)
				continue
			}
			ack.FailedIDs = append(ack.FailedIDs, FailedSubscription{Type: item.Type, SubscriptionID: id, Reason: "denied"})
		}
		if len(allowed) > 0 {
			ack.Accepted = append(ack.Accepted, SubscriptionItem{Type: item.Type, IDs: allowed})
		}
		seen := seenByType[item.Type]
		if seen == nil {
			seen = make(map[string]bool, len(allowed))
			seenByType[item.Type] = seen
		}
		for _, id := range allowed {
			if !seen[id] {
				seen[id] = true
				allowedByType[item.Type] = append(allowedByType[item.Type], id)
			}
		}
	}
	// 单条订阅消息可携带多个同类型 item，逐 item 裁剪后累加仍会突破连接级
	// 上限（500 user / 100 group）——这里按类型做总量截断，超出部分计入
	// failed_ids；resume 恢复路径每类型只提交一个 item，天然受同一约束。
	truncated := false
	if payload.Operation != "unsubscribe" {
		for kind, allowed := range allowedByType {
			limit, ok := subscriptionTypeLimit(kind)
			if !ok || len(allowed) <= limit {
				continue
			}
			for _, id := range allowed[limit:] {
				ack.FailedIDs = append(ack.FailedIDs, FailedSubscription{Type: kind, SubscriptionID: id, Reason: "limit_exceeded"})
				delete(seenByType[kind], id)
			}
			allowedByType[kind] = allowed[:limit]
			truncated = true
		}
	}
	if truncated {
		// Accepted 必须反映实际生效的订阅，总量截断后按剩余集合重建。
		ack.Accepted = ack.Accepted[:0]
		for _, item := range payload.Items {
			seen := seenByType[item.Type]
			allowed := make([]string, 0, len(item.IDs))
			for _, id := range item.IDs {
				if seen[id] {
					allowed = append(allowed, id)
				}
			}
			if len(allowed) > 0 {
				ack.Accepted = append(ack.Accepted, SubscriptionItem{Type: item.Type, IDs: allowed})
			}
		}
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

// presenceAllowedBatch 预计算 payload 中全部 user 订阅的可见性判定；nil 表示
// 该类型无批量结果（逐条回退）。判定语义与 canSubscribeUser 逐条版一致：
// self 恒通过、目标无 settings 行（Found=false）恒通过、其余按 presence_visible。
func (h *Handler) presenceAllowedBatch(ctx context.Context, viewerUserID string, payload SetSubscriptionsPayload) map[string]bool {
	batch, ok := h.userRepo.(settingsBatchReader)
	if !ok {
		return nil
	}
	ids := collectSubscriptionIDs(payload, "user")
	if len(ids) == 0 {
		return map[string]bool{}
	}
	settings, err := batch.GetSettingsBatch(ctx, ids)
	if err != nil {
		return nil
	}
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == viewerUserID {
			allowed[id] = true
			continue
		}
		s, found := settings[id]
		if !found || !s.Found {
			allowed[id] = true
			continue
		}
		allowed[id] = s.PresenceVisible
	}
	return allowed
}

// groupAllowedBatch 预计算 payload 中全部 group 订阅的成员判定；nil 表示回退逐条。
func (h *Handler) groupAllowedBatch(ctx context.Context, viewerUserID string, payload SetSubscriptionsPayload) map[string]bool {
	batch, ok := h.groupRepo.(membershipBatchReader)
	if !ok {
		return nil
	}
	ids := collectSubscriptionIDs(payload, "group")
	if len(ids) == 0 {
		return map[string]bool{}
	}
	members, err := batch.AreMembers(ctx, ids, viewerUserID)
	if err != nil {
		return nil
	}
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		allowed[id] = members[id]
	}
	return allowed
}

func collectSubscriptionIDs(payload SetSubscriptionsPayload, kind string) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0, 16)
	for _, item := range payload.Items {
		if item.Type != kind {
			continue
		}
		for _, id := range item.IDs {
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func (h *Handler) subscriptionAllowed(kind, id string, presenceAllowed, groupAllowed map[string]bool) bool {
	if kind == "user" && presenceAllowed != nil {
		return presenceAllowed[id]
	}
	if kind == "group" && groupAllowed != nil {
		return groupAllowed[id]
	}
	return h.isSubscriptionAllowed(context.Background(), "", kind, id)
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
	// 订阅鉴权只需要 presence 可见性（一次 GetSettings），不走全量 OpenTheDoor
	// 社交上下文（后者单次 15+ 次查询，500 ID 的订阅消息会放大成数千次 DB 往返）。
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
