package ratelimit

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/platform/observability"
)

// redis_store.go —— 限流决策的分布式实现。进程内实现（windows/tokens map）
// 各实例独立计数，多实例部署下限流阈值被放大 N 倍；切换到 RedisStore 后所有
// 实例共享计数，三个策略均以 Lua 脚本在 Redis 侧原子执行。
//
// 失败语义：Redis 不可达/脚本出错时 fail-open（记 Warn 并放行）——限流的
// 目的是保护而非鉴权，与封禁检查的 fail-open 口径一致；不打断业务请求。

var redisClient *redis.Client

// SetRedisStore 切换到 Redis 分布式计数。传 nil 恢复进程内实现（dev/test）。
// 由 bootstrap 在 Redis 健康检查通过后调用；调用与请求路径并发安全
// （client 指针原子性由 Go 内存模型保证——写入发生在任何请求开始前，
// bootstrap 串行装配）。
func SetRedisStore(rdb *redis.Client) { redisClient = rdb }

// redisAllow 以 Redis Lua 原子执行 rl 策略。返回与进程内 allow 相同的
// (retryAfterSeconds, limited) 契约。Redis 故障时 fail-open。
func redisAllow(now time.Time, rl rule) (int, bool) {
	if redisClient == nil {
		return allow(now, rl)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	composite := rl.name + "::" + rl.scope + "::" + rl.key
	var (
		retry     int
		limited   bool
		scriptErr error
	)
	switch rl.strategy {
	case "token_bucket":
		retry, limited, scriptErr = redisTokenBucket(ctx, redisClient, composite, now, rl.limit, rl.window)
	case "fixed":
		retry, limited, scriptErr = redisFixedWindow(ctx, redisClient, composite, now, rl.limit, rl.window)
	default:
		retry, limited, scriptErr = redisSlidingWindow(ctx, redisClient, composite, now, rl.limit, rl.window)
	}
	if scriptErr != nil {
		// fail-open：Redis 故障期间放行（限流是保护机制，不阻断业务）
		observability.L().Warn("rate limit redis backend unavailable, failing open",
			"rule", rl.name, "key", rl.key, "err", scriptErr)
		return 0, false
	}
	return retry, limited
}

// slidingWindowScript：ZSET 滑动窗口。
// KEYS[1]=key; ARGV: now_ms, window_ms, limit
// 返回 {limited(0/1), retry_after_ms}
var slidingWindowScript = redis.NewScript(`
local cutoff = tonumber(ARGV[1]) - tonumber(ARGV[2])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', cutoff)
local count = redis.call('ZCARD', KEYS[1])
if count >= tonumber(ARGV[3]) then
	local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
	local retry_ms = 0
	if oldest[2] then
		retry_ms = math.max(1, math.ceil(tonumber(oldest[2]) + tonumber(ARGV[2]) - tonumber(ARGV[1])))
	end
	return {1, retry_ms}
end
redis.call('ZADD', KEYS[1], tonumber(ARGV[1]), ARGV[1] .. ':' .. tostring(redis.call('INCR', KEYS[1] .. ':seq')))
redis.call('PEXPIRE', KEYS[1], tonumber(ARGV[2]))
redis.call('PEXPIRE', KEYS[1] .. ':seq', tonumber(ARGV[2]))
return {0, 0}
`)

func redisSlidingWindow(ctx context.Context, rdb *redis.Client, key string, now time.Time, limit int, window time.Duration) (int, bool, error) {
	res, err := slidingWindowScript.Run(ctx, rdb, []string{key},
		now.UnixMilli(), window.Milliseconds(), limit).Slice()
	if err != nil {
		return 0, false, err
	}
	return parseScriptResult(res)
}

// fixedWindowScript：固定窗口计数（键带窗口起点，自动过期双窗口时长）。
// KEYS[1]=key; ARGV: window_start_ms, window_ms, limit, now_ms
var fixedWindowScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
	redis.call('PEXPIRE', KEYS[1], tonumber(ARGV[2]) * 2)
end
if count > tonumber(ARGV[3]) then
	local retry_ms = math.max(1, tonumber(ARGV[1]) + tonumber(ARGV[2]) - tonumber(ARGV[4]))
	return {1, retry_ms}
end
return {0, 0}
`)

func redisFixedWindow(ctx context.Context, rdb *redis.Client, key string, now time.Time, limit int, window time.Duration) (int, bool, error) {
	windowStart := now.Truncate(window)
	res, err := fixedWindowScript.Run(ctx, rdb, []string{key + "::" + strconv.FormatInt(windowStart.UnixMilli(), 10)},
		windowStart.UnixMilli(), window.Milliseconds(), limit, now.UnixMilli()).Slice()
	if err != nil {
		return 0, false, err
	}
	return parseScriptResult(res)
}

// tokenBucketScript：令牌桶（HASH 存 tokens/last_refill_ms）。
// KEYS[1]=key; ARGV: now_ms, capacity(=limit), rate_per_ms(浮点)
// 每次 HSET 后续期 PEXPIRE（2×窗口，由 capacity/rate 反推窗口毫秒）：
// 此前无过期，一次性 IP/用户键永久残留，随运行时间无界增长。
var tokenBucketScript = redis.NewScript(`
local state = redis.call('HMGET', KEYS[1], 'tokens', 'last_ms')
local tokens = tonumber(state[1])
local last = tonumber(state[2])
local now = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local rate = tonumber(ARGV[3])
if tokens == nil then
	tokens = capacity
	last = now
end
tokens = math.min(capacity, tokens + (now - last) * rate)
if tokens < 1 then
	redis.call('HSET', KEYS[1], 'tokens', tokens, 'last_ms', now)
	redis.call('PEXPIRE', KEYS[1], math.ceil(tonumber(ARGV[2]) / rate * 2))
	local retry_ms = math.max(1, math.ceil((1 - tokens) / rate))
	return {1, retry_ms}
end
redis.call('HSET', KEYS[1], 'tokens', tokens - 1, 'last_ms', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(tonumber(ARGV[2]) / rate * 2))
return {0, 0}
`)

func redisTokenBucket(ctx context.Context, rdb *redis.Client, key string, now time.Time, limit int, window time.Duration) (int, bool, error) {
	// 速率单位 = 令牌/毫秒（脚本内 now/last 均为 UnixMilli）
	rate := float64(limit) / float64(window.Milliseconds())
	res, err := tokenBucketScript.Run(ctx, rdb, []string{key},
		now.UnixMilli(), limit, rate).Slice()
	if err != nil {
		return 0, false, err
	}
	return parseScriptResult(res)
}

func parseScriptResult(res []any) (int, bool, error) {
	if len(res) != 2 {
		return 0, false, errUnexpectedScriptResult
	}
	limited, ok1 := res[0].(int64)
	retryMs, ok2 := res[1].(int64)
	if !ok1 || !ok2 {
		return 0, false, errUnexpectedScriptResult
	}
	if limited == 0 {
		return 0, false, nil
	}
	retry := int(math.Ceil(float64(retryMs) / 1000))
	if retry < 1 {
		retry = 1
	}
	return retry, true, nil
}

// errUnexpectedScriptResult 表示 Redis 脚本返回了非契约形状的结果。
var errUnexpectedScriptResult = redisError("ratelimit: unexpected script result")

type redisError string

func (e redisError) Error() string { return string(e) }
