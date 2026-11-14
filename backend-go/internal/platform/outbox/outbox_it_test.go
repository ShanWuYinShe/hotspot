package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func outboxCount(t *testing.T, factory *persistence.Factory, topic string) int {
	t.Helper()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	var count int
	if err := qdb.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE topic = $1", topic).Scan(&count); err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	return count
}

func TestOutboxEnqueueRollsBackWithTransaction(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	topic := fmt.Sprintf("it_outbox_rollback_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		qdb, _ := factory.DB.(persistence.QueryableDB)
		_ = qdb.Exec(context.Background(), "DELETE FROM outbox_events WHERE topic = $1", topic)
	})
	factoryScope, cleanup := testdb.BeginScope(t)
	writer := NewWriter(factoryScope)
	if err := writer.Enqueue(context.Background(), topic, map[string]any{"k": "v"}); err != nil {
		t.Fatalf("enqueue in tx: %v", err)
	}
	cleanup()
	if got := outboxCount(t, factory, topic); got != 0 {
		t.Fatalf("rolled-back transaction must not persist outbox event, got %d", got)
	}
}

func TestOutboxDispatcherProcessesAndRetries(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	topic := fmt.Sprintf("it_outbox_dispatch_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		qdb, _ := factory.DB.(persistence.QueryableDB)
		_ = qdb.Exec(context.Background(), "DELETE FROM outbox_events WHERE topic = $1", topic)
	})
	writer := NewWriter(factory)
	dispatcher := NewDispatcher(factory, nil)

	if err := writer.Enqueue(context.Background(), topic, map[string]any{"k": "v"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	delivered := 0
	dispatcher.Register(topic, func(_ context.Context, payload json.RawMessage) error {
		delivered++
		var got map[string]string
		if err := json.Unmarshal(payload, &got); err != nil {
			return err
		}
		if got["k"] != "v" {
			return fmt.Errorf("unexpected payload: %#v", got)
		}
		return nil
	})
	if processed, err := dispatcher.DrainOnce(context.Background()); err != nil || processed != 1 {
		t.Fatalf("first drain: processed=%d err=%v", processed, err)
	}
	if delivered != 1 {
		t.Fatalf("delivered = %d, want 1", delivered)
	}
	if processed, err := dispatcher.DrainOnce(context.Background()); err != nil || processed != 0 {
		t.Fatalf("second drain must be empty: processed=%d err=%v", processed, err)
	}

	failTopic := topic + "_retry"
	if err := writer.Enqueue(context.Background(), failTopic, map[string]any{"k": "retry"}); err != nil {
		t.Fatalf("enqueue retry: %v", err)
	}
	attempts := 0
	dispatcher.Register(failTopic, func(context.Context, json.RawMessage) error {
		attempts++
		if attempts == 1 {
			return errors.New("boom")
		}
		return nil
	})
	if processed, err := dispatcher.DrainOnce(context.Background()); err != nil || processed != 1 {
		t.Fatalf("failing drain: processed=%d err=%v", processed, err)
	}
	qdb, _ := factory.DB.(persistence.QueryableDB)
	var rowAttempts int
	var processedAt *time.Time
	if err := qdb.QueryRow(context.Background(), "SELECT attempts, processed_at FROM outbox_events WHERE topic = $1", failTopic).Scan(&rowAttempts, &processedAt); err != nil {
		t.Fatalf("read retry row: %v", err)
	}
	if rowAttempts != 1 || processedAt != nil {
		t.Fatalf("retry row = attempts:%d processed:%v, want 1/nil", rowAttempts, processedAt)
	}
	if err := qdb.Exec(context.Background(), "UPDATE outbox_events SET available_at = now() WHERE topic = $1", failTopic); err != nil {
		t.Fatalf("force available: %v", err)
	}
	if processed, err := dispatcher.DrainOnce(context.Background()); err != nil || processed != 1 {
		t.Fatalf("retry drain: processed=%d err=%v", processed, err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	t.Cleanup(func() { _ = qdb.Exec(context.Background(), "DELETE FROM outbox_events WHERE topic = $1", failTopic) })
}

// TestOutboxDispatcherStartSurvivesHandlerPanic 覆盖后台循环（Start）层面：
// handler panic 转 error 后，领取事务不回滚、dispatcher goroutine 不停转，
// 毒事件按退避重试、其它事件继续投递。改造前（worker.Go + panic 逃逸）该用例
// 会因 goroutine 永久退出而超时：attempts 始终为 0、后续事件不再投递。
func TestOutboxDispatcherStartSurvivesHandlerPanic(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	suffix := time.Now().UnixNano()
	panicTopic := fmt.Sprintf("it_outbox_start_panic_%d", suffix)
	okTopic := fmt.Sprintf("it_outbox_start_ok_%d", suffix)
	t.Cleanup(func() {
		_ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = ANY($1)", []string{panicTopic, okTopic})
	})

	writer := NewWriter(factory)
	dispatcher := NewDispatcher(factory, nil)
	// 后台循环默认 1s 轮询；测试内调快，避免等待。
	dispatcher.pollInterval = 5 * time.Millisecond
	var goodDelivered atomic.Int32
	dispatcher.Register(panicTopic, func(context.Context, json.RawMessage) error {
		panic("handler panic injected by Start IT")
	})
	dispatcher.Register(okTopic, func(context.Context, json.RawMessage) error {
		goodDelivered.Add(1)
		return nil
	})
	runCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel) // LIFO：先停循环，再由上面的 cleanup 删行
	dispatcher.Start(runCtx)

	if err := writer.Enqueue(ctx, panicTopic, map[string]any{"k": "panic"}); err != nil {
		t.Fatalf("enqueue panic event: %v", err)
	}
	// attempts 递增即证明：panic 后后台循环仍然存活（旧实现会永久退出）。
	waitOutbox(t, 5*time.Second, func() bool {
		var attempts int
		if err := qdb.QueryRow(ctx, "SELECT attempts FROM outbox_events WHERE topic = $1", panicTopic).Scan(&attempts); err != nil {
			t.Fatalf("read panic attempts: %v", err)
		}
		return attempts >= 1
	}, "毒事件 attempts 未递增：dispatcher 后台循环可能在 panic 后停转")

	// 毒事件退避期间后续正常事件必须照常投递。
	if err := writer.Enqueue(ctx, okTopic, map[string]any{"k": "ok"}); err != nil {
		t.Fatalf("enqueue ok event: %v", err)
	}
	waitOutbox(t, 5*time.Second, func() bool { return goodDelivered.Load() >= 1 }, "毒事件退避期间正常事件未被投递")
}

// waitOutbox 在 deadline 内轮询 cond；超时即 t.Fatalf。
func waitOutbox(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
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

func TestOutboxWriterNoopWithoutFactory(t *testing.T) {
	var writer *Writer
	if err := writer.Enqueue(context.Background(), "t", map[string]any{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("nil writer must fail fast, got %v", err)
	}
}

// TestOutboxHandlerPanicKeepsBatchAndRetries 覆盖 handler panic 的隔离语义：
// panic 被 outbox 内部转成普通 error，领取事务不整批回滚——同批后续事件照常
// 提交，毒事件 attempts 递增并按退避重试；满 maxAttempts 后进入死信，后续事件
// 仍被正常投递（毒事件不阻塞队列）。
func TestOutboxHandlerPanicKeepsBatchAndRetries(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	suffix := time.Now().UnixNano()
	panicTopic := fmt.Sprintf("it_outbox_panic_%d", suffix)
	okTopic := fmt.Sprintf("it_outbox_after_panic_%d", suffix)
	t.Cleanup(func() {
		_ = qdb.Exec(ctx, "DELETE FROM outbox_events WHERE topic = ANY($1)", []string{panicTopic, okTopic})
	})

	writer := NewWriter(factory)
	dispatcher := NewDispatcher(factory, nil)
	goodDelivered := 0
	dispatcher.Register(panicTopic, func(context.Context, json.RawMessage) error {
		panic("handler panic injected by IT")
	})
	dispatcher.Register(okTopic, func(context.Context, json.RawMessage) error {
		goodDelivered++
		return nil
	})

	// 毒事件先入队（event_id 更小）：与正常事件同批被领取，用来验证 panic 不连累同批事务。
	if err := writer.Enqueue(ctx, panicTopic, map[string]any{"k": "panic"}); err != nil {
		t.Fatalf("enqueue panic event: %v", err)
	}
	if err := writer.Enqueue(ctx, okTopic, map[string]any{"k": "ok"}); err != nil {
		t.Fatalf("enqueue ok event: %v", err)
	}

	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != 2 {
		t.Fatalf("panic 同批 drain: processed=%d err=%v, want 2（panic 转 error 后事务不能整批回滚）", processed, err)
	}
	if goodDelivered != 1 {
		t.Fatalf("同批正常事件投递次数 = %d, want 1", goodDelivered)
	}

	// panic 事件按普通失败处理：attempts=1、未标记 processed、已排期退避，last_error 带 panic 内容。
	var attempts int
	var processedAt, availableAt *time.Time
	var lastError *string
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at, available_at, last_error FROM outbox_events WHERE topic = $1", panicTopic).
		Scan(&attempts, &processedAt, &availableAt, &lastError); err != nil {
		t.Fatalf("read panic row: %v", err)
	}
	if attempts != 1 || processedAt != nil {
		t.Fatalf("panic row = attempts:%d processed:%v, want 1/nil（panic 必须走失败重试路径）", attempts, processedAt)
	}
	if availableAt == nil || !availableAt.After(time.Now()) {
		t.Fatalf("panic row available_at = %v, 应退避到未来", availableAt)
	}
	if lastError == nil || !strings.Contains(*lastError, "handler panic injected by IT") {
		t.Fatalf("panic row last_error = %v, 应包含 panic 内容", lastError)
	}

	// 允许重试后再投递一次：attempts 必须继续递增（毒事件不会永久卡在第 1 次）。
	if err := qdb.Exec(ctx, "UPDATE outbox_events SET available_at = now() WHERE topic = $1", panicTopic); err != nil {
		t.Fatalf("force panic event available: %v", err)
	}
	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != 1 {
		t.Fatalf("panic 重试 drain: processed=%d err=%v, want 1", processed, err)
	}
	if err := qdb.QueryRow(ctx, "SELECT attempts FROM outbox_events WHERE topic = $1", panicTopic).Scan(&attempts); err != nil {
		t.Fatalf("read panic attempts: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("panic attempts = %d, want 2（panic 必须与普通失败一样累计 attempts）", attempts)
	}

	// 毒事件退避期间，后续正常事件不受阻塞（dispatcher 不停转）。
	if err := writer.Enqueue(ctx, okTopic, map[string]any{"k": "ok-2"}); err != nil {
		t.Fatalf("enqueue ok event 2: %v", err)
	}
	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != 1 {
		t.Fatalf("毒事件退避期间 drain: processed=%d err=%v, want 1（正常事件必须照常投递）", processed, err)
	}
	if goodDelivered != 2 {
		t.Fatalf("正常事件投递次数 = %d, want 2", goodDelivered)
	}

	// 追到 maxAttempts-1 再触发一次 panic：应进入死信（processed_at 非空、attempts=maxAttempts）。
	if err := qdb.Exec(ctx, "UPDATE outbox_events SET attempts = $2, available_at = now() WHERE topic = $1", panicTopic, maxAttempts-1); err != nil {
		t.Fatalf("force panic event to maxAttempts-1: %v", err)
	}
	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != 1 {
		t.Fatalf("死信 drain: processed=%d err=%v, want 1", processed, err)
	}
	if err := qdb.QueryRow(ctx, "SELECT attempts, processed_at, last_error FROM outbox_events WHERE topic = $1", panicTopic).
		Scan(&attempts, &processedAt, &lastError); err != nil {
		t.Fatalf("read dead-letter row: %v", err)
	}
	if attempts != maxAttempts || processedAt == nil {
		t.Fatalf("dead-letter row = attempts:%d processed:%v, want %d/非空", attempts, processedAt, maxAttempts)
	}
	if lastError == nil || !strings.Contains(*lastError, "handler panic injected by IT") {
		t.Fatalf("dead-letter last_error = %v, 应保留 panic 内容", lastError)
	}

	// 死信后队列继续投递：再入队一条正常事件，drain 仍能处理。
	if err := writer.Enqueue(ctx, okTopic, map[string]any{"k": "ok-3"}); err != nil {
		t.Fatalf("enqueue ok event 3: %v", err)
	}
	if processed, err := dispatcher.DrainOnce(ctx); err != nil || processed != 1 {
		t.Fatalf("死信后 drain: processed=%d err=%v, want 1", processed, err)
	}
	if goodDelivered != 3 {
		t.Fatalf("正常事件投递次数 = %d, want 3", goodDelivered)
	}
}
