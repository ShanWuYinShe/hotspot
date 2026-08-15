package realtime

import (
	"context"
	"sync"

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
	publisher      EventPublisher
	resumeStore    ResumeStore
}

func NewHub() *Hub {
	return &Hub{byUser: map[string][]FanoutSink{}, bySession: map[string][]FanoutSink{}, byUserWatchers: map[string][]FanoutSink{}, byGroup: map[string][]FanoutSink{}}
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
		h.removeSinkLocked(h.bySession, sessionIDOf(h.bySession, oldest), oldest)
		h.removeSinkLocked(h.byUser, userID, oldest)
		_ = oldest.Close()
	}
	first := len(h.byUser[userID]) == 0
	h.byUser[userID] = append(h.byUser[userID], sink)
	h.bySession[sessionID] = append(h.bySession[sessionID], sink)
	return first
}

// sessionIDOf 反查 sink 所属的 session key（仅踢旧路径使用，O(n) 可接受）。
func sessionIDOf(m map[string][]FanoutSink, sink FanoutSink) string {
	for key, sinks := range m {
		for _, s := range sinks {
			if s == sink {
				return key
			}
		}
	}
	return ""
}

func (h *Hub) UnregisterUser(userID, sessionID string, sink FanoutSink) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeSinkLocked(h.bySession, sessionID, sink)
	// 订阅表必须同步清理：sink 只在 applyAuthorizedSubscriptions 注册，此前
	// 断连不清 byUserWatchers/byGroup，sink 永久残留（内存单调泄漏 + 之后每次
	// 群播向死连接白付一次写系统调用）。
	h.removeSinkLocked(h.byUserWatchers, "", sink)
	h.removeSinkLocked(h.byGroup, "", sink)
	return h.removeSinkLocked(h.byUser, userID, sink)
}

// removeSinkLocked 从 m[key]（key 为 "" 时扫全表）移除 sink；返回该 key 下
// 是否因此变空（保留「最后一个连接」语义）。调用方必须持有 h.mu。
func (h *Hub) removeSinkLocked(m map[string][]FanoutSink, key string, sink FanoutSink) bool {
	if key != "" {
		return removeSinkFromSlice(m, key, sink)
	}
	for k := range m {
		removeSinkFromSlice(m, k, sink)
	}
	return false
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

func (h *Hub) SetUserSubscriptions(userIDs []string, sink FanoutSink) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for uid, sinks := range h.byUserWatchers {
		out := sinks[:0]
		for _, s := range sinks {
			if s != sink {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			delete(h.byUserWatchers, uid)
		} else {
			h.byUserWatchers[uid] = out
		}
	}
	for _, uid := range userIDs {
		h.byUserWatchers[uid] = append(h.byUserWatchers[uid], sink)
	}
}

func (h *Hub) SetGroupSubscriptions(groupIDs []string, sink FanoutSink) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for gid, sinks := range h.byGroup {
		out := sinks[:0]
		for _, s := range sinks {
			if s != sink {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			delete(h.byGroup, gid)
		} else {
			h.byGroup[gid] = out
		}
	}
	for _, gid := range groupIDs {
		h.byGroup[gid] = append(h.byGroup[gid], sink)
	}
}

func (h *Hub) BroadcastToUser(userID string, event Event) {
	// resume enrich（spec L173）：为事件分配全局单调 seq 并写入跨实例 user 事件流。
	// 这是 resume 续传的唯一收口——BroadcastToUser 同时掌握 userID 与事件投递，
	// 断线期间（无在线 sink）仍能按 user 累积事件。
	h.mu.RLock()
	store := h.resumeStore
	h.mu.RUnlock()
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

func (h *Hub) BroadcastToGroup(groupID string, event Event) {
	// resume enrich（spec L156, L173）：群事件分配全局单调 seq 并写入 per-group 事件流。
	// 断线成员重连时按订阅快照聚合其订阅 group 流回放（handler Replay）；广播侧不做
	// 成员解析（成员查询放热路径会放大写压力，且 Hub 路由层不应依赖业务 repo）。
	h.mu.RLock()
	store := h.resumeStore
	h.mu.RUnlock()
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
		// Cross-instance fan-out: a publish failure means subscribers on other
		// instances silently miss the event. Log it so operators can detect a
		// degraded redis bridge rather than losing events invisibly.
		if err := p.Publish(context.Background(), channel, event); err != nil {
			observability.L().Warn("hub cross-instance publish failed", "channel", channel, "error", err)
		}
	}
}
