package resourceguard

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

// StartSweeper 负责物理清理 missing 缓存（否则随机 ID 探测可让 map 无界增长）。
// 此前无测试；锁定：过期条目被清扫、ctx 取消后循环退出（不再清理）。
func TestDBCheckerStartSweeperExpiresMissingCacheAndStops(t *testing.T) {
	db := &mockQueryableDB{
		queryRowFunc: func(context.Context, string, ...any) persistence.Row {
			return &mockRow{err: pgx.ErrNoRows}
		},
	}
	checker := NewDBChecker(db)
	checker.ttl = 20 * time.Millisecond

	ctx := context.Background()
	checker.StartSweeper(ctx, 10*time.Millisecond)
	checker.StartSweeper(ctx, 10*time.Millisecond) // 重复启动为 no-op

	allowed, err := checker.MayExist(ctx, ResourceUser, "usr_cache_probe")
	if err != nil || allowed {
		t.Fatalf("MayExist missing = %v/%v, want false/nil", allowed, err)
	}
	missingCount := func() int {
		checker.mu.RLock()
		defer checker.mu.RUnlock()
		return len(checker.missing)
	}
	if got := missingCount(); got != 1 {
		t.Fatalf("missing cache entries = %d, want 1", got)
	}

	// 过期后由后台清扫移除（惰性删除之外的物理清理）。
	deadline := time.Now().Add(2 * time.Second)
	for missingCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := missingCount(); got != 0 {
		t.Fatalf("sweeper did not expire missing entry: %d", got)
	}

	// StopSweeper 等待循环退出后：新写入的 missing 条目不再被后台清理（仅剩惰性删除路径）。
	checker.StopSweeper()
	checker.StopSweeper() // 幂等
	checker.markMissing(ResourceGroup, "grp_after_stop")
	time.Sleep(60 * time.Millisecond)
	if got := missingCount(); got != 1 {
		t.Fatalf("sweeper still running after ctx cancel: entries=%d, want 1", got)
	}
	// 惰性过期仍生效（不依赖 sweeper）。
	checker.ttl = time.Millisecond
	time.Sleep(5 * time.Millisecond)
	if checker.isMissing(ResourceGroup, "grp_after_stop") {
		t.Fatal("lazy expiry should clear expired missing entry")
	}
}
