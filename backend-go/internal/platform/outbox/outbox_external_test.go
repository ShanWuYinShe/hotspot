package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

// fakeOutboxQueryable 在内存里模拟 outbox 的领取/租约/结果写回 SQL，用于不依赖
// 数据库地验证 dispatcher 的编排逻辑（真实 SQL 语义由 outbox_external_it_test.go
// 的真实库用例覆盖）：Query 按领取 SQL 返回预置行，QueryRow 模拟事务外结果写回
// 的 CTE 计数，Exec/QueryRow 都记录到 calls 供断言。
type fakeOutboxQueryable struct {
	mu        sync.Mutex
	inline    []outboxRow
	external  []outboxRow
	casResult int64 // 事务外结果写回的影响行数（默认 1；0 模拟租约过期/CAS 落空）
	calls     []fakeOutboxCall
	claimErr  error // 模拟 inline 领取 SQL 失败
	// externalClaimErr 模拟事务外领取 SQL 失败（inline 领取仍正常）。
	externalClaimErr error
}

type fakeOutboxCall struct {
	query string
	args  []any
}

func (f *fakeOutboxQueryable) Exec(_ context.Context, query string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeOutboxCall{query: query, args: append([]any(nil), args...)})
	return nil
}

func (f *fakeOutboxQueryable) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var rows []outboxRow
	switch query {
	case queryClaimOutbox:
		if f.claimErr != nil {
			return nil, f.claimErr
		}
		rows = append([]outboxRow(nil), f.inline...)
	case queryClaimOutboxExternal:
		if f.externalClaimErr != nil {
			return nil, f.externalClaimErr
		}
		rows = append([]outboxRow(nil), f.external...)
	default:
		return nil, fmt.Errorf("fakeOutboxQueryable: unexpected query %q", query)
	}
	if len(args) > 0 {
		if limit, ok := args[0].(int); ok && limit < len(rows) {
			rows = rows[:limit]
		}
	}
	return &fakeOutboxRows{rows: rows}, nil
}

func (f *fakeOutboxQueryable) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakeOutboxCall{query: query, args: append([]any(nil), args...)})
	return fakeOutboxRowResult{updated: f.casResult}
}

// callsMatching 返回记录中 SQL 等于 query 的调用（含参数）。
func (f *fakeOutboxQueryable) callsMatching(query string) []fakeOutboxCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var matched []fakeOutboxCall
	for _, call := range f.calls {
		if call.query == query {
			matched = append(matched, call)
		}
	}
	return matched
}

type fakeOutboxRows struct {
	rows []outboxRow
	idx  int
}

func (r *fakeOutboxRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeOutboxRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	if len(dest) != 4 {
		return fmt.Errorf("fakeOutboxRows: unexpected scan dest count %d", len(dest))
	}
	*(dest[0].(*int64)) = row.eventID
	*(dest[1].(*string)) = row.topic
	*(dest[2].(*[]byte)) = append([]byte(nil), row.payload...)
	*(dest[3].(*int)) = row.attempts
	return nil
}

func (r *fakeOutboxRows) Close() error { return nil }
func (r *fakeOutboxRows) Err() error   { return nil }

type fakeOutboxRowResult struct{ updated int64 }

func (r fakeOutboxRowResult) Scan(dest ...any) error {
	*(dest[0].(*int64)) = r.updated
	return nil
}

// newFakeDispatcher 构造一个只走内存假实现的 dispatcher：factory 非 nil（
// DrainOnce 的前置检查）但无 DB，事务与连接都由 ctx 里注入的 fakeOutboxQueryable
// 承担（RunInTransaction 检测到 ctx 已有事务时直接复用）。
func newFakeDispatcher(t *testing.T, fake *fakeOutboxQueryable) (*Dispatcher, context.Context) {
	t.Helper()
	dispatcher := NewDispatcher(&persistence.Factory{}, nil)
	t.Cleanup(dispatcher.Wait) // 失败时也等异步 handler 结束，避免 goroutine 泄漏
	return dispatcher, persistence.WithTx(context.Background(), fake, nil)
}

func externalTestRow(eventID int64, topic string) outboxRow {
	return outboxRow{eventID: eventID, topic: topic, payload: json.RawMessage(payloadJSON)}
}

// TestDispatcherRegisterExternalPartitionsTopics 锁定注册语义：RegisterExternal
// 的 topic 从事务外路径领取；重新用 Register 注册同一 topic 时回到事务内路径
// （external 标记被清除），两种路径共用同一 handler 表。
func TestDispatcherRegisterExternalPartitionsTopics(t *testing.T) {
	handler := func(context.Context, json.RawMessage) error { return nil }
	dispatcher := NewDispatcher(nil, nil)
	dispatcher.Register("inline.topic", handler)
	dispatcher.RegisterExternal("slow.topic", handler)

	if got := dispatcher.externalTopics(); len(got) != 1 || got[0] != "slow.topic" {
		t.Fatalf("external topics = %v, want [slow.topic]", got)
	}
	if dispatcher.handlerFor("slow.topic") == nil {
		t.Fatal("RegisterExternal 必须同时登记 handler")
	}

	dispatcher.Register("slow.topic", handler)
	if got := dispatcher.externalTopics(); len(got) != 0 {
		t.Fatalf("Register 覆盖后 external topics = %v, want 空", got)
	}
}

// TestDispatcherExternalSlotReservation 锁定事务外并发槽位：一次预留全部空闲
// 槽位、满额后返回 0、释放后可再次预留，最终可全部归还。
func TestDispatcherExternalSlotReservation(t *testing.T) {
	dispatcher := NewDispatcher(nil, nil)
	if got := dispatcher.reserveExternalSlots(); got != externalConcurrency {
		t.Fatalf("首次预留 = %d, want %d", got, externalConcurrency)
	}
	if got := dispatcher.reserveExternalSlots(); got != 0 {
		t.Fatalf("槽位已满时预留 = %d, want 0", got)
	}
	dispatcher.releaseExternalSlots(1)
	if got := dispatcher.reserveExternalSlots(); got != 1 {
		t.Fatalf("释放 1 个槽位后预留 = %d, want 1", got)
	}
	dispatcher.releaseExternalSlots(1)
	dispatcher.releaseExternalSlots(externalConcurrency - 1)
	if got := dispatcher.reserveExternalSlots(); got != externalConcurrency {
		t.Fatalf("全部归还后预留 = %d, want %d", got, externalConcurrency)
	}
}

// TestDrainOnceExternalDoesNotBlockInline 覆盖核心隔离语义：事务外 topic 的
// handler 阻塞期间，DrainOnce 立即返回、同轮其它 topic（事务内路径）照常投递；
// handler 结束后再用短事务写回成功结果。旧实现（handler 在领取事务内执行）会
// 让 DrainOnce 一直阻塞到 handler 返回。
func TestDrainOnceExternalDoesNotBlockInline(t *testing.T) {
	const slowTopic, inlineTopic = "test.external.slow", "test.inline.fast"
	fake := &fakeOutboxQueryable{
		external:  []outboxRow{externalTestRow(101, slowTopic)},
		inline:    []outboxRow{externalTestRow(102, inlineTopic)},
		casResult: 1,
	}
	dispatcher, ctx := newFakeDispatcher(t, fake)

	started := make(chan struct{})
	release := make(chan struct{})
	var (
		startedOnce  sync.Once
		releaseOnce  sync.Once
		externalDone atomic.Bool
	)
	// 失败时也必须放行 handler，否则 newFakeDispatcher 的 Wait 清理会挂到测试超时。
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	dispatcher.RegisterExternal(slowTopic, func(context.Context, json.RawMessage) error {
		startedOnce.Do(func() { close(started) })
		<-release
		externalDone.Store(true)
		return nil
	})
	var inlineCalls atomic.Int32
	dispatcher.Register(inlineTopic, func(context.Context, json.RawMessage) error {
		inlineCalls.Add(1)
		return nil
	})

	start := time.Now()
	processed, err := dispatcher.DrainOnce(ctx)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if processed != 2 {
		t.Fatalf("processed = %d, want 2", processed)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("DrainOnce 被事务外 handler 阻塞 %v：慢 handler 不得在领取/轮询路径内同步执行", elapsed)
	}
	if got := inlineCalls.Load(); got != 1 {
		t.Fatalf("同轮事务内 topic 投递次数 = %d, want 1", got)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("事务外 handler 未被启动")
	}
	// handler 阻塞期间：行已打租约（available_at 推到未来），结果尚未写回。
	if calls := fake.callsMatching(queryLeaseOutboxEvent); len(calls) != 1 {
		t.Fatalf("租约更新次数 = %d, want 1", len(calls))
	}
	if calls := fake.callsMatching(queryMarkLeasedOutboxProcessed); len(calls) != 0 {
		t.Fatalf("handler 未结束前不得写回结果，got %d 次", len(calls))
	}
	if externalDone.Load() {
		t.Fatal("handler 不应在同一轮 DrainOnce 内同步完成")
	}

	releaseOnce.Do(func() { close(release) })
	dispatcher.Wait()
	if !externalDone.Load() {
		t.Fatal("handler 未被正常执行完成")
	}
	success := fake.callsMatching(queryMarkLeasedOutboxProcessed)
	if len(success) != 1 {
		t.Fatalf("成功结果写回次数 = %d, want 1", len(success))
	}
	// 参数：event_id, 新 attempts(=1), 领取时 attempts(=0) 做 CAS。
	args := success[0].args
	if len(args) != 3 || args[0] != int64(101) || args[1] != 1 || args[2] != 0 {
		t.Fatalf("成功写回参数 = %v, want [101 1 0]", args)
	}
}

// TestDrainOnceExternalFailureSchedulesRetryAndDeadLetter 锁定事务外路径的失败
// 语义与事务内路径一致：attempts+1、指数退避重排、last_error 记录错误；达到
// maxAttempts 后进死信（processed_at 非空）。
func TestDrainOnceExternalFailureSchedulesRetryAndDeadLetter(t *testing.T) {
	const topic = "test.external.fail"
	boom := fmt.Errorf("s3 delete boom")

	t.Run("失败按退避重试", func(t *testing.T) {
		fake := &fakeOutboxQueryable{external: []outboxRow{externalTestRow(201, topic)}, casResult: 1}
		dispatcher, ctx := newFakeDispatcher(t, fake)
		dispatcher.RegisterExternal(topic, func(context.Context, json.RawMessage) error { return boom })

		if _, err := dispatcher.DrainOnce(ctx); err != nil {
			t.Fatalf("drain: %v", err)
		}
		dispatcher.Wait()

		retries := fake.callsMatching(queryScheduleLeasedOutboxRetry)
		if len(retries) != 1 {
			t.Fatalf("重试写回次数 = %d, want 1", len(retries))
		}
		// 参数：event_id, 新 attempts(=1), backoff(1<<1=2s), last_error, 领取 attempts(=0)。
		args := retries[0].args
		if len(args) != 5 || args[0] != int64(201) || args[1] != 1 || args[2] != "2s" || args[4] != 0 {
			t.Fatalf("重试写回参数 = %v, want [201 1 2s <err> 0]", args)
		}
		if errText, _ := args[3].(string); !strings.Contains(errText, "s3 delete boom") {
			t.Fatalf("last_error = %q, 应包含 handler 错误", errText)
		}
	})

	t.Run("达到上限进死信", func(t *testing.T) {
		row := externalTestRow(202, topic)
		row.attempts = maxAttempts - 1
		fake := &fakeOutboxQueryable{external: []outboxRow{row}, casResult: 1}
		dispatcher, ctx := newFakeDispatcher(t, fake)
		dispatcher.RegisterExternal(topic, func(context.Context, json.RawMessage) error { return boom })

		if _, err := dispatcher.DrainOnce(ctx); err != nil {
			t.Fatalf("drain: %v", err)
		}
		dispatcher.Wait()
		if calls := fake.callsMatching(queryMarkLeasedOutboxDeadLetter); len(calls) != 1 {
			t.Fatalf("死信写回次数 = %d, want 1", len(calls))
		}
	})
}

// TestDrainOnceExternalCASConflictDropsStaleResult 覆盖租约竞争：事务外 handler
// 结束写回时若行已被重新领取/写回（CAS 影响 0 行），结果按丢弃处理、不返回错误
// （事件会在新一轮正常投递，at-least-once 语义）。
func TestDrainOnceExternalCASConflictDropsStaleResult(t *testing.T) {
	const topic = "test.external.stale"
	fake := &fakeOutboxQueryable{external: []outboxRow{externalTestRow(301, topic)}, casResult: 0}
	dispatcher, ctx := newFakeDispatcher(t, fake)
	dispatcher.RegisterExternal(topic, func(context.Context, json.RawMessage) error { return nil })

	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	dispatcher.Wait()
	if calls := fake.callsMatching(queryMarkLeasedOutboxProcessed); len(calls) != 1 {
		t.Fatalf("CAS 写回尝试次数 = %d, want 1", len(calls))
	}
}

// TestDrainOnceExternalPanicIsolated：事务外 handler panic 必须转为普通失败
// （invokeHandler 兜底），按重试语义写回、不掀进程/不影响其它事件。
func TestDrainOnceExternalPanicIsolated(t *testing.T) {
	const topic = "test.external.panic"
	fake := &fakeOutboxQueryable{external: []outboxRow{externalTestRow(401, topic)}, casResult: 1}
	dispatcher, ctx := newFakeDispatcher(t, fake)
	dispatcher.RegisterExternal(topic, func(context.Context, json.RawMessage) error {
		panic("slow storage blew up")
	})

	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	dispatcher.Wait()
	retries := fake.callsMatching(queryScheduleLeasedOutboxRetry)
	if len(retries) != 1 {
		t.Fatalf("panic 后重试写回次数 = %d, want 1", len(retries))
	}
	if errText, _ := retries[0].args[3].(string); !strings.Contains(errText, "slow storage blew up") {
		t.Fatalf("last_error = %q, 应包含 panic 内容", errText)
	}
}

// TestDrainOnceExternalSlotLimitCapsClaims：事务外事件每轮最多领取「空闲并发
// 槽位」条；槽位占满时后续轮次不再领取（积压行留在 due 状态），handler 结束后
// 释放槽位并继续领取。
func TestDrainOnceExternalSlotLimitCapsClaims(t *testing.T) {
	const topic = "test.external.cap"
	rows := make([]outboxRow, 0, externalConcurrency+2)
	for i := 0; i < externalConcurrency+2; i++ {
		rows = append(rows, externalTestRow(int64(500+i), topic))
	}
	fake := &fakeOutboxQueryable{external: rows, casResult: 1}
	dispatcher, ctx := newFakeDispatcher(t, fake)

	release := make(chan struct{})
	var releaseOnce sync.Once
	// 失败时也必须放行 handler，否则 newFakeDispatcher 的 Wait 清理会挂到测试超时。
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var running atomic.Int32
	dispatcher.RegisterExternal(topic, func(context.Context, json.RawMessage) error {
		running.Add(1)
		<-release
		return nil
	})

	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != externalConcurrency {
		t.Fatalf("首轮 processed=%d err=%v, want %d", processed, err, externalConcurrency)
	}
	waitForCond(t, 2*time.Second, func() bool { return running.Load() == int32(externalConcurrency) },
		fmt.Sprintf("并发执行数未达到 %d：槽位预留必须限制在途 handler 数", externalConcurrency))
	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != 0 {
		t.Fatalf("槽位占满时第二轮 processed=%d err=%v, want 0", processed, err)
	}
	releaseOnce.Do(func() { close(release) })
	dispatcher.Wait()
}

// TestDrainOnceExternalClaimFailureReleasesSlots：事务外领取 SQL 失败（DB 抖动）
// 时必须归还预留的并发槽位，否则一次失败会让并发槽位永久泄漏、慢 topic 停摆。
func TestDrainOnceExternalClaimFailureReleasesSlots(t *testing.T) {
	const topic = "test.external.claimfail"
	fake := &fakeOutboxQueryable{externalClaimErr: errors.New("db down")}
	dispatcher, ctx := newFakeDispatcher(t, fake)
	dispatcher.RegisterExternal(topic, func(context.Context, json.RawMessage) error { return nil })

	if _, err := dispatcher.DrainOnce(ctx); err == nil {
		t.Fatal("领取失败必须返回错误")
	}
	dispatched := dispatcher.reserveExternalSlots()
	if dispatched != externalConcurrency {
		t.Fatalf("领取失败后可用槽位 = %d, want %d（预留槽位必须归还）", dispatched, externalConcurrency)
	}
	dispatcher.releaseExternalSlots(dispatched)
}

// TestRunExternalShutdownKeepsEventForLeaseRetry：父 ctx 取消（停机）时 handler
// 被中断，不把「中断」记成投递失败（不写 attempts/退避），事件保持租约等重启后
// 到期重投——避免反复滚动重启把合法事件推进死信。
func TestRunExternalShutdownKeepsEventForLeaseRetry(t *testing.T) {
	const topic = "test.external.shutdown"
	fake := &fakeOutboxQueryable{external: []outboxRow{externalTestRow(601, topic)}, casResult: 1}
	dispatcher := NewDispatcher(&persistence.Factory{}, nil)
	ctx, cancel := context.WithCancel(persistence.WithTx(context.Background(), fake, nil))
	started := make(chan struct{})
	dispatcher.RegisterExternal(topic, func(handlerCtx context.Context, _ json.RawMessage) error {
		close(started)
		<-handlerCtx.Done()
		return handlerCtx.Err()
	})
	t.Cleanup(dispatcher.Wait)

	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("事务外 handler 未被启动")
	}
	cancel()
	dispatcher.Wait()
	if retries := fake.callsMatching(queryScheduleLeasedOutboxRetry); len(retries) != 0 {
		t.Fatalf("停机中断不得写重试状态，got %d 次", len(retries))
	}
	if successes := fake.callsMatching(queryMarkLeasedOutboxProcessed); len(successes) != 0 {
		t.Fatalf("停机中断不得写成功状态，got %d 次", len(successes))
	}
}

// waitForCond 在 timeout 内轮询 cond（异步 handler 的启动/完成无同步点）。
func waitForCond(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

// TestDispatcherWaitNilAndNoFactory：nil dispatcher 的 Wait/DrainOnce 降级语义。
func TestDispatcherWaitNilAndNoFactory(t *testing.T) {
	var nilDispatcher *Dispatcher
	nilDispatcher.Wait()
	if _, err := nilDispatcher.DrainOnce(context.Background()); err == nil {
		t.Fatal("nil dispatcher 的 DrainOnce 必须返回错误")
	}
	if _, err := NewDispatcher(nil, nil).DrainOnce(context.Background()); err == nil {
		t.Fatal("无 factory 的 DrainOnce 必须返回错误")
	}
}
