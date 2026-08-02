package worker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestGo_RecoversPanic verifies a panicking worker does not crash the process
// (the test process itself) and that siblings keep running.
func TestGo_RecoversPanic(t *testing.T) {
	var ran atomic.Int32
	var wg sync.WaitGroup
	wg.Add(2)
	Go("panicker", func() {
		defer wg.Done()
		panic("boom")
	})
	Go("sibling", func() {
		defer wg.Done()
		ran.Store(1)
	})
	wg.Wait()
	if ran.Load() != 1 {
		t.Fatal("sibling worker did not complete after panicker recovered")
	}
}

// TestRecover_NoPanic verifies Recover is a no-op when no panic occurred.
func TestRecover_NoPanic(t *testing.T) {
	// Just confirm it doesn't panic when called directly with no panic in flight.
	done := make(chan struct{})
	go func() {
		defer Recover("noop")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Recover blocked on a non-panicking goroutine")
	}
}
