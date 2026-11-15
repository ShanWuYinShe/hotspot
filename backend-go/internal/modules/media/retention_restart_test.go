package media

import (
	"context"
	"sync"
	"testing"
	"time"
)

// panickingRetentionRepo 在第一次 ListStaleUploads 时 panic，之后正常返回并记录
// 每轮看到的 ctx 是否已取消：用于模拟 worker.GoRestartable 的 panic 重启，验证
// 重启后的 sweep 不再复用「panic 展开时被 defer cancel 的」派生 ctx。
type panickingRetentionRepo struct {
	*InMemoryRepository
	mu        sync.Mutex
	listCalls int
	canceled  bool
	swept     chan struct{}
	sweptOnce sync.Once
}

func (r *panickingRetentionRepo) ListStaleUploads(ctx context.Context, _ time.Time, _ int) ([]pendingUpload, error) {
	r.mu.Lock()
	r.listCalls++
	call := r.listCalls
	r.mu.Unlock()
	if call == 1 {
		panic("injected retention sweep panic")
	}
	if ctx.Err() != nil {
		r.mu.Lock()
		r.canceled = true
		r.mu.Unlock()
		return nil, ctx.Err()
	}
	r.sweptOnce.Do(func() { close(r.swept) })
	return nil, nil
}

func (r *panickingRetentionRepo) snapshot() (calls int, canceled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listCalls, r.canceled
}

// TestRetentionSweeperRestartAfterPanicKeepsSweeping 覆盖第 82 轮审计 P2-4：
// sweeper worker 一旦 panic，worker.GoRestartable 会重新执行同一闭包；旧实现把
// sweepCtx 派生在闭包外并在闭包内 defer cancel，重启后每轮 sweep 都拿到已取消
// 的 ctx（仓储返回 context canceled），未完成上传永不回收。改造后 ctx/ticker 在
// 每次运行时创建，panic 重启后仍能正常清扫。
func TestRetentionSweeperRestartAfterPanicKeepsSweeping(t *testing.T) {
	// 全局幂等标志按测试起点复位（同包其它用例不启动 sweeper，此处防御性重置）。
	retentionSweepMu.Lock()
	retentionSweepStarted = false
	retentionSweepMu.Unlock()

	repo := &panickingRetentionRepo{InMemoryRepository: NewInMemoryRepository(), swept: make(chan struct{})}
	svc := NewServiceWithRepoAndProvider(repo, &retentionProvider{failOn: map[string]error{}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartRetentionSweeper(ctx)

	// 第一次 sweep panic → GoRestartable 退避 1s 后重启 → 第二次 sweep 必须用
	// 未取消的 ctx 正常执行（旧实现会因 ctx 已取消直接跳过清扫）。
	select {
	case <-repo.swept:
	case <-time.After(10 * time.Second):
		calls, canceled := repo.snapshot()
		t.Fatalf("panic 重启后的 sweeper 未再执行（listCalls=%d ctxCanceled=%v）：重启不得复用已取消的 ctx", calls, canceled)
	}
	if calls, canceled := repo.snapshot(); canceled {
		t.Fatalf("panic 重启后的 sweep 收到已取消的 ctx（listCalls=%d）：ctx 必须在每次运行时重新创建", calls)
	}

	// 停机：ctx 取消后 worker 干净退出并复位幂等标志；之后再调用可正常再启动。
	cancel()
	waitForMediaCond(t, 5*time.Second, func() bool {
		retentionSweepMu.Lock()
		defer retentionSweepMu.Unlock()
		return !retentionSweepStarted
	}, "ctx 取消后 sweeper 未退出（幂等标志未复位）")
}

// waitForMediaCond 在 timeout 内轮询 cond。
func waitForMediaCond(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(msg)
}
