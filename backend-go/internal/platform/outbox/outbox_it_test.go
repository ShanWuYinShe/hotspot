package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestOutboxWriterNoopWithoutFactory(t *testing.T) {
	var writer *Writer
	if err := writer.Enqueue(context.Background(), "t", map[string]any{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("nil writer must fail fast, got %v", err)
	}
}
