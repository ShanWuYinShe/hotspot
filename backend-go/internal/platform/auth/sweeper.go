package auth

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/worker"
)

// authSweepCadence is how often the in-memory session/refresh stores get
// physically pruned. Reads already apply lazy expiry (so correctness does
// not depend on the sweeper); the sweeper just stops the maps from growing
// forever in long-lived processes that never read a given key again.
const authSweepCadence = 10 * time.Minute

// StartSweeper 为显式传入的 Service 启动后台清扫（不再依赖进程级
// DefaultService）：内存 session/refresh store 物理清理；Postgres store 走
// 保留期清理。返回幂等的 stop。
func StartSweeper(ctx context.Context, svc *Service) (stop func()) {
	if svc == nil {
		return func() {}
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	worker.GoRestartable("platformauth.sweeper", func() { runAuthSweeper(sweepCtx, svc) })
	return cancel
}

func runAuthSweeper(ctx context.Context, svc *Service) {
	ticker := time.NewTicker(authSweepCadence)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepStores(svc, time.Now().UTC())
		}
	}
}

func sweepStores(svc *Service, now time.Time) {
	if svc == nil {
		return
	}
	svc.sweepBanCache(now)
	if s, ok := svc.sessions.(*InMemorySessionStore); ok {
		s.Sweep(now)
	}
	if r, ok := svc.refreshes.(*InMemoryRefreshStore); ok {
		r.Sweep(now)
	}
	if r, ok := svc.refreshes.(*PostgresRefreshStore); ok {
		// PG 版清理此前缺失：auth_refresh_tokens 每次轮换新增一行且永不删除，
		// 表随登录/轮换无限增长。短超时上下文：清理失败静默等下一轮。
		sweepCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r.Sweep(sweepCtx, now)
	}
}
