package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedisTracker(t *testing.T) (*RedisLoginTracker, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRedisLoginTracker(rdb), mr
}

// P2-2：登录失败锁定切 Redis——多实例共享计数、重启不丢、阈值语义与进程内
// 实现一致（5 次失败 → 15 分钟锁定）。
func TestRedisLoginTrackerLockoutSemantics(t *testing.T) {
	tracker, mr := newTestRedisTracker(t)
	ctx := context.Background()

	for i := 0; i < defaultMaxFailures-1; i++ {
		tracker.RecordFailed(ctx, "user@example.com")
	}
	if err := tracker.Check(ctx, "user@example.com"); err != nil {
		t.Fatalf("expected not locked before threshold, got %v", err)
	}

	tracker.RecordFailed(ctx, "user@example.com")
	if err := tracker.Check(ctx, "user@example.com"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked at threshold, got %v", err)
	}
	// 其他 login_id 不受影响（按账号维度隔离）。
	if err := tracker.Check(ctx, "other@example.com"); err != nil {
		t.Fatalf("expected other login unaffected, got %v", err)
	}

	// 锁定后继续失败不重置锁定窗口（TTL 保持）。
	ttlBefore := mr.TTL(redisLoginTrackerKey("user@example.com"))
	tracker.RecordFailed(ctx, "user@example.com")
	ttlAfter := mr.TTL(redisLoginTrackerKey("user@example.com"))
	if ttlAfter > ttlBefore+time.Millisecond {
		t.Fatalf("expected lock TTL not extended: before=%v after=%v", ttlBefore, ttlAfter)
	}

	// 成功登录清除计数。
	tracker.Clear(ctx, "user@example.com")
	if err := tracker.Check(ctx, "user@example.com"); err != nil {
		t.Fatalf("expected unlocked after clear, got %v", err)
	}
}

func TestRedisLoginTrackerTTLExpiresLock(t *testing.T) {
	tracker, mr := newTestRedisTracker(t)
	ctx := context.Background()

	for i := 0; i < defaultMaxFailures; i++ {
		tracker.RecordFailed(ctx, "user@example.com")
	}
	if err := tracker.Check(ctx, "user@example.com"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected locked, got %v", err)
	}
	if got := mr.TTL(redisLoginTrackerKey("user@example.com")); got <= 0 || got > defaultLockDuration {
		t.Fatalf("expected lock TTL within (0, %v], got %v", defaultLockDuration, got)
	}
	// TTL 到期 → 键过期 → 自动解锁。
	mr.FastForward(defaultLockDuration + time.Second)
	if err := tracker.Check(ctx, "user@example.com"); err != nil {
		t.Fatalf("expected auto unlock after TTL, got %v", err)
	}
}
