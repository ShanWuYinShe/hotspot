package realtime

import (
	"sync"
	"testing"
)

// TestSetDefaultHub_NilIgnored verifies a nil argument does not clear the
// singleton (which would leave DefaultHub() returning nil and panic callers).
func TestSetDefaultHub_NilIgnored(t *testing.T) {
	original := DefaultHub()
	SetDefaultHub(nil)
	if DefaultHub() != original {
		t.Fatal("DefaultHub changed after SetDefaultHub(nil)")
	}
}

// TestSetDefaultHub_ReplacesHub verifies SetDefaultHub swaps the singleton
// and the new instance is observable via DefaultHub().
func TestSetDefaultHub_ReplacesHub(t *testing.T) {
	t.Cleanup(ResetHub)
	original := DefaultHub()
	fresh := NewHub()
	SetDefaultHub(fresh)
	if DefaultHub() != fresh {
		t.Fatal("DefaultHub did not return the newly set hub")
	}
	if DefaultHub() == original {
		t.Fatal("DefaultHub still returns the old hub")
	}
}

// TestResetHub_ReplacesWithFreshInstance verifies ResetHub installs a new hub.
func TestResetHub_ReplacesWithFreshInstance(t *testing.T) {
	before := DefaultHub()
	ResetHub()
	if DefaultHub() == before {
		t.Fatal("ResetHub did not replace the hub")
	}
}

// TestDefaultHubConcurrentAccess exercises concurrent reads and writes of the
// singleton. Run with `go test -race` to detect the data race that a plain
// `var` (non-atomic) implementation would have.
func TestDefaultHubConcurrentAccess(t *testing.T) {
	t.Cleanup(ResetHub)

	var wg sync.WaitGroup
	const readers, writers = 8, 4
	// Bounded iterations keep the race-detector run fast while still
	// overlapping reads (DefaultHub + BroadcastToUser) with writes
	// (SetDefaultHub/ResetHub) on the singleton.
	const readIters, writeIters = 200, 50

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < readIters; j++ {
				DefaultHub().BroadcastToUser("u", Event{Type: "test"})
			}
		}()
	}
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < writeIters; j++ {
				SetDefaultHub(NewHub())
				ResetHub()
			}
		}()
	}
	wg.Wait()
}
