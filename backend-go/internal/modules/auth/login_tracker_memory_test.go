package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 审查 P1-5 防回归：sweeper 只清「已锁定且锁期已过」与「未锁定且静默期
// （loginCounterDecay）已过」的条目。此前裸 now.After(lockUntil) 会把零值
// lockUntil 的未锁定条目全部清光——攻击者以「每清扫周期失败 max-1 次」
// 的节奏永不触发锁定。
func TestInMemoryLoginTrackerSweepKeepsPartialCounters(t *testing.T) {
	tracker := newInMemoryLoginTracker(context.Background())
	ctx := context.Background()

	// 连续失败 4 次（未达阈值），静默仅一个清扫周期（10min < 衰减窗口 15min）：
	// 计数必须保留，第 5 次失败即锁定。
	for i := 0; i < defaultMaxFailures-1; i++ {
		tracker.RecordFailed(ctx, "alice")
	}
	tracker.sweepOnce(time.Now().Add(defaultTrackerCleanup + time.Second))
	tracker.RecordFailed(ctx, "alice")
	if err := tracker.Check(ctx, "alice"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected lock after sweep + 5th failure, got %v", err)
	}
}

func TestInMemoryLoginTrackerSweepDecaysSilentCounters(t *testing.T) {
	tracker := newInMemoryLoginTracker(context.Background())
	ctx := context.Background()

	tracker.RecordFailed(ctx, "bob")
	// 静默超过衰减窗口：计数作废，条目被清（防无界增长）。
	tracker.sweepOnce(time.Now().Add(loginCounterDecay + time.Second))
	if err := tracker.Check(ctx, "bob"); err != nil {
		t.Fatalf("expected no lock after decay, got %v", err)
	}
	tracker.RecordFailed(ctx, "bob")
	if err := tracker.Check(ctx, "bob"); err != nil {
		t.Fatalf("expected counter reset to 1 after decay, got %v", err)
	}
}

func TestInMemoryLoginTrackerSweepKeepsActiveLocks(t *testing.T) {
	tracker := newInMemoryLoginTracker(context.Background())
	ctx := context.Background()

	for i := 0; i < defaultMaxFailures; i++ {
		tracker.RecordFailed(ctx, "carol")
	}
	// 锁定期内：条目必须保留（锁定不被清扫提前释放）。
	tracker.sweepOnce(time.Now().Add(time.Minute))
	if err := tracker.Check(ctx, "carol"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected active lock preserved by sweep, got %v", err)
	}
	// 锁期已过：条目清除，计数随之作废。
	tracker.sweepOnce(time.Now().Add(defaultLockDuration + time.Second))
	if err := tracker.Check(ctx, "carol"); err != nil {
		t.Fatalf("expected lock released after expiry, got %v", err)
	}
}
