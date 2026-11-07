package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/worker"
)

// TopicSystemAnnouncement 是系统公告 WS 广播事件（admin 生产，bootstrap 订阅）。
const TopicSystemAnnouncement = "realtime.system_announcement"

const (
	defaultPollInterval = time.Second
	defaultBatchSize    = 50
	maxAttempts         = 10
	maxBackoff          = 5 * time.Minute
)

const queryInsertOutbox = `INSERT INTO outbox_events (topic, payload) VALUES ($1, $2::jsonb)`

const queryClaimOutbox = `SELECT event_id, topic, payload, attempts
FROM outbox_events
WHERE processed_at IS NULL AND available_at <= now()
ORDER BY event_id
LIMIT $1
FOR UPDATE SKIP LOCKED`

// Writer 在业务事务内写事件行；ctx 内存在事务时与业务写同事务提交。
type Writer struct {
	factory *persistence.Factory
}

func NewWriter(factory *persistence.Factory) *Writer {
	if factory == nil {
		return nil
	}
	return &Writer{factory: factory}
}

// Enqueue 写入一条 outbox 事件。有事务上下文时加入该事务（推荐，保证与
// 状态变更原子）；无事务时退化为池直写（best-effort，测试/非事务调用）。
func (w *Writer) Enqueue(ctx context.Context, topic string, payload any) error {
	if w == nil || w.factory == nil {
		return persistence.ErrNotImplemented
	}
	if topic == "" {
		return errors.New("outbox: empty topic")
	}
	blob, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox: encode payload: %w", err)
	}
	queryable, ok := persistence.QueryableForTx(ctx, w.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryInsertOutbox, topic, string(blob)); err != nil {
		return fmt.Errorf("outbox: enqueue: %w", err)
	}
	return nil
}

// Handler 投递单条事件；返回错误时按指数退避重试（最多 maxAttempts）。
type Handler func(ctx context.Context, payload json.RawMessage) error

// Dispatcher 定时领取未处理事件并投递。Register 必须在 Start/DrainOnce 前调用。
type Dispatcher struct {
	factory      *persistence.Factory
	logger       *observability.Logger
	handlers     map[string]Handler
	handlersMu   sync.RWMutex
	pollInterval time.Duration
	batchSize    int
}

func NewDispatcher(factory *persistence.Factory, logger *observability.Logger) *Dispatcher {
	if logger == nil {
		logger = observability.L()
	}
	return &Dispatcher{factory: factory, logger: logger, handlers: map[string]Handler{}, pollInterval: defaultPollInterval, batchSize: defaultBatchSize}
}

// Register 注册 topic 处理器（dispatcher 启动前调用）。
func (d *Dispatcher) Register(topic string, handler Handler) {
	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()
	d.handlers[topic] = handler
}

func (d *Dispatcher) handlerFor(topic string) Handler {
	d.handlersMu.RLock()
	defer d.handlersMu.RUnlock()
	return d.handlers[topic]
}

// Start 在 ctx 生命周期内后台轮询；随应用优雅停机退出。
func (d *Dispatcher) Start(ctx context.Context) {
	if d == nil || d.factory == nil {
		return
	}
	worker.Go("outbox.dispatcher", func() {
		ticker := time.NewTicker(d.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := d.DrainOnce(ctx); err != nil {
					d.logger.Warn("outbox: drain failed", "error", err)
				}
			}
		}
	})
}

type outboxRow struct {
	eventID  int64
	topic    string
	payload  json.RawMessage
	attempts int
}

// DrainOnce 领取并投递一批事件，返回处理（含失败）条数。测试与后台循环共用。
func (d *Dispatcher) DrainOnce(ctx context.Context) (int, error) {
	if d == nil || d.factory == nil {
		return 0, persistence.ErrNotImplemented
	}
	processed := 0
	err := persistence.RunInTransaction(ctx, d.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableFromContext(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		rows, err := queryable.Query(txCtx, queryClaimOutbox, d.batchSize)
		if err != nil {
			return err
		}
		claimed := make([]outboxRow, 0, d.batchSize)
		for rows.Next() {
			var row outboxRow
			var payload []byte
			if err := rows.Scan(&row.eventID, &row.topic, &payload, &row.attempts); err != nil {
				rows.Close()
				return err
			}
			row.payload = append(json.RawMessage(nil), payload...)
			claimed = append(claimed, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, row := range claimed {
			processed++
			handler := d.handlerFor(row.topic)
			var handlerErr error
			if handler == nil {
				handlerErr = fmt.Errorf("outbox: no handler for topic %q", row.topic)
			} else {
				handlerErr = handler(txCtx, row.payload)
			}
			nextAttempts := row.attempts + 1
			if handlerErr == nil {
				if err := queryable.Exec(txCtx, `UPDATE outbox_events SET processed_at = now(), attempts = $2, last_error = NULL WHERE event_id = $1`, row.eventID, nextAttempts); err != nil {
					return err
				}
				continue
			}
			if nextAttempts >= maxAttempts {
				d.logger.Error("outbox: event dead-lettered", "event_id", row.eventID, "topic", row.topic, "attempts", nextAttempts, "error", handlerErr)
				if err := queryable.Exec(txCtx, `UPDATE outbox_events SET processed_at = now(), attempts = $2, last_error = $3 WHERE event_id = $1`, row.eventID, nextAttempts, handlerErr.Error()); err != nil {
					return err
				}
				continue
			}
			backoff := time.Duration(1<<uint(nextAttempts)) * time.Second
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			if err := queryable.Exec(txCtx, `UPDATE outbox_events SET attempts = $2, available_at = now() + $3::interval, last_error = $4 WHERE event_id = $1`, row.eventID, nextAttempts, backoff.String(), handlerErr.Error()); err != nil {
				return err
			}
			d.logger.Warn("outbox: delivery failed, scheduled retry", "event_id", row.eventID, "topic", row.topic, "attempts", nextAttempts, "error", handlerErr)
		}
		return nil
	})
	if err != nil {
		return processed, err
	}
	return processed, nil
}
