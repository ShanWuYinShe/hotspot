package realtime

import (
	"context"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/observability"
)

type FanoutSink interface {
	Send(Event) error
	Close() error
}

// maxConnectionsPerBroadcast caps the pre-allocation in deliverAllLocal to
// prevent a sudden allocation spike when broadcasting to a hub with a very
// large number of connections. BroadcastToAll is rare (system events only),
// so the O(n) iteration is acceptable.
const maxConnectionsPerBroadcast = 1000

// maxConnectionsPerUser caps concurrent WS connections per user (all
// sessions/devices). Beyond the cap the oldest connection is evicted — a
// single authenticated user must not be able to hold unbounded connections
// (memory + broadcast fan-out amplification).
const maxConnectionsPerUser = 20

type Hub struct {
	mu             sync.RWMutex
	byUser         map[string][]FanoutSink
	bySession      map[string][]FanoutSink
	byUserWatchers map[string][]FanoutSink
	byGroup        map[string][]FanoutSink
	// subsBySink 反向索引（审计 perf P3-5）：每个 sink 登记过的订阅 keys，
	// Set*Subscriptions 全量重建与 UnregisterUser 清理时按记录精确删除，
	// 取代此前对 byUserWatchers/byGroup 的全表扫描。
	subsBySink map[FanoutSink]*subEntry
	// sessionBySink 反向索引（审计 perf P1-8）：sink → 注册时的 sessionID。
	// 踢最旧连接时 O(1) 定位其 session，取代对 bySession 的全表扫描——
	// 该扫描持写锁，大在线量下会阻塞所有广播/注册/注销（全局停顿）。
	sessionBySink map[FanoutSink]string
	publisher     EventPublisher
	resumeStore   ResumeStore
}

// recordPublisher 是 ResumeStore 的可选扩展：单次 Redis 往返完成「seq 分配 +
// Stream 写入 + 跨实例 PUBLISH」（审计 perf P1-4，把广播热路径的两次串行往返
// 合并为一次）。RedisResumeStore 实现之；NopResumeStore 与测试桩不必实现——
// Hub 经类型断言探测，未实现时走既有两步路径（Record → publish）。
// published=false 表示脚本失败，Hub 回退独立 publish（降级语义不变）。
type recordPublisher interface {
	RecordUserAndPublish(userID, channel string, event Event) (seq int64, published bool)
	RecordGroupAndPublish(groupID, channel string, event Event) (seq int64, published bool)
}

// subEntry 记录单个 sink 在订阅表中的 keys 集合（所有权归 Hub：创建/更新/
// 销毁都发生在 Hub 持锁的方法内）。
type subEntry struct {
	userKeys  map[string]struct{}
	groupKeys map[string]struct{}
}

func NewHub() *Hub {
	return &Hub{byUser: map[string][]FanoutSink{}, bySession: map[string][]FanoutSink{}, byUserWatchers: map[string][]FanoutSink{}, byGroup: map[string][]FanoutSink{}, subsBySink: map[FanoutSink]*subEntry{}, sessionBySink: map[FanoutSink]string{}}
}

func (h *Hub) SetPublisher(p EventPublisher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.publisher = p
}

// SetResumeStore 注入 resume 增量恢复存储（spec L156, L159）。nil 时 BroadcastToUser
// 不分配 seq（降级为客户端全量重放，spec L161）。由 bootstrap 在 Redis 可用时注入
// RedisResumeStore；多实例共享同一 Redis，跨实例 resume 一致。
func (h *Hub) SetResumeStore(store ResumeStore) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resumeStore = store
}

func (h *Hub) RegisterUser(userID, sessionID string, sink FanoutSink) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.byUser[userID]) >= maxConnectionsPerUser {
		// 踢最旧连接：新连接几乎总是活跃方；被踢连接由其读循环的 Close 报错退出。
		oldest := h.byUser[userID][0]
		if evictedSession, ok := h.sessionBySink[oldest]; ok {
			h.removeSinkFromSliceAt(h.bySession, evictedSession, oldest)
			delete(h.sessionBySink, oldest)
		}
		// 订阅反向索引同步清理（与 UnregisterUser 同口径）：等读循环退出再清，
		// 窗口期内群播会向死 sink 白付一次写系统调用。
		if entry, ok := h.subsBySink[oldest]; ok {
			for key := range entry.userKeys {
				removeSinkFromSlice(h.byUserWatchers, key, oldest)
			}
			for key := range entry.groupKeys {
				removeSinkFromSlice(h.byGroup, key, oldest)
			}
			delete(h.subsBySink, oldest)
		}
		h.removeSinkFromSliceAt(h.byUser, userID, oldest)
		_ = oldest.Close()
	}
	first := len(h.byUser[userID]) == 0
	h.byUser[userID] = append(h.byUser[userID], sink)
	h.bySession[sessionID] = append(h.bySession[sessionID], sink)
	h.sessionBySink[sink] = sessionID
	return first
}

func (h *Hub) UnregisterUser(userID, sessionID string, sink FanoutSink) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeSinkFromSliceAt(h.bySession, sessionID, sink)
	delete(h.sessionBySink, sink)
	// 订阅表同步清理：按反向索引记录的 keys 精确删除（O(k)），不再全表扫描。
	// sink 只在 Set*Subscriptions 注册；断连时若订阅表残留，之后每次群播都会
	// 向死连接白付一次写系统调用。
	if entry, ok := h.subsBySink[sink]; ok {
		for key := range entry.userKeys {
			removeSinkFromSlice(h.byUserWatchers, key, sink)
		}
		for key := range entry.groupKeys {
			removeSinkFromSlice(h.byGroup, key, sink)
		}
		delete(h.subsBySink, sink)
	}
	return h.removeSinkFromSliceAt(h.byUser, userID, sink)
}

// removeSinkFromSliceAt 从 m[key] 移除 sink；返回该 key 下是否因此变空
// （保留「最后一个连接」语义）。调用方必须持有 h.mu。
func (h *Hub) removeSinkFromSliceAt(m map[string][]FanoutSink, key string, sink FanoutSink) bool {
	return removeSinkFromSlice(m, key, sink)
}

// removeSinkFromSlice 原地过滤 m[key] 中的 sink，清空时删除 key。
func removeSinkFromSlice(m map[string][]FanoutSink, key string, sink FanoutSink) bool {
	sinks := m[key]
	out := sinks[:0]
	for _, s := range sinks {
		if s != sink {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		delete(m, key)
		return true
	}
	m[key] = out
	return false
}

func (h *Hub) CloseSession(sessionID string, event Event) {
	h.closeSessionLocal(sessionID, event)
	h.publish("session:"+sessionID, event)
}

// DrainConnections 向全部在线连接发送 1001（Going Away）并等待其读循环退出，
// 优雅停机专用。http.Server.Shutdown 不会等待 hijacked 的 WS 连接：若跳过排空
// 直接关 DB 池，存活连接的后续查询全部报「池已关闭」，客户端也要等 ping 超时
// 才发现断线。带界轮询：单连接 close 写有超时（socketSink.closeWithCode），
// 不配合的客户端最多拖到 timeout。返回仍在注册表中的残留连接数（诊断用）。
func (h *Hub) DrainConnections(timeout time.Duration) int {
	deadline := time.Now().Add(timeout)
	for {
		h.mu.RLock()
		if len(h.sessionBySink) == 0 {
			h.mu.RUnlock()
			return 0
		}
		sinks := make([]FanoutSink, 0, len(h.sessionBySink))
		for sink := range h.sessionBySink {
			sinks = append(sinks, sink)
		}
		h.mu.RUnlock()
		for _, sink := range sinks {
			if ws, ok := sink.(interface {
				closeWithCode(code int, reason string) error
			}); ok {
				_ = ws.closeWithCode(1001, "server shutting down")
				continue
			}
			_ = sink.Close()
		}
		if time.Now().After(deadline) {
			h.mu.RLock()
			defer h.mu.RUnlock()
			return len(h.sessionBySink)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (h *Hub) closeSessionLocal(sessionID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.bySession[sessionID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
		// SessionRevoked（登出/改密/撤销会话）按 spec 以 close code 4001 断开，
		// 客户端据此刷新 token 或重新登录后重连（api_spec_websocket.md「Close Code 规范」）。
		if ws, ok := sink.(interface {
			closeWithCode(code int, reason string) error
		}); ok {
			_ = ws.closeWithCode(4001, "session revoked")
			continue
		}
		_ = sink.Close()
	}
}

// entryForSinkLocked 返回 sink 的反向索引条目，缺失时创建。
func (h *Hub) entryForSinkLocked(sink FanoutSink) *subEntry {
	entry, ok := h.subsBySink[sink]
	if !ok {
		entry = &subEntry{userKeys: map[string]struct{}{}, groupKeys: map[string]struct{}{}}
		h.subsBySink[sink] = entry
	}
	return entry
}

func (h *Hub) SetUserSubscriptions(userIDs []string, sink FanoutSink) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := h.entryForSinkLocked(sink)
	// 全量重建语义：先按反向索引删掉该 sink 的旧 user keys（O(旧 keys)），
	// 不再全表扫描（审计 perf P3-5）。
	for key := range entry.userKeys {
		removeSinkFromSlice(h.byUserWatchers, key, sink)
	}
	entry.userKeys = make(map[string]struct{}, len(userIDs))
	for _, uid := range userIDs {
		h.byUserWatchers[uid] = append(h.byUserWatchers[uid], sink)
		entry.userKeys[uid] = struct{}{}
	}
}

func (h *Hub) SetGroupSubscriptions(groupIDs []string, sink FanoutSink) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := h.entryForSinkLocked(sink)
	for key := range entry.groupKeys {
		removeSinkFromSlice(h.byGroup, key, sink)
	}
	entry.groupKeys = make(map[string]struct{}, len(groupIDs))
	for _, gid := range groupIDs {
		h.byGroup[gid] = append(h.byGroup[gid], sink)
		entry.groupKeys[gid] = struct{}{}
	}
}

func (h *Hub) BroadcastToUser(userID string, event Event) {
	// resume enrich（spec L173）：为事件分配全局单调 seq 并写入跨实例 user 事件流。
	// 这是 resume 续传的唯一收口——BroadcastToUser 同时掌握 userID 与事件投递，
	// 断线期间（无在线 sink）仍能按 user 累积事件。
	h.mu.RLock()
	store := h.resumeStore
	h.mu.RUnlock()
	// 组合路径（审计 perf P1-4）：seq 分配 + Stream 写入 + 跨实例 PUBLISH 在
	// 一次 Redis 往返内完成。组合已含发布时跳过独立 publish；脚本失败时回退
	// 独立 publish（此时 seq 为 0，与既有 record 失败降级语义一致）。
	if rp, ok := store.(recordPublisher); ok {
		seq, published := rp.RecordUserAndPublish(userID, "user:"+userID, event)
		if seq > 0 {
			event.ServerSeq = seq
		}
		h.deliverUserLocal(userID, event)
		if !published {
			h.publish("user:"+userID, event)
		}
		return
	}
	if store != nil {
		if seq := store.Record(userID, event); seq > 0 {
			event.ServerSeq = seq
		}
	}
	h.deliverUserLocal(userID, event)
	h.publish("user:"+userID, event)
}

// BroadcastToUserNoRecord 投递给该用户的全部本地/跨实例连接但不写 resume 流，
// 用于 typing 等瞬态事件（断线期间无重放价值，写入只会稀释 seq 空间并膨胀
// Stream，重连回放陈旧 typing 也是错误 UX——见 BroadcastToUserSubscribers 注释）。
func (h *Hub) BroadcastToUserNoRecord(userID string, event Event) {
	h.deliverUserLocal(userID, event)
	h.publish("user:"+userID, event)
}

func (h *Hub) deliverUserLocal(userID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byUser[userID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

func (h *Hub) BroadcastToUserSubscribers(userID string, event Event) {
	// presence/typing 类瞬态事件不写 resume 流：断线期间的状态无重放价值（重连后
	// 通过订阅重建 + presence 快照恢复），写入反而让 60s 窗口内的 Stream 无谓膨胀。
	h.deliverUserSubscribersLocal(userID, event)
	h.publish("usersub:"+userID, event)
}

func (h *Hub) deliverUserSubscribersLocal(userID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byUserWatchers[userID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

// subscriptionRevoker 由连接层实现：服务端强制摘除订阅时（踢出/退群/群解散），
// 经 sink 回调同步清理该连接的订阅快照——否则下一次 SetSubscriptions 全量重建
// 会把已撤销的订阅加回 byGroup（applyAuthorizedSubscriptions 以连接侧快照为
// 重建依据），撤销形同虚设。
type subscriptionRevoker interface {
	revokeSubscription(kind, id string)
}

// RevokeGroupSubscription 强制摘除某用户全部连接对指定群的订阅（成员被移出/
// 封禁/退群时由 groups 模块调用）。群消息广播按 byGroup 直投全文，而订阅鉴权
// 只发生在 SetSubscriptions 时刻——成员资格变更后不摘除，被移出者保持连接即可
// 继续实时收到群内消息。跨实例经 revoke: 控制通道同步（对端只清注册表，不下发
// 任何客户端事件：spec 未定义订阅撤销事件，客户端经既有 GroupMemberRemoved 等
// 事件感知成员资格变更）。
func (h *Hub) RevokeGroupSubscription(userID, groupID string) {
	h.revokeGroupLocal(userID, groupID)
	h.publish(redisChannelRevoke+userID, Event{Type: EventTypeSubscriptionRevoked, Payload: SubscriptionRevokedPayload{GroupID: groupID}})
}

// revokeGroupLocal 摘除本实例内该用户全部连接的群订阅，并经 sink 回调同步连接层
// 订阅快照。回调在锁外调用：其内部触碰连接侧 subscriptions（自有锁），与 h.mu
// 无交集，避免锁序耦合。
func (h *Hub) revokeGroupLocal(userID, groupID string) {
	h.mu.Lock()
	affected := make([]FanoutSink, 0, 1)
	for _, sink := range h.byUser[userID] {
		if entry, ok := h.subsBySink[sink]; ok {
			if _, subscribed := entry.groupKeys[groupID]; subscribed {
				delete(entry.groupKeys, groupID)
				affected = append(affected, sink)
			}
		}
	}
	if len(affected) > 0 {
		byGroup := h.byGroup[groupID]
		out := byGroup[:0]
		for _, sink := range byGroup {
			revoked := false
			for _, a := range affected {
				if sink == a {
					revoked = true
					break
				}
			}
			if !revoked {
				out = append(out, sink)
			}
		}
		if len(out) == 0 {
			delete(h.byGroup, groupID)
		} else {
			h.byGroup[groupID] = out
		}
	}
	h.mu.Unlock()
	for _, sink := range affected {
		if revoker, ok := sink.(subscriptionRevoker); ok {
			revoker.revokeSubscription("group", groupID)
		}
	}
}

// DropGroup 群解散后摘除全部订阅者对该群的订阅（解散影响全组成员，逐成员
// RevokeGroupSubscription 需要遍历成员表，直接按群 key 清理一次到位）。
func (h *Hub) DropGroup(groupID string) {
	h.dropGroupLocal(groupID)
	h.publish(redisChannelGroupDrop+groupID, Event{Type: EventTypeSubscriptionRevoked, Payload: SubscriptionRevokedPayload{GroupID: groupID}})
}

func (h *Hub) dropGroupLocal(groupID string) {
	h.mu.Lock()
	sinks := h.byGroup[groupID]
	delete(h.byGroup, groupID)
	for _, sink := range sinks {
		if entry, ok := h.subsBySink[sink]; ok {
			delete(entry.groupKeys, groupID)
		}
	}
	h.mu.Unlock()
	for _, sink := range sinks {
		if revoker, ok := sink.(subscriptionRevoker); ok {
			revoker.revokeSubscription("group", groupID)
		}
	}
}

func (h *Hub) BroadcastToGroup(groupID string, event Event) {
	// resume enrich（spec L156, L173）：群事件分配全局单调 seq 并写入 per-group 事件流。
	// 断线成员重连时按订阅快照聚合其订阅 group 流回放（handler Replay）；广播侧不做
	// 成员解析（成员查询放热路径会放大写压力，且 Hub 路由层不应依赖业务 repo）。
	// 组合路径语义与 BroadcastToGroupUser 对应的 user 版相同（审计 perf P1-4，
	// 单次 Redis 往返；失败回退独立 publish）。
	h.mu.RLock()
	store := h.resumeStore
	h.mu.RUnlock()
	if rp, ok := store.(recordPublisher); ok {
		seq, published := rp.RecordGroupAndPublish(groupID, "group:"+groupID, event)
		if seq > 0 {
			event.ServerSeq = seq
		}
		h.deliverGroupLocal(groupID, event)
		if !published {
			h.publish("group:"+groupID, event)
		}
		return
	}
	if store != nil {
		if seq := store.RecordGroup(groupID, event); seq > 0 {
			event.ServerSeq = seq
		}
	}
	h.deliverGroupLocal(groupID, event)
	h.publish("group:"+groupID, event)
}

// BroadcastToGroupNoRecord 投递给群订阅者但不写 resume 流（typing 等瞬态事件，
// 语义同 BroadcastToUserNoRecord）。
func (h *Hub) BroadcastToGroupNoRecord(groupID string, event Event) {
	h.deliverGroupLocal(groupID, event)
	h.publish("group:"+groupID, event)
}

func (h *Hub) deliverGroupLocal(groupID string, event Event) {
	h.mu.RLock()
	sinks := append([]FanoutSink(nil), h.byGroup[groupID]...)
	h.mu.RUnlock()
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

// BroadcastToAll sends event to every connected session. It is O(n) for all
// connections — it collects a snapshot of all sinks under RLock then iterates
// outside the lock. This is used for rare system-wide events only; it should
// not be called on hot paths.
func (h *Hub) BroadcastToAll(event Event) {
	h.deliverAllLocal(event)
	h.publish("all", event)
}

// deliverAllLocal collects a snapshot of every connected sink under RLock and
// then delivers to each outside the lock. The number of sinks actually
// delivered is capped at maxConnectionsPerBroadcast (drop-tail): beyond the
// cap, additional sessions are silently skipped and a single warning is
// logged per call so operators can detect fan-out storms. This is O(n) for
// all active connections.
func (h *Hub) deliverAllLocal(event Event) {
	h.mu.RLock()
	total := 0
	for _, sessionSinks := range h.bySession {
		total += len(sessionSinks)
	}
	sinks := make([]FanoutSink, 0, min(total, maxConnectionsPerBroadcast))
	for _, sessionSinks := range h.bySession {
		sinks = append(sinks, sessionSinks...)
		if len(sinks) >= maxConnectionsPerBroadcast {
			break
		}
	}
	h.mu.RUnlock()
	if total > maxConnectionsPerBroadcast {
		observability.L().Warn(
			"broadcast fanout exceeded cap; tail sessions dropped",
			"total", total,
			"cap", maxConnectionsPerBroadcast,
			"delivered", len(sinks),
		)
	}
	for _, sink := range sinks {
		_ = sink.Send(event)
	}
}

func (h *Hub) publish(channel string, event Event) {
	h.mu.RLock()
	p := h.publisher
	h.mu.RUnlock()
	if p != nil {
		// 1s 硬 deadline（审计 perf P2-5）：publish 在广播热路径上，Redis 卡死
		// 时不能继承无界 ctx 拖住广播方；超时按发布失败降级处理。不用请求 ctx：
		// 广播常由后台任务触发，请求 ctx 可能早已取消。
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// Cross-instance fan-out: a publish failure means subscribers on other
		// instances silently miss the event. Log it so operators can detect a
		// degraded redis bridge rather than losing events invisibly.
		if err := p.Publish(ctx, channel, event); err != nil {
			observability.L().Warn("hub cross-instance publish failed", "channel", channel, "error", err)
		}
	}
}
