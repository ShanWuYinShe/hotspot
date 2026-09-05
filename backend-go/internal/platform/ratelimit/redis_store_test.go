package ratelimit

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// 与进程内实现的契约对齐：
//   - 滑动窗口：limit 内放行、超限拒绝且 retry_after ≈ 最旧事件过窗时间
//   - 固定窗口：跨窗口计数重置
//   - 令牌桶：突发额度耗尽后按速率恢复
//   - fail-open：Redis 不可用时不阻断请求
//   - 共享计数：两个“实例”（client）看到同一计数（多实例语义的核心）

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb, mr
}

func TestRedisSlidingWindowLimitAndRetry(t *testing.T) {
	rdb, _ := newTestRedis(t)
	SetRedisStore(rdb)
	t.Cleanup(func() { SetRedisStore(nil) })

	now := time.Now()
	rl := rule{name: "t", strategy: "sliding", limit: 3, window: time.Minute, key: "u1"}
	for i := 0; i < 3; i++ {
		if retry, limited := redisAllow(now.Add(time.Duration(i)*time.Second), rl); limited {
			t.Fatalf("request %d must pass, got limited retry=%d", i, retry)
		}
	}
	retry, limited := redisAllow(now.Add(4*time.Second), rl)
	if !limited {
		t.Fatal("4th request must be limited")
	}
	// 最旧事件在 +1s，过窗时间 = +61s → retry ≈ 57s
	if retry < 55 || retry > 59 {
		t.Fatalf("unexpected retry_after: %d", retry)
	}
}

func TestRedisFixedWindowResetsAcrossWindows(t *testing.T) {
	rdb, _ := newTestRedis(t)
	SetRedisStore(rdb)
	t.Cleanup(func() { SetRedisStore(nil) })

	now := time.Now().Truncate(time.Minute)
	rl := rule{name: "t", strategy: "fixed", limit: 2, window: time.Minute, key: "u1"}
	for i := 0; i < 2; i++ {
		if _, limited := redisAllow(now.Add(time.Duration(i)*time.Second), rl); limited {
			t.Fatalf("request %d must pass", i)
		}
	}
	if _, limited := redisAllow(now.Add(3*time.Second), rl); !limited {
		t.Fatal("3rd request in window must be limited")
	}
	// 下一窗口重置
	if _, limited := redisAllow(now.Add(2*time.Minute), rl); limited {
		t.Fatal("next window must reset the count")
	}
}

func TestRedisTokenBucketRefills(t *testing.T) {
	rdb, _ := newTestRedis(t)
	SetRedisStore(rdb)
	t.Cleanup(func() { SetRedisStore(nil) })

	now := time.Now()
	// limit=2 / minute → 2 令牌突发后耗尽
	rl := rule{name: "t", strategy: "token_bucket", limit: 2, window: time.Minute, key: "u1"}
	for i := 0; i < 2; i++ {
		if _, limited := redisAllow(now, rl); limited {
			t.Fatalf("burst request %d must pass", i)
		}
	}
	retry, limited := redisAllow(now, rl)
	if !limited {
		t.Fatal("empty bucket must limit")
	}
	if retry < 25 || retry > 35 { // 1 令牌 / (2/60)s ≈ 30s
		t.Fatalf("unexpected retry_after: %d", retry)
	}
}

func TestRedisSharedCountAcrossInstances(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	SetRedisStore(rdb)
	t.Cleanup(func() { SetRedisStore(nil) })

	now := time.Now()
	rl := rule{name: "t", strategy: "sliding", limit: 3, window: time.Minute, key: "shared"}
	// 三个“实例”各自发一条：共享计数下第 4 条必须被限
	for i := 0; i < 3; i++ {
		if _, limited := redisAllow(now.Add(time.Duration(i)*time.Second), rl); limited {
			t.Fatalf("request %d must pass", i)
		}
	}
	if _, limited := redisAllow(now.Add(4*time.Second), rl); !limited {
		t.Fatal("shared counter must limit the 4th request across instances")
	}
}

func TestRedisFailOpenOnBackendDown(t *testing.T) {
	rdb, mr := newTestRedis(t)
	SetRedisStore(rdb)
	t.Cleanup(func() { SetRedisStore(nil) })
	mr.Close() // 后端宕机

	rl := rule{name: "t", strategy: "sliding", limit: 1, window: time.Minute, key: "u1"}
	for i := 0; i < 3; i++ {
		if _, limited := redisAllow(time.Now(), rl); limited {
			t.Fatal("backend down must fail open (not limit)")
		}
	}
}
