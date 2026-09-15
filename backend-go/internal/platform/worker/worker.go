// Package worker provides a panic-safe goroutine launcher for long-lived
// background workers (sweepers, bridges, fan-out loops).
//
// Background goroutines are not covered by the HTTP Recoverer middleware, so
// an unrecovered panic in one of them crashes the whole process. Go starts a
// worker with worker.Go so the panic is caught, logged with a stack trace,
// and the process stays up.
package worker

import (
	"runtime/debug"
	"time"

	"hotspot/backend-go/internal/platform/observability"
)

// Go starts fn on a new goroutine whose panics are recovered and logged.
// Use it for any background worker that would otherwise take the process down
// on a panic (sweepers, pub/sub receive loops, etc.).
//
// Recovering means the goroutine exits cleanly rather than crashing the
// process; callers that need the worker to keep running should add their own
// restart loop around the work.
func Go(name string, fn func()) {
	go func() {
		defer Recover(name)
		fn()
	}()
}

// GoRestartable is the panic self-healing variant of Go for long-lived loop
// workers: when fn panics it is logged and restarted with exponential backoff
// (1s 起步，封顶 30s)；fn 干净返回（如 ctx 取消触发的停机退出）不重启。
// 此前单循环长驻 sweeper（通知清理、限流清理等）一次未预期 panic 即永久
// 停转——例如通知 prune 停转会让 notifications 表恢复无限增长。
func GoRestartable(name string, fn func()) {
	go func() {
		backoff := time.Second
		for {
			crashed := runRecovered(name, fn)
			if !crashed {
				return
			}
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}()
}

// runRecovered runs fn once; reports whether it exited via panic.
func runRecovered(name string, fn func()) (crashed bool) {
	defer func() {
		if rec := recover(); rec != nil {
			observability.L().Error("background worker panic recovered, restarting",
				"worker", name,
				"panic", rec,
				"stack", string(debug.Stack()),
			)
			crashed = true
		}
	}()
	fn()
	return false
}

// Recover logs a panic (with stack) under the given worker name. Install it as
// the first defer in a worker function:
//
//	defer worker.Recover("auth sweeper")
//
// It is a no-op when there is no panic.
func Recover(name string) {
	if rec := recover(); rec != nil {
		observability.L().Error("background worker panic recovered",
			"worker", name,
			"panic", rec,
			"stack", string(debug.Stack()),
		)
	}
}
