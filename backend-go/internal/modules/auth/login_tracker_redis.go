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
//   - 键 auth:loginfail:<login_id>，值 = 失败次数，TTL = defaultLockDuration
//   - 累计失败 >= defaultMaxFailures 即锁定；锁定到期 = 键 TTL 自然过期
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

// recordFailedScript：原子 INCR + 首次失败设置窗口 TTL。
// KEYS[1]=计数键; ARGV[1]=窗口毫秒。已锁定的键继续累计但不重置 TTL。
var recordFailedScript = redis.NewScript(`
local v = redis.call('INCR', KEYS[1])
if v == 1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return v
`)

func (t *RedisLoginTracker) RecordFailed(ctx context.Context, loginID string) {
	err := recordFailedScript.Run(ctx, t.rdb, []string{redisLoginTrackerKey(loginID)},
		defaultLockDuration.Milliseconds()).Err()
	if err != nil {
		observability.L().Warn("login tracker redis record failed", "err", err)
	}
}

func (t *RedisLoginTracker) Clear(ctx context.Context, loginID string) {
	if err := t.rdb.Del(ctx, redisLoginTrackerKey(loginID)).Err(); err != nil {
		observability.L().Warn("login tracker redis clear failed", "err", err)
	}
}
