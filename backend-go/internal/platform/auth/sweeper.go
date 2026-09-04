package auth

import (
	"context"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/worker"
)

// authSweepCadence is how often the in-memory session/refresh stores get
// physically pruned. Reads already apply lazy expiry (so correctness does
// not depend on the sweeper); the sweeper just stops the maps from growing
// forever in long-lived processes that never read a given key again.
const authSweepCadence = 10 * time.Minute

var (
	authSweeperMu      sync.Mutex
	authSweeperCancel  context.CancelFunc
	authSweeperStarted bool
)

// StartSweeper launches a background goroutine that periodically calls
// Sweep on the default service's in-memory session and refresh stores, if
// those stores are the in-memory implementations. It is safe to call from
// bootstrap; calling it again while a sweeper is already running is a
// no-op. The goroutine exits when ctx is cancelled.
//
// If the default service uses Postgres stores (production wiring) this is a
// cheap no-op on each tick.
func StartSweeper(ctx context.Context) {
	authSweeperMu.Lock()
	defer authSweeperMu.Unlock()
	if authSweeperStarted {
		return
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	authSweeperCancel = cancel
	authSweeperStarted = true
	worker.Go("platformauth.sweeper", func() { runAuthSweeper(sweepCtx) })
}

// StopSweeper cancels any running sweeper and is intended for tests.
func StopSweeper() {
	authSweeperMu.Lock()
	if authSweeperCancel == nil {
		authSweeperMu.Unlock()
		return
	}
	authSweeperCancel()
	authSweeperCancel = nil
	authSweeperStarted = false
	authSweeperMu.Unlock()
}

func runAuthSweeper(ctx context.Context) {
	defer func() {
		authSweeperMu.Lock()
		authSweeperCancel = nil
		authSweeperStarted = false
		authSweeperMu.Unlock()
	}()
	ticker := time.NewTicker(authSweepCadence)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepDefaultStores(time.Now().UTC())
		}
	}
}

func sweepDefaultStores(now time.Time) {
	svc := DefaultService()
	if svc == nil {
		return
	}
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
