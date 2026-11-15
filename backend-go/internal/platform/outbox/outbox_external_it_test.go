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
	"hotspot/backend-go/internal/platform/testdb"
)

// 真实库用例：事务外执行（RegisterExternal）的隔离语义。覆盖第 82 轮审计
// P1-4 的两个目标：慢 handler（对象存储批量删除）不放在领取事务内、也不阻塞
// 其它 topic 投递。
func TestPostgresIT_OutboxExternalSlowHandlerDoesNotBlockOtherTopics(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	suffix := time.Now().UnixNano()
	slowTopic := fmt.Sprintf("it_outbox_ext_slow_%d", suffix)
	fastTopic := fmt.Sprintf("it_outbox_ext_fast_%d", suffix)
	t.Cleanup(func() {
		_ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = ANY($1)", []string{slowTopic, fastTopic})
	})

	writer := NewWriter(factory)
	dispatcher := NewDispatcher(factory, nil)
	// LIFO 清理顺序：先放行阻塞的 handler，再等待其结束，最后由上面的 cleanup 删行。
	t.Cleanup(dispatcher.Wait)
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var (
		startedOnce    sync.Once
		slowFinished   atomic.Bool
		fastCalls      atomic.Int32
		slowSawNoTxCtx atomic.Bool
	)
	dispatcher.RegisterExternal(slowTopic, func(handlerCtx context.Context, _ json.RawMessage) error {
		if _, hasTx := persistence.QueryableFromContext(handlerCtx); hasTx {
			slowSawNoTxCtx.Store(true)
		}
		startedOnce.Do(func() { close(started) })
		select {
		case <-release:
		case <-time.After(10 * time.Second): // 测试失败兜底：不让 handler 永久阻塞
		}
		slowFinished.Store(true)
		return nil
	})
	dispatcher.Register(fastTopic, func(context.Context, json.RawMessage) error {
		fastCalls.Add(1)
		return nil
	})

	// 慢 topic 事件先入队（event_id 更小，旧实现会占据队列头部并阻塞后续事件）。
	if err := writer.Enqueue(ctx, slowTopic, map[string]any{"k": "slow"}); err != nil {
		t.Fatalf("enqueue slow event: %v", err)
	}
	if err := writer.Enqueue(ctx, fastTopic, map[string]any{"k": "fast-1"}); err != nil {
		t.Fatalf("enqueue fast event: %v", err)
	}

	start := time.Now()
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Fatalf("DrainOnce 被事务外 handler 阻塞 %v：S3 类慢 handler 不得在领取/轮询路径内同步执行", elapsed)
	}
	if got := fastCalls.Load(); got != 1 {
		t.Fatalf("慢 handler 期间事务内 topic 投递次数 = %d, want 1", got)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("事务外 handler 未被启动")
	}
	if slowSawNoTxCtx.Load() {
		t.Fatal("事务外 handler 不应在领取事务 ctx 内执行（S3 I/O 必须在事务外）")
	}

	// 租约证据：helper 执行期间行未标记 processed，available_at 已推到未来。
	var attempts int
	var processedAt *time.Time
	var leasedFuture bool
	if err := qdb.QueryRow(ctx,
		"SELECT attempts, processed_at, available_at > now() + interval '2 minutes' FROM outbox_events WHERE topic = $1",
		slowTopic).Scan(&attempts, &processedAt, &leasedFuture); err != nil {
		t.Fatalf("read slow row: %v", err)
	}
	if processedAt != nil || attempts != 0 {
		t.Fatalf("handler 未结束前行状态 = processed:%v attempts:%d, want nil/0", processedAt, attempts)
	}
	if !leasedFuture {
		t.Fatal("执行中的行必须打租约（available_at 推到未来），避免被重复领取")
	}

	// 慢 handler 仍阻塞时，另一个事务内 topic 事件必须能被正常投递。
	if err := writer.Enqueue(ctx, fastTopic, map[string]any{"k": "fast-2"}); err != nil {
		t.Fatalf("enqueue fast event 2: %v", err)
	}
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain while slow handler running: %v", err)
	}
	if got := fastCalls.Load(); got != 2 {
		t.Fatalf("慢 handler 阻塞期间其它 topic 投递次数 = %d, want 2（慢 topic 不得阻塞其它 topic）", got)
	}
	if slowFinished.Load() {
		t.Fatal("handler 不应在阻塞放行前完成")
	}

	releaseOnce.Do(func() { close(release) })
	dispatcher.Wait()
	var lastError *string
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at, last_error FROM outbox_events WHERE topic = $1", slowTopic).
		Scan(&attempts, &processedAt, &lastError); err != nil {
		t.Fatalf("read slow row after finish: %v", err)
	}
	if lastError != nil {
		t.Fatalf("成功投递后 last_error 必须为空, got %q", *lastError)
	}
	if processedAt == nil || attempts != 1 {
		t.Fatalf("成功投递后行状态 = processed:%v attempts:%d, want 非空/1", processedAt, attempts)
	}
}

// 真实库用例：事务外路径的失败重试与死信语义必须与事务内路径一致
// （attempts 递增、available_at 退避、last_error 记录；满 maxAttempts 进死信）。
func TestPostgresIT_OutboxExternalRetryAndDeadLetter(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	suffix := time.Now().UnixNano()
	retryTopic := fmt.Sprintf("it_outbox_ext_retry_%d", suffix)
	deadTopic := fmt.Sprintf("it_outbox_ext_dead_%d", suffix)
	t.Cleanup(func() {
		_ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = ANY($1)", []string{retryTopic, deadTopic})
	})

	writer := NewWriter(factory)
	dispatcher := NewDispatcher(factory, nil)
	t.Cleanup(dispatcher.Wait)
	var failing atomic.Bool
	failing.Store(true)
	dispatcher.RegisterExternal(retryTopic, func(context.Context, json.RawMessage) error {
		if failing.Load() {
			return errors.New("storage blackhole")
		}
		return nil
	})
	dispatcher.RegisterExternal(deadTopic, func(context.Context, json.RawMessage) error {
		return errors.New("still down")
	})

	// 第一次投递失败：attempts=1、未终结、退避到未来、last_error 带错误文本。
	if err := writer.Enqueue(ctx, retryTopic, map[string]any{"k": "fail"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain fail: %v", err)
	}
	dispatcher.Wait()
	var attempts int
	var processedAt, availableAt *time.Time
	var lastError *string
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at, available_at, last_error FROM outbox_events WHERE topic = $1", retryTopic).
		Scan(&attempts, &processedAt, &availableAt, &lastError); err != nil {
		t.Fatalf("read retry row: %v", err)
	}
	if attempts != 1 || processedAt != nil {
		t.Fatalf("失败后行状态 = attempts:%d processed:%v, want 1/nil", attempts, processedAt)
	}
	if availableAt == nil || !availableAt.After(time.Now()) {
		t.Fatalf("失败后 available_at = %v, 应退避到未来", availableAt)
	}
	if lastError == nil || !strings.Contains(*lastError, "storage blackhole") {
		t.Fatalf("last_error = %v, 应包含 handler 错误", lastError)
	}

	// 退避到期后重投成功：attempts=2、processed 非空、last_error 清空。
	failing.Store(false)
	if err := qdb.Exec(ctx, "UPDATE outbox_events SET available_at = now() WHERE topic = $1", retryTopic); err != nil {
		t.Fatalf("force available: %v", err)
	}
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain retry: %v", err)
	}
	dispatcher.Wait()
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at, last_error FROM outbox_events WHERE topic = $1", retryTopic).
		Scan(&attempts, &processedAt, &lastError); err != nil {
		t.Fatalf("read retried row: %v", err)
	}
	if attempts != 2 || processedAt == nil || lastError != nil {
		t.Fatalf("重试成功后行状态 = attempts:%d processed:%v last_error:%v, want 2/非空/nil", attempts, processedAt, lastError)
	}

	// 死信：把 attempts 推到 maxAttempts-1 后再次失败 → processed 非空 + last_error 保留。
	if err := writer.Enqueue(ctx, deadTopic, map[string]any{"k": "dead"}); err != nil {
		t.Fatalf("enqueue dead letter: %v", err)
	}
	var deadEventID int64
	if err := qdb.QueryRow(ctx, "SELECT event_id FROM outbox_events WHERE topic = $1", deadTopic).Scan(&deadEventID); err != nil {
		t.Fatalf("read dead letter event id: %v", err)
	}
	if err := qdb.Exec(ctx, "UPDATE outbox_events SET attempts = $2, available_at = now() WHERE event_id = $1", deadEventID, maxAttempts-1); err != nil {
		t.Fatalf("force attempts: %v", err)
	}
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain dead letter: %v", err)
	}
	dispatcher.Wait()
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at, last_error FROM outbox_events WHERE event_id = $1", deadEventID).
		Scan(&attempts, &processedAt, &lastError); err != nil {
		t.Fatalf("read dead letter row: %v", err)
	}
	if attempts != maxAttempts || processedAt == nil || lastError == nil {
		t.Fatalf("死信行 = attempts:%d processed:%v last_error:%v, want %d/非空/非空", attempts, processedAt, lastError, maxAttempts)
	}
}

// 真实库用例：崩溃窗口重复投递。进程在「handler 执行完」与「结果写回」之间崩溃
// 时，行停留在租约中：租约到期前不会被领取，到期后重新投递；handler 幂等（对象
// 删除 404 视为已删），重复投递仍能正常终结。
func TestPostgresIT_OutboxExternalCrashWindowRedeliveryIdempotent(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	suffix := time.Now().UnixNano()
	topic := fmt.Sprintf("it_outbox_ext_crash_%d", suffix)
	t.Cleanup(func() { _ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = $1", topic) })

	writer := NewWriter(factory)
	dispatcher := NewDispatcher(factory, nil)
	t.Cleanup(dispatcher.Wait)
	var delivered atomic.Int32
	dispatcher.RegisterExternal(topic, func(context.Context, json.RawMessage) error {
		delivered.Add(1)
		return nil
	})

	if err := writer.Enqueue(ctx, topic, map[string]any{"k": "purge"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// 模拟「已领取并打租约、但进程崩溃前没写回」的窗口：handler 不会被执行。
	if err := qdb.Exec(ctx, "UPDATE outbox_events SET available_at = now() + interval '2 minutes' WHERE topic = $1", topic); err != nil {
		t.Fatalf("simulate crash lease: %v", err)
	}
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain during lease: %v", err)
	}
	dispatcher.Wait()
	if got := delivered.Load(); got != 0 {
		t.Fatalf("租约未到期时不得重复投递，delivered = %d", got)
	}

	// 租约到期 → 重新投递并正常终结。
	if err := qdb.Exec(ctx, "UPDATE outbox_events SET available_at = now() WHERE topic = $1", topic); err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain after lease expiry: %v", err)
	}
	dispatcher.Wait()
	var attempts int
	var processedAt *time.Time
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at FROM outbox_events WHERE topic = $1", topic).Scan(&attempts, &processedAt); err != nil {
		t.Fatalf("read leased row: %v", err)
	}
	if delivered.Load() != 1 || processedAt == nil {
		t.Fatalf("租约到期后应重投并终结: delivered=%d processed=%v", delivered.Load(), processedAt)
	}

	// 模拟重复投递（at-least-once）：同一事件再次进入待投递状态，幂等 handler 仍成功。
	if err := qdb.Exec(ctx, "UPDATE outbox_events SET processed_at = NULL, attempts = 0, available_at = now() WHERE topic = $1", topic); err != nil {
		t.Fatalf("simulate redelivery: %v", err)
	}
	if _, err := dispatcher.DrainOnce(ctx); err != nil {
		t.Fatalf("drain redelivery: %v", err)
	}
	dispatcher.Wait()
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at FROM outbox_events WHERE topic = $1", topic).Scan(&attempts, &processedAt); err != nil {
		t.Fatalf("read redelivered row: %v", err)
	}
	if got := delivered.Load(); got != 2 {
		t.Fatalf("at-least-once 重新投递次数 = %d, want 2", got)
	}
	if processedAt == nil {
		t.Fatal("幂等重复投递后事件必须再次终结")
	}
}
