package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisLoginTrackerSetLimitsCov(t *testing.T) {
	tracker, _ := newTestRedisTracker(t)
	ctx := context.Background()
	tracker.SetLimits(2, time.Minute)
	tracker.RecordFailed(ctx, "cov_limited@example.com")
	if err := tracker.Check(ctx, "cov_limited@example.com"); err != nil {
		t.Fatalf("1 failure must not lock: %v", err)
	}
	tracker.RecordFailed(ctx, "cov_limited@example.com")
	if err := tracker.Check(ctx, "cov_limited@example.com"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("2nd failure must lock with configured threshold, got %v", err)
	}

	// 零值回退默认阈值。
	tracker.SetLimits(0, 0)
	for i := 0; i < defaultMaxFailures; i++ {
		tracker.RecordFailed(ctx, "cov_default@example.com")
	}
	if err := tracker.Check(ctx, "cov_default@example.com"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("default threshold must lock after %d failures, got %v", defaultMaxFailures, err)
	}
}

func TestRedisLoginTrackerUnavailableFailsOpenCov(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	tracker := NewRedisLoginTracker(rdb)
	ctx := context.Background()

	// 先正常计数，再关闭 Redis：Check 必须 fail-open（返回 nil），
	// RecordFailed/Clear 只告警不 panic。
	tracker.RecordFailed(ctx, "cov_failopen@example.com")
	mr.Close()

	if err := tracker.Check(ctx, "cov_failopen@example.com"); err != nil {
		t.Fatalf("redis unavailable must fail open, got %v", err)
	}
	tracker.RecordFailed(ctx, "cov_failopen@example.com")
	tracker.Clear(ctx, "cov_failopen@example.com")
	tracker.SetLimits(3, time.Minute)
	tracker.SetLimits(0, 0)
}
