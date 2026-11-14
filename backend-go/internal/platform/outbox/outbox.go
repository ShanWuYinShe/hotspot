package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/worker"
)

// TopicSystemAnnouncement 是系统公告 WS 广播事件（admin 生产，bootstrap 订阅）。
const TopicSystemAnnouncement = "realtime.system_announcement"

// TopicAdminReportHandled 是举报裁决结果 WS 通知（admin 生产，bootstrap 订阅）。
const TopicAdminReportHandled = "realtime.admin_report_handled"

const (
	defaultPollInterval = time.Second
	defaultBatchSize    = 50
	maxAttempts         = 10
	maxBackoff          = 5 * time.Minute

	// 保留期清理（审计缺口：outbox_events 只增不减）：成功投递的事件行保留
	// defaultRetention 后分批物理删除；死信与重试中的行永不自动删除——失败记录
	// 要留给人排查。成功/死信此前都只靠 processed_at 标记、无法区分，这里以
	// last_error 为判别口径：成功路径显式清空 last_error，死信路径必然写入
	// 错误文本，故 last_error IS NULL 等价于「成功投递」。
	defaultRetention      = 7 * 24 * time.Hour // 成功事件保留期（7 天）
	retentionBatchSize    = 500                // 单批删除行数上限，避免一次 DELETE 长时间持锁/撑大 WAL
	retentionMaxBatches   = 100                // 单轮批数上限（单轮最多 5 万行，积压时多轮追平）
	retentionSweepPeriod  = time.Hour          // 后台清理周期
	retentionSweepTimeout = 5 * time.Minute    // 单轮超时，避免慢删除拖住后续周期
)

const queryInsertOutbox = `INSERT INTO outbox_events (topic, payload) VALUES ($1, $2::jsonb)`

const queryClaimOutbox = `SELECT event_id, topic, payload, attempts
FROM outbox_events
WHERE processed_at IS NULL AND available_at <= now()
ORDER BY event_id
LIMIT $1
FOR UPDATE SKIP LOCKED`

// queryDeleteProcessedOutboxBefore 单批删除「成功投递且超过保留期」的事件行：
// 子查询借 idx_outbox_processed 按 processed_at 顺序取最早的一批，再按 ctid
// 定位删除，避免无界 DELETE 一次锁住大批历史行。last_error IS NULL 是成功
// 投递的判别口径（见上方保留期注释），死信与重试中的行不满足条件、不会被删。
// CTE + RETURNING 用于统计删除行数（persistence.Queryable 的 Exec 不暴露
// affected rows）。
const queryDeleteProcessedOutboxBefore = `
WITH deleted AS (
    DELETE FROM outbox_events
    WHERE ctid IN (
        SELECT ctid FROM outbox_events
        WHERE processed_at IS NOT NULL
          AND processed_at < $1
          AND last_error IS NULL
        ORDER BY processed_at
        LIMIT $2
    )
    RETURNING 1
)
SELECT COUNT(*) FROM deleted`

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

// invokeHandler 在 recover 保护下调用 handler：panic 转成普通 error（记 ERROR
// 日志 + 堆栈）。handler 在领取事务内执行，panic 若逃逸会掀掉整批事务——同批
// 事件一起回滚、attempts 不增长，毒事件每次被重新领取并永久阻塞它之后的事件。
// 转成 error 后走既有失败路径：事务继续提交，attempts/指数退避/dead-letter
// 语义与普通 handler 错误一致，at-least-once 语义不变。
func (d *Dispatcher) invokeHandler(ctx context.Context, handler Handler, row outboxRow) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			d.logger.Error("outbox: handler panic recovered",
				"event_id", row.eventID,
				"topic", row.topic,
				"panic", rec,
				"stack", string(debug.Stack()),
			)
			err = fmt.Errorf("outbox: handler panic on topic %q: %v", row.topic, rec)
		}
	}()
	return handler(ctx, row.payload)
}

// Start 在 ctx 生命周期内后台轮询；随应用优雅停机退出。
//
// 用 worker.GoRestartable 而非 worker.Go：handler 的 panic 已在本包的
// invokeHandler 里转成 error，但领取/提交等非 handler 路径仍可能 panic，而
// worker.Go 恢复后 goroutine 就永久退出——dispatcher 停转会让所有 topic
// （含事务里刚提交的事件）再也不投递。GoRestartable 记录堆栈后退避重启；
// 干净返回（ctx 取消）不重启，符合优雅停机语义。
func (d *Dispatcher) Start(ctx context.Context) {
	if d == nil || d.factory == nil {
		return
	}
	worker.GoRestartable("outbox.dispatcher", func() {
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
	// 保留期清理挂在 Start 的既有生命周期内（rootCtx 取消即退出），不额外增加
	// bootstrap wiring。单独 goroutine：大批删除不阻塞事件投递；启动先跑一轮
	//（与既有 notify/media/auth sweeper 一致），之后每 retentionSweepPeriod 一轮。
	worker.GoRestartable("outbox.retention", func() {
		d.pruneRetentionOnce(ctx)
		ticker := time.NewTicker(retentionSweepPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.pruneRetentionOnce(ctx)
			}
		}
	})
}

// pruneRetentionOnce 执行一轮保留期清理：按 processed_at 从旧到新分批删除成功
// 投递的事件，直到没有可删行或达到单轮批数上限（积压时多轮追平）。单轮独立
// 超时，避免慢删除拖住后续周期；ctx 取消（优雅停机）时立即返回。
func (d *Dispatcher) pruneRetentionOnce(ctx context.Context) {
	if d == nil || d.factory == nil || ctx.Err() != nil {
		return
	}
	sweepCtx, cancel := context.WithTimeout(ctx, retentionSweepTimeout)
	defer cancel()
	cutoff := time.Now().UTC().Add(-defaultRetention)
	var total int64
	for batch := 0; batch < retentionMaxBatches; batch++ {
		deleted, err := d.deleteProcessedBatch(sweepCtx, cutoff, retentionBatchSize)
		total += deleted
		if err != nil {
			if sweepCtx.Err() == nil {
				d.logger.Warn("outbox: retention prune failed", "deleted", total, "error", err)
			}
			return
		}
		if deleted < retentionBatchSize {
			break
		}
	}
	if total > 0 {
		d.logger.Info("outbox: processed events pruned", "deleted", total, "cutoff", cutoff)
	}
}

// deleteProcessedBatch 删除一批超过保留期的已成功投递事件，返回删除行数。
func (d *Dispatcher) deleteProcessedBatch(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	queryable, ok := persistence.QueryableForTx(ctx, d.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var deleted int64
	if err := queryable.QueryRow(ctx, queryDeleteProcessedOutboxBefore, cutoff, limit).Scan(&deleted); err != nil {
		return 0, err
	}
	return deleted, nil
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
				handlerErr = d.invokeHandler(txCtx, handler, row)
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
