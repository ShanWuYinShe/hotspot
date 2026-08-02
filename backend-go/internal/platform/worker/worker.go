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
