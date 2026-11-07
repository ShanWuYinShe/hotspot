package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"hotspot/backend-go/internal/platform/observability"
)

// FanoutSink 是广播目标（生产为 socketSink，测试可用轻量桩）。
type FanoutSink interface {
	Send(Event) error
	Close() error
}

// preparedMessageSink 由支持「一次准备、多连接复用」的 sink 实现（socketSink）。
type preparedMessageSink interface {
	SendPrepared(pm *websocket.PreparedMessage, seq int64) error
}

func broadcastSinks(sinks []FanoutSink, event Event) {
	if len(sinks) == 0 {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		observability.L().Warn("realtime: marshal broadcast event failed", "type", event.Type, "error", err)
		return
	}
	pm, err := websocket.NewPreparedMessage(websocket.TextMessage, data)
	if err != nil {
		observability.L().Warn("realtime: prepare broadcast event failed", "type", event.Type, "error", err)
		return
	}
	for _, sink := range sinks {
		if ps, ok := sink.(preparedMessageSink); ok {
			_ = ps.SendPrepared(pm, event.ServerSeq)
			continue
		}
		_ = sink.Send(event)
	}
}

// maxConnectionsPerUser caps concurrent WS connections per user (all
// sessions/devices). Beyond the cap the oldest connection is evicted — a
// single authenticated user must not be able to hold unbounded connections
// (memory + broadcast fan-out amplification).
const maxConnectionsPerUser = 20

// hubShardCount 是注册表分片数：广播/注册/注销按 user/group/session key
// 散列到独立 shard，不再共享一把全局 RWMutex。幂次便于取模。
const hubShardCount = 32

// hubShard 是一组注册表 + 独立读写锁。同一 shard 内的操作仍串行，但不同
// user/group/session 的广播与连接抖动互不阻塞；单 key 的热点仍由该 shard
// 的 RWMutex 保护（RWMutex 本身支持并发读）。
type hubShard struct {
	mu             sync.RWMutex
	byUser         map[string][]FanoutSink
	bySession      map[string][]FanoutSink
	byUserWatchers map[string][]FanoutSink
	byGroup        map[string][]FanoutSink
}

// subEntry 记录单个 sink 在订阅表中的 keys 集合（所有权归 Hub）。entry 自身
// 用互斥锁保护：订阅重建与注销可能并发（连接关闭与 subscription 更新竞争），
// 不能依赖全局 Hub 锁。closed 标记注销后不再接受订阅重建，避免注销清理后
// SetSubscriptions 把死连接重新写回路由表。
type subEntry struct {
	mu        sync.Mutex
	closed    bool
	userKeys  map[string]struct{}
	groupKeys map[string]struct{}
}

type Hub struct {
	shards [hubShardCount]hubShard
	// subsBySink / sessionBySink 是 sink 维度反向索引（sync.Map：注册/注销
	// 并发下无锁读，避免与 shard 锁交叉）。FanoutSink 以接口值作 key，
	// 实现方必须可比较（生产 socketSink 与测试桩均为指针）。
	subsBySink    sync.Map // map[FanoutSink]*subEntry
	sessionBySink sync.Map // map[FanoutSink]string
	// sessionCount 是当前注册 sink 的近似计数，仅用于 snapshot 预分配容量
	//（精确值由 shard 内 map 决定；Load 仅供建 slice 的容量提示）。
	sessionCount atomic.Int64
	// cfgMu 只保护低频写入的 publisher/resumeStore；广播热路径用 RLock。
	cfgMu       sync.RWMutex
	publisher   EventPublisher
	resumeStore ResumeStore
	// rejectedConnections 是全局连接上限触发的拒绝计数（HubCollector 暴露）。
	rejectedConnections atomic.Int64
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

func NewHub() *Hub {
	h := &Hub{}
	for i := range h.shards {
		h.shards[i].byUser = map[string][]FanoutSink{}
		h.shards[i].bySession = map[string][]FanoutSink{}
		h.shards[i].byUserWatchers = map[string][]FanoutSink{}
		h.shards[i].byGroup = map[string][]FanoutSink{}
	}
	return h
}

// shardFor 用 FNV-1a 散列 key；前缀区分 user/session/watch/group 四种命名空间，
// 避免不同类型 key 在分布上相关。
func (h *Hub) shardFor(key string) *hubShard {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	var hash uint32 = offset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= prime32
	}
	return &h.shards[hash%hubShardCount]
}

// NoteConnectionRejected 记录一次因全局连接上限被拒的 WS 连接。
func (h *Hub) NoteConnectionRejected() { h.rejectedConnections.Add(1) }

// RejectedConnections 返回累计拒绝连接数（指标/测试用）。
func (h *Hub) RejectedConnections() int64 { return h.rejectedConnections.Load() }

func (h *Hub) SetPublisher(p EventPublisher) {
	h.cfgMu.Lock()
	defer h.cfgMu.Unlock()
	h.publisher = p
}

func (h *Hub) SetResumeStore(store ResumeStore) {
	h.cfgMu.Lock()
	defer h.cfgMu.Unlock()
	h.resumeStore = store
}

func (h *Hub) getResumeStore() ResumeStore {
	h.cfgMu.RLock()
	defer h.cfgMu.RUnlock()
	return h.resumeStore
}

func (h *Hub) RegisterUser(userID, sessionID string, sink FanoutSink) bool {
	userShard := h.shardFor("user:" + userID)
	var evicted FanoutSink
	var first bool
	userShard.mu.Lock()
	if len(userShard.byUser[userID]) >= maxConnectionsPerUser {
		// 踢最旧连接：新连接几乎总是活跃方；被踢连接由其读循环的 Close 报错退出。
		oldest := userShard.byUser[userID][0]
		evicted = oldest
		removeSinkFromSlice(userShard.byUser, userID, oldest)
	}
	first = len(userShard.byUser[userID]) == 0
	userShard.byUser[userID] = append(userShard.byUser[userID], sink)
	userShard.mu.Unlock()

	sessionShard := h.shardFor("session:" + sessionID)
	sessionShard.mu.Lock()
	sessionShard.bySession[sessionID] = append(sessionShard.bySession[sessionID], sink)
	sessionShard.mu.Unlock()
	h.sessionBySink.Store(sink, sessionID)
	h.sessionCount.Add(1)

	// 逐出连接的清理必须在释放 shard 锁之后：socketSink.Close/closeWithCode
	// 会做网络 Close 并最多等待写超时（5s），持锁关闭会阻塞同 shard 广播。
	if evicted != nil {
		h.dropSink(evicted)
	}
	return first
}

// dropSink 从全部注册表/反向索引移除 sink（逐出或注销共用）。清理逻辑不持
// 任何 shard 写锁跨多个 shard，避免锁顺序问题。
func (h *Hub) dropSink(sink FanoutSink) {
	if sessionID, ok := h.sessionBySink.LoadAndDelete(sink); ok {
		h.sessionCount.Add(-1)
		if id, ok := sessionID.(string); ok {
			sessionShard := h.shardFor("session:" + id)
			sessionShard.mu.Lock()
			removeSinkFromSlice(sessionShard.bySession, id, sink)
			sessionShard.mu.Unlock()
		}
	}
	if entryAny, ok := h.subsBySink.LoadAndDelete(sink); ok {
		if entry, ok := entryAny.(*subEntry); ok {
			entry.mu.Lock()
			entry.closed = true
			userKeys := make([]string, 0, len(entry.userKeys))
			for key := range entry.userKeys {
				userKeys = append(userKeys, key)
			}
			groupKeys := make([]string, 0, len(entry.groupKeys))
			for key := range entry.groupKeys {
				groupKeys = append(groupKeys, key)
			}
			entry.mu.Unlock()
			for _, key := range userKeys {
				shard := h.shardFor("watch:" + key)
				shard.mu.Lock()
				removeSinkFromSlice(shard.byUserWatchers, key, sink)
				shard.mu.Unlock()
			}
			for _, key := range groupKeys {
				shard := h.shardFor("group:" + key)
				shard.mu.Lock()
				removeSinkFromSlice(shard.byGroup, key, sink)
				shard.mu.Unlock()
			}
		}
	}
}

func (h *Hub) UnregisterUser(userID, sessionID string, sink FanoutSink) bool {
	userShard := h.shardFor("user:" + userID)
	userShard.mu.Lock()
	empty := removeSinkFromSlice(userShard.byUser, userID, sink)
	userShard.mu.Unlock()

	sessionShard := h.shardFor("session:" + sessionID)
	sessionShard.mu.Lock()
	removeSinkFromSlice(sessionShard.bySession, sessionID, sink)
	sessionShard.mu.Unlock()
	if _, existed := h.sessionBySink.LoadAndDelete(sink); existed {
		h.sessionCount.Add(-1)
	}
	h.dropSinkSubscriptions(sink)
	return empty
}

// dropSinkSubscriptions 移除 sink 的订阅反向索引；session/user 已由调用方
// 精确清理，这里只处理订阅 key。
func (h *Hub) dropSinkSubscriptions(sink FanoutSink) {
	entryAny, ok := h.subsBySink.LoadAndDelete(sink)
	if !ok {
		return
	}
	entry, ok := entryAny.(*subEntry)
	if !ok {
		return
	}
	entry.mu.Lock()
	entry.closed = true
	userKeys := make([]string, 0, len(entry.userKeys))
	for key := range entry.userKeys {
		userKeys = append(userKeys, key)
	}
	groupKeys := make([]string, 0, len(entry.groupKeys))
	for key := range entry.groupKeys {
		groupKeys = append(groupKeys, key)
	}
	entry.mu.Unlock()
	for _, key := range userKeys {
		shard := h.shardFor("watch:" + key)
		shard.mu.Lock()
		removeSinkFromSlice(shard.byUserWatchers, key, sink)
		shard.mu.Unlock()
	}
	for _, key := range groupKeys {
		shard := h.shardFor("group:" + key)
		shard.mu.Lock()
		removeSinkFromSlice(shard.byGroup, key, sink)
		shard.mu.Unlock()
	}
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
	// 已发过关闭的 sink 不再重复关：closeWithCode/Close 在关闭中的连接上重复
	// 写 close 帧属无必要系统调用，不配合的客户端由 deadline 统一兜底。
	closed := make(map[FanoutSink]struct{})
	for {
		sinks := h.snapshotSessions()
		remaining := 0
		pending := sinks[:0]
		for _, sink := range sinks {
			if _, done := closed[sink]; done {
				remaining++
				continue
			}
			pending = append(pending, sink)
		}
		if len(pending) == 0 {
			return remaining
		}
		for _, sink := range pending {
			if ws, ok := sink.(interface {
				closeWithCode(code int, reason string) error
			}); ok {
				_ = ws.closeWithCode(1001, "server shutting down")
			} else {
				_ = sink.Close()
			}
			closed[sink] = struct{}{}
		}
		if time.Now().After(deadline) {
			return len(h.snapshotSessions())
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// snapshotSessions 返回当前全部在线 sink 快照（每个 shard 只持读锁复制）。
func (h *Hub) snapshotSessions() []FanoutSink {
	hint := h.sessionCount.Load()
	if hint < 0 {
		hint = 0
	}
	sinks := make([]FanoutSink, 0, hint)
	for i := range h.shards {
		shard := &h.shards[i]
		shard.mu.RLock()
		for _, sessionSinks := range shard.bySession {
			sinks = append(sinks, sessionSinks...)
		}
		shard.mu.RUnlock()
	}
	return sinks
}

func (h *Hub) closeSessionLocal(sessionID string, event Event) {
	sessionShard := h.shardFor("session:" + sessionID)
	sessionShard.mu.RLock()
	sinks := append([]FanoutSink(nil), sessionShard.bySession[sessionID]...)
	sessionShard.mu.RUnlock()
	broadcastSinks(sinks, event)
	for _, sink := range sinks {
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

// entryForSink 返回 sink 的订阅反向索引条目，缺失时创建（sync.Map 无锁读）。
func (h *Hub) entryForSink(sink FanoutSink) *subEntry {
	if entryAny, ok := h.subsBySink.Load(sink); ok {
		if entry, ok := entryAny.(*subEntry); ok {
			return entry
		}
	}
	entry := &subEntry{userKeys: map[string]struct{}{}, groupKeys: map[string]struct{}{}}
	actual, _ := h.subsBySink.LoadOrStore(sink, entry)
	if stored, ok := actual.(*subEntry); ok {
		return stored
	}
	return entry
}

func (h *Hub) SetUserSubscriptions(userIDs []string, sink FanoutSink) {
	entry := h.entryForSink(sink)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed {
		return
	}
	// 全量重建语义：先按反向索引删掉该 sink 的旧 user keys（O(旧 keys)），
	// 不再全表扫描（审计 perf P3-5）。
	for key := range entry.userKeys {
		shard := h.shardFor("watch:" + key)
		shard.mu.Lock()
		removeSinkFromSlice(shard.byUserWatchers, key, sink)
		shard.mu.Unlock()
	}
	entry.userKeys = make(map[string]struct{}, len(userIDs))
	for _, uid := range userIDs {
		shard := h.shardFor("watch:" + uid)
		shard.mu.Lock()
		shard.byUserWatchers[uid] = append(shard.byUserWatchers[uid], sink)
		shard.mu.Unlock()
		entry.userKeys[uid] = struct{}{}
	}
}

func (h *Hub) SetGroupSubscriptions(groupIDs []string, sink FanoutSink) {
	entry := h.entryForSink(sink)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.closed {
		return
	}
	for key := range entry.groupKeys {
		shard := h.shardFor("group:" + key)
		shard.mu.Lock()
		removeSinkFromSlice(shard.byGroup, key, sink)
		shard.mu.Unlock()
	}
	entry.groupKeys = make(map[string]struct{}, len(groupIDs))
	for _, gid := range groupIDs {
		shard := h.shardFor("group:" + gid)
		shard.mu.Lock()
		shard.byGroup[gid] = append(shard.byGroup[gid], sink)
		shard.mu.Unlock()
		entry.groupKeys[gid] = struct{}{}
	}
}

func (h *Hub) BroadcastToUser(userID string, event Event) {
	// resume enrich（spec L173）：为事件分配全局单调 seq 并写入跨实例 user 事件流。
	// 这是 resume 续传的唯一收口——BroadcastToUser 同时掌握 userID 与事件投递，
	// 断线期间（无在线 sink）仍能按 user 累积事件。
	store := h.getResumeStore()
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
	shard := h.shardFor("user:" + userID)
	shard.mu.RLock()
	sinks := append([]FanoutSink(nil), shard.byUser[userID]...)
	shard.mu.RUnlock()
	broadcastSinks(sinks, event)
}

func (h *Hub) BroadcastToUserSubscribers(userID string, event Event) {
	// presence/typing 类瞬态事件不写 resume 流：断线期间的状态无重放价值（重连后
	// 通过订阅重建 + presence 快照恢复），写入反而让 60s 窗口内的 Stream 无谓膨胀。
	h.deliverUserSubscribersLocal(userID, event)
	h.publish("usersub:"+userID, event)
}

func (h *Hub) deliverUserSubscribersLocal(userID string, event Event) {
	shard := h.shardFor("watch:" + userID)
	shard.mu.RLock()
	sinks := append([]FanoutSink(nil), shard.byUserWatchers[userID]...)
	shard.mu.RUnlock()
	broadcastSinks(sinks, event)
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
// 订阅快照。回调在锁外调用：其内部触碰连接侧 subscriptions（自有锁），与 hub
// 锁无交集，避免锁序耦合。
func (h *Hub) revokeGroupLocal(userID, groupID string) {
	userShard := h.shardFor("user:" + userID)
	userShard.mu.RLock()
	userSinks := append([]FanoutSink(nil), userShard.byUser[userID]...)
	userShard.mu.RUnlock()

	affected := make([]FanoutSink, 0, len(userSinks))
	for _, sink := range userSinks {
		entry := h.entryForSink(sink)
		entry.mu.Lock()
		if _, subscribed := entry.groupKeys[groupID]; subscribed {
			delete(entry.groupKeys, groupID)
			affected = append(affected, sink)
		}
		entry.mu.Unlock()
	}
	if len(affected) > 0 {
		groupShard := h.shardFor("group:" + groupID)
		groupShard.mu.Lock()
		byGroup := groupShard.byGroup[groupID]
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
			delete(groupShard.byGroup, groupID)
		} else {
			groupShard.byGroup[groupID] = out
		}
		groupShard.mu.Unlock()
	}
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
	groupShard := h.shardFor("group:" + groupID)
	groupShard.mu.Lock()
	sinks := append([]FanoutSink(nil), groupShard.byGroup[groupID]...)
	delete(groupShard.byGroup, groupID)
	groupShard.mu.Unlock()
	for _, sink := range sinks {
		entry := h.entryForSink(sink)
		entry.mu.Lock()
		delete(entry.groupKeys, groupID)
		entry.mu.Unlock()
	}
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
	store := h.getResumeStore()
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
	shard := h.shardFor("group:" + groupID)
	shard.mu.RLock()
	sinks := append([]FanoutSink(nil), shard.byGroup[groupID]...)
	shard.mu.RUnlock()
	broadcastSinks(sinks, event)
}

// BroadcastToAll sends event to every connected session. It is O(n) for all
// connections — it collects a snapshot of all sinks under per-shard RLock then
// iterates outside the locks. This is used for rare system-wide events only; it
// should not be called on hot paths.
func (h *Hub) BroadcastToAll(event Event) {
	broadcastSinks(h.snapshotSessions(), event)
	h.publish("all", event)
}

func (h *Hub) deliverAllLocal(event Event) {
	broadcastSinks(h.snapshotSessions(), event)
}

func (h *Hub) publish(channel string, event Event) {
	h.cfgMu.RLock()
	p := h.publisher
	h.cfgMu.RUnlock()
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
