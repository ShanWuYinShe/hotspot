package worker

import (
	"sync/atomic"
	"testing"
	"time"
)

// setTestBackoff 把退避参数临时缩短到毫秒级（生产默认 1s/30s），测试结束恢复。
func setTestBackoff(t *testing.T, initial, max time.Duration) {
	t.Helper()
	oldInitial, oldMax := restartBackoffInitial, restartBackoffMax
	restartBackoffInitial, restartBackoffMax = initial, max
	t.Cleanup(func() {
		restartBackoffInitial, restartBackoffMax = oldInitial, oldMax
	})
}

func TestGoRestartableRestartsAfterPanicAndStopsOnCleanReturn(t *testing.T) {
	setTestBackoff(t, 10*time.Millisecond, 20*time.Millisecond)
	var calls atomic.Int32
	done := make(chan struct{})
	start := time.Now()
	GoRestartable("test.restartable", func() {
		switch calls.Add(1) {
		case 1, 2:
			panic("boom")
		default:
			close(done)
		}
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("worker did not finish after restarts, calls=%d", calls.Load())
	}
	elapsed := time.Since(start)
	if got := calls.Load(); got != 3 {
		t.Fatalf("worker calls = %d, want 3 (panic→restart→panic→restart→clean)", got)
	}
	// 两次 panic 后分别退避 10ms、20ms；给调度留余量只做下界断言。
	if elapsed < 25*time.Millisecond {
		t.Fatalf("restart backoff not applied: elapsed=%v", elapsed)
	}
	// 干净返回后不得再重启（否则长驻 worker 的停机路径会变成忙循环）。
	time.Sleep(100 * time.Millisecond)
	if got := calls.Load(); got != 3 {
		t.Fatalf("worker restarted after clean return: calls=%d", got)
	}
}

func TestGoRestartableBackoffIsCapped(t *testing.T) {
	// max 低于 initial 的两倍：验证封顶生效（第二次退避取 max 而非 2*initial）。
	setTestBackoff(t, 10*time.Millisecond, 15*time.Millisecond)
	var calls atomic.Int32
	done := make(chan struct{})
	start := time.Now()
	GoRestartable("test.capped", func() {
		switch calls.Add(1) {
		case 1, 2:
			panic("boom")
		default:
			close(done)
		}
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("worker did not finish, calls=%d", calls.Load())
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
	// 10ms + 15ms（封顶，而非 20ms）
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("elapsed=%v, capped backoff not applied", elapsed)
	}
}

func TestRunRecoveredReportsPanic(t *testing.T) {
	if crashed := runRecovered("test.clean", func() {}); crashed {
		t.Fatal("clean return must report crashed=false")
	}
	if crashed := runRecovered("test.panic", func() { panic("boom") }); !crashed {
		t.Fatal("panic must report crashed=true")
	}
}
