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

// ResumeStore 维护 resume 增量恢复所需的跨实例状态（spec L156, L159）。
//
// 实现中立（spec L159）：Redis 不可用时用 NopResumeStore 降级——Record 返回 0
// （event 不带 seq），resume 恒失效，客户端回退全量重放（spec L161 降级保证）。
type ResumeStore interface {
	// Record 为 userID 的一条出站事件分配 per-user 单调 seq 并写入事件流，
	// 返回分配的 seq（供 Hub enrich 到 event.ServerSeq）。失败/降级返回 0。
	Record(userID string, event Event) int64
	// Replay 返回 userID 事件流中 seq > fromSeq 的事件（按 seq 升序，已带 ServerSeq）。
	// 存储不可用或无事件时返回 nil。
	Replay(userID string, fromSeq int64) []Event
	// LastSeq 返回 userID 事件流的当前最大 seq（连接建立时下发，供客户端记录基线）。
	LastSeq(userID string) int64
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

func (NopResumeStore) Record(string, Event) int64           { return 0 }
func (NopResumeStore) Replay(string, int64) []Event         { return nil }
func (NopResumeStore) LastSeq(string) int64                 { return 0 }
func (NopResumeStore) SaveState(string, ResumeState)        {}
func (NopResumeStore) LoadState(string) (ResumeState, bool) { return ResumeState{}, false }
func (NopResumeStore) Close() error                         { return nil }

// Redis 键命名约定：
//   - resume:user:<userID>  Stream：该 user 的出站事件队列（entry ID = seq）
//   - resume:seq:<userID>   INCR 计数器：该 user 的 per-user 单调 seq
//   - resume:state:<token>  String：resume_token ↔ ResumeState（JSON）
const (
	resumeStreamPrefix = "resume:user:"
	resumeSeqPrefix    = "resume:seq:"
	resumeStatePrefix  = "resume:state:"
)

// recordScript 原子完成「per-user seq 分配 + Stream 写入 + TTL 续期」（一次往返）。
// 用整数 seq 作为 Stream entry ID（Redis 允许纯数字递增 ID），使 seq 与 entry ID
// 一致——Replay 时直接用 XRANGE by ID 精确续传，并从 entry ID 反推 ServerSeq。
// event 以原始形态（不含 seq）存储以省空间；seq 在 Replay 时由 entry ID 补回。
var recordScript = redis.NewScript(`
local seq = redis.call('INCR', KEYS[1])
redis.call('XADD', KEYS[2], seq, 'e', ARGV[1])
redis.call('PEXPIRE', KEYS[1], ARGV[2])
redis.call('PEXPIRE', KEYS[2], ARGV[2])
return seq
`)

// RedisResumeStore 用 Redis Streams + INCR 实现 ResumeStore（跨实例共享，spec L159）。
// 共享调用方的 *redis.Client；Close 不关闭它（由 bootstrap 统一管理，与 RedisBridge 同源）。
type RedisResumeStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewRedisResumeStore 创建基于共享 rdb 的存储。ttl<=0 时用 DefaultResumeTTL。
// rdb 必须非 nil（Redis 不可用时调用方应改用 NopResumeStore）。
func NewRedisResumeStore(rdb *redis.Client, ttl time.Duration) *RedisResumeStore {
	if ttl <= 0 {
		ttl = DefaultResumeTTL
	}
	return &RedisResumeStore{rdb: rdb, ttl: ttl}
}

func (s *RedisResumeStore) seqKey(userID string) string    { return resumeSeqPrefix + userID }
func (s *RedisResumeStore) streamKey(userID string) string { return resumeStreamPrefix + userID }
func (s *RedisResumeStore) stateKey(token string) string   { return resumeStatePrefix + token }

// Record 分配 seq 并写入 Stream。best-effort：任何 Redis 错误只记日志并返回 0，
// 不阻塞 Hub 广播（event 不带 seq → 客户端按全量重放兜底，spec L161）。
func (s *RedisResumeStore) Record(userID string, event Event) int64 {
	if userID == "" {
		return 0
	}
	blob, err := json.Marshal(event)
	if err != nil {
		observability.L().Warn("resume: marshal event failed", "user_id", userID, "err", err)
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ttlMs := int64(s.ttl / time.Millisecond)
	seq, err := recordScript.Run(ctx, s.rdb,
		[]string{s.seqKey(userID), s.streamKey(userID)},
		string(blob), ttlMs).Int64()
	if err != nil {
		observability.L().Warn("resume: record failed, client will fall back to full replay", "user_id", userID, "err", err)
		return 0
	}
	return seq
}

// Replay 回放 seq > fromSeq 的事件。从 entry ID（=seq）反推 ServerSeq 补回 event。
func (s *RedisResumeStore) Replay(userID string, fromSeq int64) []Event {
	if userID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := fmt.Sprintf("%d", fromSeq+1)
	vals, err := s.rdb.XRange(ctx, s.streamKey(userID), start, "+").Result()
	if err != nil {
		observability.L().Warn("resume: replay failed", "user_id", userID, "err", err)
		return nil
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
	return events
}

// LastSeq 返回当前最大 seq（INCR 计数器当前值）。
func (s *RedisResumeStore) LastSeq(userID string) int64 {
	if userID == "" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	seq, err := s.rdb.Get(ctx, s.seqKey(userID)).Int64()
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
