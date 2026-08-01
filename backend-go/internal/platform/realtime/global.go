package realtime

import "sync/atomic"

// defaultHub is the process-wide singleton Hub. It is read on every broadcast
// and WS connection, and swapped by tests (SetDefaultHub/ResetHub), so it is
// stored as an atomic pointer to avoid a data race between concurrent reads
// and writes — matching the pattern used by platformauth.defaultService,
// idempotency.defaultStore and resourceguard.defaultChecker.
var defaultHub atomic.Pointer[Hub]

func init() {
	defaultHub.Store(NewHub())
}

// SetDefaultHub replaces the default hub. A nil hub is ignored so callers
// can't accidentally clear the singleton.
func SetDefaultHub(hub *Hub) {
	if hub != nil {
		defaultHub.Store(hub)
	}
}

// DefaultHub returns the process-wide hub. Every caller reads the current
// pointer atomically, so it is safe to call concurrently with SetDefaultHub.
func DefaultHub() *Hub {
	return defaultHub.Load()
}

// ResetHub replaces the default hub with a fresh instance.
// Intended for use in test suites to ensure test isolation.
func ResetHub() { defaultHub.Store(NewHub()) }
