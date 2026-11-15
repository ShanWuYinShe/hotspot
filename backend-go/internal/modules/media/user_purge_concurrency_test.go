package media

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// purgeConcurrencyProvider 记录 Delete 的最大并发度、调用次数，并可注入取消
// 回调与单条耗时，用于验证 DeletePurgeObjects 的有界并发与 ctx 响应。
type purgeConcurrencyProvider struct {
	*coverageProvider
	mu          sync.Mutex
	inFlight    int
	maxInFlight int
	calls       int
	hold        time.Duration
	cancel      context.CancelFunc
}

func (p *purgeConcurrencyProvider) Delete(ctx context.Context, _ string) error {
	p.mu.Lock()
	p.calls++
	p.inFlight++
	if p.inFlight > p.maxInFlight {
		p.maxInFlight = p.inFlight
	}
	cancel := p.cancel
	hold := p.hold
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.inFlight--
		p.mu.Unlock()
	}()
	if cancel != nil {
		cancel()
	}
	if hold > 0 {
		select {
		case <-time.After(hold):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return ctx.Err()
}

func (p *purgeConcurrencyProvider) stats() (calls, maxInFlight int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls, p.maxInFlight
}

// TestDeletePurgeObjectsBoundedConcurrency 锁定单事件内删除的并发上限：handler
// 在 dispatcher 事务外执行，200 键全串行时单条 10s 超时会把单事件拖到 2000s；
// 有界并发（purgeDeleteConcurrency）在不打爆对象存储的前提下压短最坏时长。
func TestDeletePurgeObjectsBoundedConcurrency(t *testing.T) {
	provider := &purgeConcurrencyProvider{coverageProvider: &coverageProvider{}, hold: 2 * time.Millisecond}
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider)
	keys := make([]string, 0, 2*purgeDeleteConcurrency)
	for i := 0; i < 2*purgeDeleteConcurrency; i++ {
		keys = append(keys, fmt.Sprintf("published/u1/%d.jpg", i))
	}
	if err := svc.DeletePurgeObjects(context.Background(), "u1", keys); err != nil {
		t.Fatalf("delete purge objects: %v", err)
	}
	calls, maxInFlight := provider.stats()
	if calls != len(keys) {
		t.Fatalf("Delete 调用次数 = %d, want %d", calls, len(keys))
	}
	if maxInFlight > purgeDeleteConcurrency {
		t.Fatalf("并发度 = %d, 超过上限 %d", maxInFlight, purgeDeleteConcurrency)
	}
	if maxInFlight != purgeDeleteConcurrency {
		t.Fatalf("并发度 = %d, want %d（实现必须真正并发，不能退化为串行）", maxInFlight, purgeDeleteConcurrency)
	}
}

// TestDeletePurgeObjectsStopsOnCanceledContext 锁定单事件总超时/停机响应：ctx
// 取消后不再发起新批，返回错误让 outbox 重试（已删对象 404 幂等），剩余键不会
// 被继续无意义地打向对象存储。
func TestDeletePurgeObjectsStopsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &purgeConcurrencyProvider{coverageProvider: &coverageProvider{}, cancel: cancel}
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider)
	keys := make([]string, 0, 3*purgeDeleteChunkSize)
	for i := 0; i < 3*purgeDeleteChunkSize; i++ {
		keys = append(keys, fmt.Sprintf("published/u1/%d.jpg", i))
	}
	err := svc.DeletePurgeObjects(ctx, "u1", keys)
	if err == nil {
		t.Fatal("ctx 取消后必须返回错误供 outbox 重试")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled 包装", err)
	}
	calls, _ := provider.stats()
	if calls > purgeDeleteChunkSize {
		t.Fatalf("取消后仍发起 %d 次删除，超过单批 %d：批间必须检查 ctx", calls, purgeDeleteChunkSize)
	}
}

// TestDeletePurgeObjectsHonorsEventTimeout 锁定单事件总超时语义：ctx 超时后
// 不再发起新批，handler 返回 DeadlineExceeded 错误（dispatcher 据此按退避重试，
// 达到 maxAttempts 进死信），不会无限串行。
func TestDeletePurgeObjectsHonorsEventTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	provider := &purgeConcurrencyProvider{coverageProvider: &coverageProvider{}, hold: 10 * time.Millisecond}
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider)
	keys := make([]string, 0, 3*purgeDeleteChunkSize)
	for i := 0; i < 3*purgeDeleteChunkSize; i++ {
		keys = append(keys, fmt.Sprintf("published/u1/%d.jpg", i))
	}
	err := svc.DeletePurgeObjects(ctx, "u1", keys)
	if err == nil {
		t.Fatal("ctx 超时后必须返回错误供 outbox 重试/死信")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded 包装", err)
	}
	calls, _ := provider.stats()
	if calls > purgeDeleteChunkSize {
		t.Fatalf("超时后仍发起 %d 次删除，超过单批 %d：批间必须检查 ctx", calls, purgeDeleteChunkSize)
	}
}
