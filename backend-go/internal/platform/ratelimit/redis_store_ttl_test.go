package ratelimit

import (
	"context"
	"testing"
	"time"
)

// TestRedisTokenBucketSetsExpiry 回归：令牌桶键必须带过期，否则一次性
// IP/用户键永久残留、无界增长。断言允许与拒绝两条路径都续期。
func TestRedisTokenBucketSetsExpiry(t *testing.T) {
	rdb, _ := newTestRedis(t)
	SetRedisStore(rdb)
	t.Cleanup(func() { SetRedisStore(nil) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rl := rule{name: "t", strategy: "token_bucket", limit: 1, window: time.Minute, scope: "s", key: "k"}
	now := time.Now().UTC()
	if _, limited := redisAllow(now, rl); limited {
		t.Fatal("first request must pass")
	}
	if _, limited := redisAllow(now, rl); !limited {
		t.Fatal("second request must be limited")
	}
	ttl, err := rdb.PTTL(ctx, "t::s::k").Result()
	if err != nil {
		t.Fatalf("pttl: %v", err)
	}
	if ttl <= 0 || ttl > 2*time.Minute {
		t.Fatalf("expected 0 < ttl <= 2m, got %v", ttl)
	}
}
