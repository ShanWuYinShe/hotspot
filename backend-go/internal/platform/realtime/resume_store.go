package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/platform/observability"
)

// DefaultResumeTTL 是 resume 状态的默认保留窗口（spec L156：TTL ≈ 2×ping_interval，
// ping_interval=30s → 60s）。连接断开后服务端在该窗口内可增量续传；超期则客户端
// 回退全量重放（spec L157, L161）。可通过 config RESUME_TTL_SECONDS 覆盖。
const DefaultResumeTTL = 60 * time.Second

// resumeStreamMaxLen 是单条 resume Stream 的条目硬上限（XTRIM MAXLEN）：
// 脚本每次写入都续期 PEXPIRE，持续活跃的流永不过期，若无上限会随进程
// 运行时间无界增长（热群 1 条/秒即每天 8.6 万条全量事件 JSON）。2000 条覆盖
// 60 秒窗口内 33 条/秒的持续写入；超出后最旧条目被丢弃，基线早于裁剪点的
// 重连客户端将静默缺失中间事件（与 60 秒 TTL 过期回退全量重放同属弱一致范畴，
// 残余风险：单流 60 秒超 2000 事件的极端热点）。
const resumeStreamMaxLen = 2000

// ResumeStore 维护 resume 增量恢复所需的跨实例状态（spec L156, L159）。
//
// seq 为全局单调（单一 INCR 计数器）：user 事件与 group 事件共享同一 seq 空间，
// 保证客户端视角的 server_seq 单调递增（spec L173）——若 user/group 各用独立
// 计数器，两套 seq 不可比，客户端按 seq 去重/排序会错乱。
//
// 实现中立（spec L159）：Redis 不可用时用 NopResumeStore 降级——Record 返回 0
// （event 不带 seq），resume 恒失效，客户端回退全量重放（spec L161 降级保证）。
type ResumeStore interface {
	// Record 为 userID 的一条出站事件分配全局单调 seq 并写入该 user 的事件流，
	// 返回分配的 seq（供 Hub enrich 到 event.ServerSeq）。失败/降级返回 0。
	Record(userID string, event Event) int64
	// RecordGroup 为 groupID 的一条出站事件分配全局单调 seq 并写入该 group 的
	// 事件流（断线成员重连时按订阅快照聚合回放，见 Replay）。失败/降级返回 0。
	RecordGroup(groupID string, event Event) int64
	// Replay 返回 userID 的 user 流 + groupIDs 各 group 流中 seq > fromSeq 的事件，
	// 跨流按全局 seq 升序归并（已带 ServerSeq）。存储不可用或无事件时返回 nil。
	Replay(userID string, groupIDs []string, fromSeq int64) []Event
	// LastSeq 返回当前全局最大 seq（连接建立时下发，供客户端记录基线）。
	LastSeq() int64
	// SaveState 将 resume 状态与 token 绑定并设置 TTL（spec L156「订阅状态由服务端恢复」）。
	SaveState(token string, state ResumeState)
	// LoadState 读取 token 绑定的 resume 状态；token 不存在/过期返回 ok=false。
	LoadState(token string) (ResumeState, bool)
	// Close 释放底层资源。
	Close() error
}

// NopResumeStore 是 Redis 不可用时的降级实现：所有操作 no-op，Record 返回 0
// 触发客户端全量重放兜底（spec L161）。零值可用。
type NopResumeStore struct{}

func (NopResumeStore) Record(string, Event) int64             { return 0 }
func (NopResumeStore) RecordGroup(string, Event) int64        { return 0 }
func (NopResumeStore) Replay(string, []string, int64) []Event { return nil }
func (NopResumeStore) LastSeq() int64                         { return 0 }
func (NopResumeStore) SaveState(string, ResumeState)          {}
func (NopResumeStore) LoadState(string) (ResumeState, bool)   { return ResumeState{}, false }
func (NopResumeStore) Close() error                           { return nil }

// Redis 键命名约定：
//   - resume:gseq             INCR 计数器：全局单调 seq（不过期，保证单调性）
//   - resume:user:<userID>    Stream：该 user 的私聊/通知出站事件队列（entry ID = 全局 seq）
//   - resume:group:<groupID>  Stream：该 group 的群事件队列（entry ID = 全局 seq）
//   - resume:state:<token>    String：resume_token ↔ ResumeState（JSON）
const (
	resumeGlobalSeqKey      = "resume:gseq"
	resumeStreamPrefix      = "resume:user:"
	resumeGroupStreamPrefix = "resume:group:"
	resumeStatePrefix       = "resume:state:"
)

// recordScript 原子完成「全局 seq 分配 + Stream 写入 + Stream TTL 续期」（一次往返）。
// 用整数 seq 作为 Stream entry ID（Redis 允许纯数字递增 ID），使 seq 与 entry ID
// 一致——Replay 时直接用 XRANGE by ID 精确续传，并从 entry ID 反推 ServerSeq。
// gseq 键不设 TTL（全局单调性必须持久）；event 以原始形态（不含 seq）存储以省空间，
// seq 在 Replay 时由 entry ID 补回。
var recordScript = redis.NewScript(`
local seq = redis.call('INCR', KEYS[1])
redis.call('XADD', KEYS[2], seq, 'e', ARGV[1])
redis.call('XTRIM', KEYS[2], 'MAXLEN', ARGV[3])
redis.call('PEXPIRE', KEYS[2], ARGV[2])
return seq
`)

// recordAndPublishScript 在 recordScript 基础上追加跨实例 PUBLISH（仍一次往返），
// 广播热路径由两次串行 Redis 往返（record → publish）降为一次（审计 perf P1-4）。
// 发布载荷为 bridge 信封的手工拼接 JSON：{"o":<origin>,"s":<seq>,"e":<event>}——
// s 为顶层数字 seq（int64 十进制文本拼接，无 double 精度问题），e 为不含 seq 的
// 原始事件 blob；bridge 解码后把 s 回填到 Event.ServerSeq（见 decodeEnvelope）。
// 不用 cjson 补写事件内 server_seq：cjson 数字经 double 往返有精度风险，且拼接
// 不依赖事件字段形态。PUBLISH 放脚本末尾：XADD 先落，跨实例投递不会快于落库。
var recordAndPublishScript = redis.NewScript(`
local seq = redis.call('INCR', KEYS[1])
redis.call('XADD', KEYS[2], seq, 'e', ARGV[1])
redis.call('XTRIM', KEYS[2], 'MAXLEN', ARGV[3])
redis.call('PEXPIRE', KEYS[2], ARGV[2])
redis.call('PUBLISH', KEYS[3], '{"o":' .. ARGV[4] .. ',"s":' .. seq .. ',"e":' .. ARGV[1] .. '}')
return seq
`)

// RedisResumeStore 用 Redis Streams + INCR 实现 ResumeStore（跨实例共享，spec L159）。
// 共享调用方的 *redis.Client；Close 不关闭它（由 bootstrap 统一管理，与 RedisBridge 同源）。
type RedisResumeStore struct {
	rdb    *redis.Client
	ttl    time.Duration
	origin string
}

// NewRedisResumeStore 创建基于共享 rdb 的存储。ttl<=0 时用 DefaultResumeTTL。
// rdb 必须非 nil（Redis 不可用时调用方应改用 NopResumeStore）。
// origin 为本实例标识（与 RedisBridge 的 instanceID 一致）：组合脚本发布的信封
// 带此 origin，bridge 靠它跳过本实例消息——两处不一致会导致本实例重复投递。
func NewRedisResumeStore(rdb *redis.Client, ttl time.Duration, origin string) *RedisResumeStore {
	if ttl <= 0 {
		ttl = DefaultResumeTTL
	}
	return &RedisResumeStore{rdb: rdb, ttl: ttl, origin: origin}
}

func (s *RedisResumeStore) streamKey(userID string) string { return resumeStreamPrefix + userID }
func (s *RedisResumeStore) groupStreamKey(groupID string) string {
	return resumeGroupStreamPrefix + groupID
}
func (s *RedisResumeStore) stateKey(token string) string { return resumeStatePrefix + token }

// recordInto 分配全局 seq 并写入指定 Stream。best-effort：任何 Redis 错误只记日志
// 并返回 0，不阻塞 Hub 广播（event 不带 seq → 客户端按全量重放兜底，spec L161）。
func (s *RedisResumeStore) recordInto(streamKey string, event Event) int64 {
	blob, err := json.Marshal(event)
	if err != nil {
		observability.L().Warn("resume: marshal event failed", "stream", streamKey, "err", err)
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ttlMs := int64(s.ttl / time.Millisecond)
	seq, err := recordScript.Run(ctx, s.rdb,
		[]string{resumeGlobalSeqKey, streamKey},
		string(blob), ttlMs, resumeStreamMaxLen).Int64()
	if err != nil {
		observability.L().Warn("resume: record failed, client will fall back to full replay", "stream", streamKey, "err", err)
		return 0
	}
	return seq
}

// Record 为 user 私聊/通知事件分配全局 seq 并写入 user Stream。
func (s *RedisResumeStore) Record(userID string, event Event) int64 {
	if userID == "" {
		return 0
	}
	return s.recordInto(s.streamKey(userID), event)
}

// RecordGroup 为群事件分配全局 seq 并写入 group Stream。断线成员重连时按订阅快照
// 聚合回放（Replay），无需在广播时解析成员（成员查询放广播热路径会放大写压力）。
func (s *RedisResumeStore) RecordGroup(groupID string, event Event) int64 {
	if groupID == "" {
		return 0
	}
	return s.recordInto(s.groupStreamKey(groupID), event)
}

// RecordUserAndPublish 在单次 Redis 往返内完成「seq 分配 + Stream 写入 + 跨实例
// PUBLISH」（审计 perf P1-4：广播热路径原为 record → publish 两次串行往返）。
// published=true 表示跨实例发布已随脚本完成，Hub 可跳过独立 publish。
// 失败返回 (0,false)，Hub 降级回退两步路径（record 语义：seq 缺失 → 客户端
// 全量重放兜底；publish 语义：失败仅记日志）。origin 未配置时按不支持处理。
func (s *RedisResumeStore) RecordUserAndPublish(userID, channel string, event Event) (int64, bool) {
	if userID == "" || s.origin == "" {
		return 0, false
	}
	return s.recordAndPublishInto(s.streamKey(userID), channel, event)
}

// RecordGroupAndPublish 同 RecordUserAndPublish，写入 group Stream 并发布群频道。
func (s *RedisResumeStore) RecordGroupAndPublish(groupID, channel string, event Event) (int64, bool) {
	if groupID == "" || s.origin == "" {
		return 0, false
	}
	return s.recordAndPublishInto(s.groupStreamKey(groupID), channel, event)
}

// recordAndPublishInto 组合脚本执行。channel 必须是 bridge 语义的不带前缀频道
// （如 "user:<id>"），脚本内补 hotspot: 前缀，与 RedisBridge.Publish 对齐。
func (s *RedisResumeStore) recordAndPublishInto(streamKey, channel string, event Event) (int64, bool) {
	blob, err := json.Marshal(event)
	if err != nil {
		observability.L().Warn("resume: marshal event failed", "stream", streamKey, "err", err)
		return 0, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ttlMs := int64(s.ttl / time.Millisecond)
	seq, err := recordAndPublishScript.Run(ctx, s.rdb,
		[]string{resumeGlobalSeqKey, streamKey, redisChannelPrefix + channel},
		string(blob), ttlMs, resumeStreamMaxLen, strconv.Quote(s.origin)).Int64()
	if err != nil {
		observability.L().Warn("realtime: record+publish failed, falling back to two-step path", "stream", streamKey, "err", err)
		return 0, false
	}
	return seq, true
}

// Replay 回放 user 流 + groupIDs 各 group 流中 seq > fromSeq 的事件，跨流按全局
// seq 升序归并（k-way merge）。各流内部 XRANGE 已按 entry ID（=seq）升序，且全局
// INCR 保证同一 seq 只写入一个流，故归并结果严格递增。groupIDs 上限由订阅上限
// （spec：group ≤ 100）约束，resume 为低频操作，k ≤ 101 的归并可接受。
func (s *RedisResumeStore) Replay(userID string, groupIDs []string, fromSeq int64) []Event {
	if userID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := fmt.Sprintf("%d", fromSeq+1)
	keys := make([]string, 0, len(groupIDs)+1)
	keys = append(keys, s.streamKey(userID))
	for _, gid := range groupIDs {
		if gid != "" {
			keys = append(keys, s.groupStreamKey(gid))
		}
	}
	pipe := s.rdb.Pipeline()
	cmds := make([]*redis.XMessageSliceCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.XRange(ctx, key, start, "+")
	}
	if _, err := pipe.Exec(ctx); err != nil {
		// Exec 只报传输级错误；单键错误在各 cmd.Err() 中，下面按流降级。
		observability.L().Debug("resume: replay pipeline exec failed", "user_id", userID, "err", err)
	}

	streams := make([][]Event, 0, len(keys))
	for i, cmd := range cmds {
		vals, err := cmd.Result()
		if err != nil {
			observability.L().Warn("resume: replay stream failed, skipping", "user_id", userID, "stream", keys[i], "err", err)
			streams = append(streams, nil)
			continue
		}
		events := make([]Event, 0, len(vals))
		for _, m := range vals {
			var ev Event
			if raw, ok := m.Values["e"].(string); ok {
				if err := json.Unmarshal([]byte(raw), &ev); err == nil {
					ev.ServerSeq = seqFromStreamID(m.ID)
					events = append(events, ev)
				}
			}
		}
		streams = append(streams, events)
	}
	return mergeEvents(streams)
}

// mergeEvents 对多个「按 ServerSeq 升序」的事件流做 k-way 归并。
func mergeEvents(streams [][]Event) []Event {
	total := 0
	for _, s := range streams {
		total += len(s)
	}
	if total == 0 {
		return nil
	}
	idx := make([]int, len(streams))
	out := make([]Event, 0, total)
	for len(out) < total {
		pick := -1
		for i, s := range streams {
			if idx[i] >= len(s) {
				continue
			}
			if pick == -1 || s[idx[i]].ServerSeq < streams[pick][idx[pick]].ServerSeq {
				pick = i
			}
		}
		out = append(out, streams[pick][idx[pick]])
		idx[pick]++
	}
	return out
}

// LastSeq 返回全局 INCR 计数器的当前值（全局单调，不过期）。
func (s *RedisResumeStore) LastSeq() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	seq, err := s.rdb.Get(ctx, resumeGlobalSeqKey).Int64()
	if err != nil {
		return 0 // redis.Nil 或其他错误都视为 0（无历史事件）
	}
	return seq
}

// SaveState 将 ResumeState 以 JSON 存入 Redis 并设置 TTL（绑定 token）。
func (s *RedisResumeStore) SaveState(token string, state ResumeState) {
	if token == "" {
		return
	}
	blob, err := json.Marshal(state)
	if err != nil {
		observability.L().Warn("resume: marshal state failed", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.rdb.Set(ctx, s.stateKey(token), blob, s.ttl).Err(); err != nil {
		observability.L().Warn("resume: save state failed", "err", err)
	}
}

// LoadState 读取 token 绑定的状态。token 不存在/过期/存储故障返回 ok=false（回退全量）。
func (s *RedisResumeStore) LoadState(token string) (ResumeState, bool) {
	if token == "" {
		return ResumeState{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	blob, err := s.rdb.Get(ctx, s.stateKey(token)).Bytes()
	if err == redis.Nil {
		return ResumeState{}, false
	}
	if err != nil {
		observability.L().Warn("resume: load state failed", "err", err)
		return ResumeState{}, false
	}
	var state ResumeState
	if err := json.Unmarshal(blob, &state); err != nil {
		observability.L().Warn("resume: unmarshal state failed", "err", err)
		return ResumeState{}, false
	}
	return state, true
}

// Close 不关闭共享 rdb（由 bootstrap 统一管理，与 RedisBridge 同源）。
func (s *RedisResumeStore) Close() error { return nil }

// seqFromStreamID 从 Redis Stream entry ID 解析 seq。entry ID 形如 "123"（我们用
// 整数 seq 作 ID）；兼容 Redis 内部 "ms-seq" 格式，取 '-' 前段。
func seqFromStreamID(id string) int64 {
	if i := strings.IndexByte(id, '-'); i >= 0 {
		id = id[:i]
	}
	seq, _ := strconv.ParseInt(id, 10, 64)
	return seq
}
