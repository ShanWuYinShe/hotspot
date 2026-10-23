package auth

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/platform/observability"
)

// RedisLoginTracker 是 FailedLoginTracker 的 Redis 实现（P2-2）：进程内实现
// 各实例独立计数且重启即丢，多实例部署下爆破阈值被放大 N 倍。切换到本实现后
// 所有实例共享计数，重启不丢锁定状态。
//
// 键语义（与 inMemoryLoginTracker 阈值一致）：
//   - 键 auth:loginfail:<login_id>，值 = 失败次数
//   - 未达阈值时每次失败滑动续期 TTL = defaultLockDuration：计数仅在
//     「静默满 defaultLockDuration」后衰减（对齐 inMemory 的
//     loginCounterDecay），窗口内连续失败持续累计直至锁定；
//   - 累计失败达到 defaultMaxFailures 的那一次重设 TTL = defaultLockDuration，
//     即锁定窗口自「触发锁定的时刻」起算（对齐 inMemory 的
//     lockUntil = now + defaultLockDuration；此前 TTL 自首次失败起算，
//     慢速攻击下实际锁期可被压缩到分钟级）；
//   - 已锁定后继续失败不重置 TTL（对齐 inMemory 的「不重置计时器」）
type RedisLoginTracker struct {
	rdb *redis.Client
}

func NewRedisLoginTracker(rdb *redis.Client) *RedisLoginTracker { return &RedisLoginTracker{rdb: rdb} }

func redisLoginTrackerKey(loginID string) string { return "auth:loginfail:" + loginID }

func (t *RedisLoginTracker) Check(ctx context.Context, loginID string) error {
	v, err := t.rdb.Get(ctx, redisLoginTrackerKey(loginID)).Int()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		// Redis 故障 fail-open：锁定计数无 Postgres 持久事实可回退（与限流
		// 口径一致），放行但记 Warn 保持可观测。
		observability.L().Warn("login tracker redis unavailable, failing open", "err", err)
		return nil
	}
	if v >= defaultMaxFailures {
		return ErrAccountLocked
	}
	return nil
}

// recordFailedScript：原子 INCR + TTL 管理。
// KEYS[1]=计数键; ARGV[1]=窗口毫秒; ARGV[2]=锁定阈值。
//   - v <= 阈值：每次失败滑动续期 TTL（计数衰减 = 静默满窗口，与 inMemory
//     的 loginCounterDecay 同语义）；
//   - v == 阈值：锁定触发，TTL 重设为完整窗口（锁期自触发时刻起算）；
//   - v > 阈值：已锁定，不重置 TTL（不延长锁期）。
var recordFailedScript = redis.NewScript(`
local v = redis.call('INCR', KEYS[1])
if v <= tonumber(ARGV[2]) then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return v
`)

func (t *RedisLoginTracker) RecordFailed(ctx context.Context, loginID string) {
	err := recordFailedScript.Run(ctx, t.rdb, []string{redisLoginTrackerKey(loginID)},
		defaultLockDuration.Milliseconds(), defaultMaxFailures).Err()
	if err != nil {
		observability.L().Warn("login tracker redis record failed", "err", err)
	}
}

func (t *RedisLoginTracker) Clear(ctx context.Context, loginID string) {
	if err := t.rdb.Del(ctx, redisLoginTrackerKey(loginID)).Err(); err != nil {
		observability.L().Warn("login tracker redis clear failed", "err", err)
	}
}
