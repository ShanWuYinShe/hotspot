package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/worker"
)

// TopicSystemAnnouncement 是系统公告 WS 广播事件（admin 生产，bootstrap 订阅）。
const TopicSystemAnnouncement = "realtime.system_announcement"

// TopicAdminReportHandled 是举报裁决结果 WS 通知（admin 生产，bootstrap 订阅）。
const TopicAdminReportHandled = "realtime.admin_report_handled"

// TopicMediaPurge 是账户注销级联的媒体对象清理事件（media 生产，bootstrap
// 订阅）：payload 是该用户一批待删 storage_key（media.PurgeObjectsPayload）。
// 生产方在删除上传会话行之后按批入队，由 dispatcher 在请求生命周期外逐条删除
// 对象，避免超大媒体账户的注销请求同步串行删除数千对象而超时（第 81 轮审计
// 批次 5 残留）；单条删除失败返回错误走 outbox 既有重试/死信语义。
// 该 topic 由 bootstrap 以 RegisterExternal（事务外执行）注册：S3 删除不放在
// 领取事务内，慢/黑洞存储不会拖住其它 topic 投递或长期占用数据库连接。
const TopicMediaPurge = "media.purge_objects"

const (
	defaultPollInterval = time.Second
	defaultBatchSize    = 50
	maxAttempts         = 10
	maxBackoff          = 5 * time.Minute

	// 事务外（external）topic 的执行参数（RegisterExternal）：handler 不在领取
	// 事务内运行，而是短事务领取后在有界并发 goroutine 里执行、再用短事务写回。
	// 这样慢外部 IO（如对象存储批量删除）既不占用领取事务/连接，也不会阻塞其它
	// topic 在本 dispatcher 循环里的投递。
	externalHandlerTimeout  = 60 * time.Second // 单事件 handler 总时长上限（超时按投递失败重试）
	externalFinalizeTimeout = 15 * time.Second // 结果写回短事务上限
	externalConcurrency     = 4                // 同时执行的事务外事件数上限（每个事件一条 goroutine）
	// externalLease 是领取事务里给行打上的租约：执行期间 available_at 被推到
	// 未来，其它实例/下一轮不会重复领取。必须显著大于单事件超时，否则执行中的
	// 行会被重新领取（at-least-once 下 handler 幂等，但会放大外部 IO）。
	externalLease = 3 * time.Minute

	// 保留期清理（审计缺口：outbox_events 只增不减）：成功投递的事件行保留
	// defaultRetention 后分批物理删除；重试中的行永不自动删除——失败记录要留给
	// 人排查。成功/死信此前都只靠 processed_at 标记、无法区分，这里以 last_error
	// 为判别口径：成功路径显式清空 last_error，死信路径必然写入错误文本，故
	// last_error IS NULL 等价于「成功投递」。
	//
	// 死信（processed_at 非空 + last_error 非空）单独用更长的 deadLetterRetention：
	// 其 payload/last_error 是排查投递失败的唯一线索，先由 Gauge
	// hotspot_outbox_dead_letters + 告警暴露堆积，超期后才分批清理，避免静默
	// 丢弃排查线索（审计第 81 轮遗留项）。
	defaultRetention      = 7 * 24 * time.Hour  // 成功事件保留期（7 天）
	deadLetterRetention   = 30 * 24 * time.Hour // 死信保留期（30 天）
	retentionBatchSize    = 500                 // 单批删除行数上限，避免一次 DELETE 长时间持锁/撑大 WAL
	retentionMaxBatches   = 100                 // 单轮批数上限（单轮最多 5 万行，积压时多轮追平）
	retentionSweepPeriod  = time.Hour           // 后台清理周期
	retentionSweepTimeout = 5 * time.Minute     // 单轮超时，避免慢删除拖住后续周期
)

const queryInsertOutbox = `INSERT INTO outbox_events (topic, payload) VALUES ($1, $2::jsonb)`

// queryClaimOutbox 按 topic 白名单领取一批待投递事件，$1 是领取条数上限、$2 是
// topic 列表。inline 与 external 两条路径共用同一 SQL，只是 topic 列表换成各自
// 注册集合（inline 用 Register 注册的 topic、external 用 RegisterExternal 注册的
// topic），因此两类 topic 分开领取、互不挤占同一批行。
//
// 领取条件用白名单 "topic = ANY($2)" 而不是此前的「排除事务外 topic」反向过滤
// （topic <> ALL(externalTopics)）：外部 topic（媒体清理）极端积压时，反向过滤
// 必须按 event_id 顺序逐行扫描再丢弃积压行（O(积压)，10 万积压实测 10.5ms）且随
// 积压线性增长；白名单可直接走迁移 072 的 (topic, available_at, event_id) 部分
// 索引按 topic 定位，成本与其它 topic 的积压量无关。
const queryClaimOutbox = `SELECT event_id, topic, payload, attempts
FROM outbox_events
WHERE processed_at IS NULL AND available_at <= now() AND topic = ANY($2)
ORDER BY event_id
LIMIT $1
FOR UPDATE SKIP LOCKED`

// queryLeaseOutboxEvent 给本轮领取的事务外事件打租约：available_at 推到未来，
// 执行期间不会被其它实例/下一轮重复领取；进程崩溃（结果没写回）时租约到期后
// 重新投递，保持 at-least-once 语义（handler 必须幂等）。
const queryLeaseOutboxEvent = `UPDATE outbox_events SET available_at = now() + $2::interval WHERE event_id = $1 AND processed_at IS NULL`

// queryForceCustomPlan 让本轮领取事务内的语句按实际参数值重新规划（custom
// plan）。领取查询的 topic 列表是运行时数组参数：pgx 默认走预编译语句，而
// PostgreSQL 的 plan_cache_mode=auto 在若干次执行后可能改为 generic plan——
// 此时 planner 不知道数组实际元素，按「可能匹配很多 topic」估算（实测 10 万
// 行积压时估算 ~98% 命中），退化为按 event_id 顺序的主键扫描（Rows Removed by
// Filter 10 万、约 10ms），迁移 072 的 topic 部分索引完全用不上，白名单改造
// 也就失效。SET LOCAL 只作用于本次领取事务（提交/回滚即失效），不影响同一
// 连接上其它查询；custom plan 的规划开销约 0.05-0.1ms，相对 1s 轮询可忽略。
const queryForceCustomPlan = `SET LOCAL plan_cache_mode = force_custom_plan`

// 事务内执行（Register）路径的结果写回 SQL：handler 与写回同处领取事务，无需
// 额外保护。
const (
	queryMarkOutboxProcessed  = `UPDATE outbox_events SET processed_at = now(), attempts = $2, last_error = NULL WHERE event_id = $1`
	queryMarkOutboxDeadLetter = `UPDATE outbox_events SET processed_at = now(), attempts = $2, last_error = $3 WHERE event_id = $1`
	queryScheduleOutboxRetry  = `UPDATE outbox_events SET attempts = $2, available_at = now() + $3::interval, last_error = $4 WHERE event_id = $1`
)

// 事务外执行（RegisterExternal）路径的结果写回 SQL：handler 在事务外运行，
// 期间行可能因租约过期被重新领取、甚至已被新一轮写回，因此用 attempts（领取
// 时的值）做比较交换当乐观锁，旧结果落空时丢弃（事件会在新一轮正常投递）。
// persistence.Queryable 的 Exec 不暴露 affected rows，用 CTE + RETURNING 计数
// （与上方保留期清理同口径）。$N 末尾参数是领取时的 attempts。
const (
	queryMarkLeasedOutboxProcessed = `
WITH updated AS (
    UPDATE outbox_events SET processed_at = now(), attempts = $2, last_error = NULL
    WHERE event_id = $1 AND processed_at IS NULL AND attempts = $3
    RETURNING 1
)
SELECT COUNT(*) FROM updated`
	queryMarkLeasedOutboxDeadLetter = `
WITH updated AS (
    UPDATE outbox_events SET processed_at = now(), attempts = $2, last_error = $3
    WHERE event_id = $1 AND processed_at IS NULL AND attempts = $4
    RETURNING 1
)
SELECT COUNT(*) FROM updated`
	queryScheduleLeasedOutboxRetry = `
WITH updated AS (
    UPDATE outbox_events SET attempts = $2, available_at = now() + $3::interval, last_error = $4
    WHERE event_id = $1 AND processed_at IS NULL AND attempts = $5
    RETURNING 1
)
SELECT COUNT(*) FROM updated`
)

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

// queryDeleteDeadLettersBefore 单批删除「已进入死信且超过保留期」的事件行：
// 结构与 queryDeleteProcessedOutboxBefore 相同，判别条件换成 last_error IS NOT
// NULL（死信），保留期从 processed_at（进入死信的时间）起算。成功事件与死信
// 是互斥集合（last_error 为空/非空），两者由不同循环清理、互不影响。
const queryDeleteDeadLettersBefore = `
WITH deleted AS (
    DELETE FROM outbox_events
    WHERE ctid IN (
        SELECT ctid FROM outbox_events
        WHERE processed_at IS NOT NULL
          AND processed_at < $1
          AND last_error IS NOT NULL
        ORDER BY processed_at
        LIMIT $2
    )
    RETURNING 1
)
SELECT COUNT(*) FROM deleted`

// queryCountOutboxDeadLetters 统计当前死信行数（Gauge 取值来源）：processed_at
// 非空（已终结）且 last_error 非空（最后一次投递失败）。只由保留期 sweeper 按
// 周期查询并刷新 Gauge，不在 /metrics 抓取路径查库（抓取远快于清扫周期，且 DB
// 故障时不应拖住指标抓取）。
const queryCountOutboxDeadLetters = `SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NOT NULL AND last_error IS NOT NULL`

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

// Dispatcher 定时领取未处理事件并投递。Register/RegisterExternal 必须在
// Start/DrainOnce 前调用。
type Dispatcher struct {
	factory      *persistence.Factory
	logger       *observability.Logger
	handlers     map[string]Handler
	external     map[string]struct{} // 注册为「事务外执行」的 topic（RegisterExternal）
	handlersMu   sync.RWMutex
	pollInterval time.Duration
	batchSize    int
	// externalInFlight/externalWG 跟踪事务外执行的并发槽位与在途 handler：槽位
	// 用 externalMu 保护「查空位 → 领取 → 占位」序列，避免并发 DrainOnce 超额
	// 领取；Wait 供测试与优雅停机编排等待在途 handler 结束。
	externalMu       sync.Mutex
	externalInFlight int
	externalWG       sync.WaitGroup
	// deadLetters 是最近一轮保留期清扫查到的死信行数，hotspot_outbox_dead_letters
	// Gauge 的取值来源（见 NewDeadLettersCollector）：清扫 goroutine 写、/metrics
	// 抓取读，用原子量避免数据竞争；抓取路径不查库。
	deadLetters atomic.Int64
}

func NewDispatcher(factory *persistence.Factory, logger *observability.Logger) *Dispatcher {
	if logger == nil {
		logger = observability.L()
	}
	return &Dispatcher{
		factory:      factory,
		logger:       logger,
		handlers:     map[string]Handler{},
		external:     map[string]struct{}{},
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
	}
}

// Register 注册「事务内执行」的 topic 处理器（dispatcher 启动前调用）：handler
// 在领取事务内同步执行，只适合内存操作（WS 广播等）——慢 handler 会拖长领取
// 事务并阻塞同批其它 topic，外部 IO 型 handler 必须用 RegisterExternal。
func (d *Dispatcher) Register(topic string, handler Handler) {
	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()
	d.handlers[topic] = handler
	delete(d.external, topic)
}

// RegisterExternal 注册「事务外执行」的 topic 处理器（dispatcher 启动前调用）：
// dispatcher 在短事务里领取事件并用 available_at 租约占位（不持锁到执行结束），
// 提交后在独立 goroutine 中执行 handler（有界并发 externalConcurrency、单事件
// 超时 externalHandlerTimeout），成功/失败再用短事务写回 processed_at/attempts。
//
// 适用于外部 IO 型 handler（如对象存储批量删除）：慢 handler 不占用领取事务/
// 数据库连接，也不会阻塞其它 topic 的投递；重试/死信/panic 隔离语义与 Register
// 一致。语义为 at-least-once——进程在「执行完成」与「结果写回」之间崩溃、或
// 租约过期被重新领取时，事件会被再次投递，handler 必须幂等。
func (d *Dispatcher) RegisterExternal(topic string, handler Handler) {
	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()
	d.handlers[topic] = handler
	d.external[topic] = struct{}{}
}

func (d *Dispatcher) handlerFor(topic string) Handler {
	d.handlersMu.RLock()
	defer d.handlersMu.RUnlock()
	return d.handlers[topic]
}

// inlineTopics 返回当前注册为「事务内执行」的 topic 列表（领取 SQL 的参数）：
// handlers 中除去 external 的 topic。Register 与 RegisterExternal 维护两集合
// 互斥（Register 会把同 topic 的 external 标记清掉），故 handlers-external 即
// inline 集合；未注册任何 inline topic 时返回 nil，调用方跳过领取。
func (d *Dispatcher) inlineTopics() []string {
	d.handlersMu.RLock()
	defer d.handlersMu.RUnlock()
	if len(d.handlers) == len(d.external) {
		return nil
	}
	topics := make([]string, 0, len(d.handlers)-len(d.external))
	for topic := range d.handlers {
		if _, ok := d.external[topic]; ok {
			continue
		}
		topics = append(topics, topic)
	}
	return topics
}

// externalTopics 返回当前注册为事务外执行的 topic 列表（领取 SQL 的参数）。
func (d *Dispatcher) externalTopics() []string {
	d.handlersMu.RLock()
	defer d.handlersMu.RUnlock()
	if len(d.external) == 0 {
		return nil
	}
	topics := make([]string, 0, len(d.external))
	for topic := range d.external {
		topics = append(topics, topic)
	}
	return topics
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
	// 死信 Gauge 也由该 goroutine 顺带刷新（启动先刷一次），/metrics 抓取不查库。
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

// pruneRetentionOnce 执行一轮保留期清理：先删成功投递超期事件（7 天），再删
// 超期死信（30 天）。两类都按 processed_at 从旧到新分批删除，直到没有可删行或
// 达到单轮批数上限（积压时多轮追平）；单轮独立超时，避免慢删除拖住后续周期，
// ctx 取消（优雅停机）时立即返回。两类清理相互独立：成功事件清理失败不阻断
// 死信清理（反之亦然），各自记录 Warn 后继续。
//
// 死信 Gauge 在本轮前后各刷新一次：开始时先暴露历史堆积（清理失败也看得到），
// 结束后校正为清理后的取值（避免告警因过期读数误报）；启动首轮即完成首次刷新。
func (d *Dispatcher) pruneRetentionOnce(ctx context.Context) {
	if d == nil || d.factory == nil || ctx.Err() != nil {
		return
	}
	sweepCtx, cancel := context.WithTimeout(ctx, retentionSweepTimeout)
	defer cancel()
	d.refreshDeadLetterGauge(sweepCtx)
	now := time.Now().UTC()

	processedCutoff := now.Add(-defaultRetention)
	processedDeleted, err := d.pruneBatches(sweepCtx, processedCutoff, d.deleteProcessedBatch)
	if err != nil {
		if sweepCtx.Err() == nil {
			d.logger.Warn("outbox: retention prune failed", "deleted", processedDeleted, "error", err)
		}
	} else if processedDeleted > 0 {
		d.logger.Info("outbox: processed events pruned", "deleted", processedDeleted, "cutoff", processedCutoff)
	}

	deadCutoff := now.Add(-deadLetterRetention)
	deadDeleted, err := d.pruneBatches(sweepCtx, deadCutoff, d.deleteDeadLetterBatch)
	if err != nil {
		if sweepCtx.Err() == nil {
			d.logger.Warn("outbox: dead letter retention prune failed", "deleted", deadDeleted, "error", err)
		}
	} else if deadDeleted > 0 {
		d.logger.Info("outbox: dead letters pruned", "deleted", deadDeleted, "cutoff", deadCutoff)
	}

	d.refreshDeadLetterGauge(sweepCtx)
}

// pruneBatches 以 cutoff 为界分批删除，直到没有可删行或达到单轮批数上限（积压
// 时多轮追平），返回本轮累计删除数与最后错误；出错时停止该类清理。
func (d *Dispatcher) pruneBatches(ctx context.Context, cutoff time.Time, deleteBatch func(context.Context, time.Time, int) (int64, error)) (int64, error) {
	var total int64
	for batch := 0; batch < retentionMaxBatches; batch++ {
		deleted, err := deleteBatch(ctx, cutoff, retentionBatchSize)
		total += deleted
		if err != nil {
			return total, err
		}
		if deleted < retentionBatchSize {
			break
		}
	}
	return total, nil
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

// deleteDeadLetterBatch 删除一批超过保留期的死信事件，返回删除行数。
func (d *Dispatcher) deleteDeadLetterBatch(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	queryable, ok := persistence.QueryableForTx(ctx, d.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var deleted int64
	if err := queryable.QueryRow(ctx, queryDeleteDeadLettersBefore, cutoff, limit).Scan(&deleted); err != nil {
		return 0, err
	}
	return deleted, nil
}

// countDeadLetters 查询当前死信行数（新进入的死信 + 尚未过保留期的历史死信）。
func (d *Dispatcher) countDeadLetters(ctx context.Context) (int64, error) {
	queryable, ok := persistence.QueryableForTx(ctx, d.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var count int64
	if err := queryable.QueryRow(ctx, queryCountOutboxDeadLetters).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// refreshDeadLetterGauge 刷新死信 Gauge 取值；查询失败只记 Warn、保留上一次
// 取值——把「查不到」写成 0 会把真实堆积伪装成「没有死信」，让告警静默失效。
func (d *Dispatcher) refreshDeadLetterGauge(ctx context.Context) {
	count, err := d.countDeadLetters(ctx)
	if err != nil {
		if ctx.Err() == nil {
			d.logger.Warn("outbox: count dead letters failed", "error", err)
		}
		return
	}
	d.deadLetters.Store(count)
}

// NewDeadLettersCollector 返回死信总数 Gauge（hotspot_outbox_dead_letters）：
// 取值来自最近一轮保留期清扫的查询结果（见 Dispatcher.refreshDeadLetterGauge），
// /metrics 抓取路径零查询、不依赖 DB 可用性。同名告警规则见
// docker/prometheus-alerts.yml（HotspotOutboxDeadLetters）。
func NewDeadLettersCollector(d *Dispatcher) prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "hotspot_outbox_dead_letters",
		Help: "outbox 死信总数（已终结但最后一次投递失败的事件；超过保留期会被分批清理）",
	}, func() float64 {
		if d == nil {
			return 0
		}
		return float64(d.deadLetters.Load())
	})
}

type outboxRow struct {
	eventID  int64
	topic    string
	payload  json.RawMessage
	attempts int
}

// DrainOnce 领取并投递一批事件，返回处理（含失败）条数。测试与后台循环共用。
//
// 两类 topic 分开领取，互不阻塞：
//   - 普通 topic（Register）：只按注册的 inline topic 白名单领取，在同一个事务内
//     执行 handler 并写回结果；handler 只应是内存操作，慢 handler 会拖长该事务。
//     （白名单精确匹配使领取成本与外部 topic 积压量无关，见 queryClaimOutbox）
//   - 事务外 topic（RegisterExternal）：独立短事务里按空闲并发槽位领取并打租约
//     （available_at 推到未来），提交后在有界并发 goroutine 里执行 handler（单
//     事件超时），再用短事务写回结果；外部 IO 不占用领取事务/连接，也不阻塞
//     其它 topic（do not wait：等待会重新把慢 handler 放回轮询循环的关键路径）。
func (d *Dispatcher) DrainOnce(ctx context.Context) (int, error) {
	if d == nil || d.factory == nil {
		return 0, persistence.ErrNotImplemented
	}
	processed, err := d.drainInline(ctx)
	if err != nil {
		return processed, err
	}
	external, err := d.drainExternal(ctx)
	return processed + external, err
}

// drainInline 执行「事务内」路径：领取一批事件、在同一个事务里逐条执行 handler
// 并写回结果（成功/重试/死信）。panic 由 invokeHandler 转 error，不掀事务。
//
// 只领取 Register 注册的 inline topic（白名单精确匹配，见 queryClaimOutbox）；
// 未注册任何 inline topic 时直接返回、不开启领取事务——纯事务外 topic 装配
// （或未装配 outbox 的测试桩）不必为「注定空批」的领取付出事务/查询开销。
func (d *Dispatcher) drainInline(ctx context.Context) (int, error) {
	topics := d.inlineTopics()
	if len(topics) == 0 {
		return 0, nil
	}
	processed := 0
	err := persistence.RunInTransaction(ctx, d.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableFromContext(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		// 按实际 topic 数组值规划，避免 plan_cache_mode=auto 退化为 generic plan
		// 后改走主键扫描（见 queryForceCustomPlan 注释）。
		if err := queryable.Exec(txCtx, queryForceCustomPlan); err != nil {
			return err
		}
		claimed, err := claimOutboxRows(txCtx, queryable, queryClaimOutbox, d.batchSize, topics)
		if err != nil {
			return err
		}
		for _, row := range claimed {
			processed++
			handler := d.handlerFor(row.topic)
			var handlerErr error
			if handler == nil {
				handlerErr = fmt.Errorf("outbox: no handler for topic %q", row.topic)
			} else {
				handlerErr = d.invokeHandler(txCtx, handler, row)
			}
			if err := d.writeOutcome(txCtx, queryable, row, handlerErr, false); err != nil {
				return err
			}
		}
		return nil
	})
	return processed, err
}

// drainExternal 执行「事务外」路径的领取阶段：按空闲并发槽位在短事务里领取
// 事务外 topic 的事件并打租约，提交后把每个事件交给独立 goroutine 执行。
// 本函数不等待 handler 结束（否则慢 handler 会重新阻塞 dispatcher 轮询循环）；
// 结果由 runExternal 用短事务写回，测试/停机编排可用 Wait 等待在途 handler。
func (d *Dispatcher) drainExternal(ctx context.Context) (int, error) {
	topics := d.externalTopics()
	if len(topics) == 0 {
		return 0, nil
	}
	reserved := d.reserveExternalSlots()
	if reserved == 0 {
		// 并发槽位已满：本轮不领取新事件，积压行保持 due 状态等下一轮。
		return 0, nil
	}
	// 未派发出去的预留槽位必须归还（含领取失败与 panic 路径）：槽位泄漏会让
	// 事务外并发永久缩水甚至停摆（dispatcher 的 GoRestartable 会重启轮询循环，
	// 但 dispatcher 实例上的槽位计数不会被重置）。
	dispatched := 0
	defer func() {
		if spare := reserved - dispatched; spare > 0 {
			d.releaseExternalSlots(spare)
		}
	}()
	var rows []outboxRow
	err := persistence.RunInTransaction(ctx, d.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableFromContext(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		// 与 inline 领取共用同一白名单 SQL：topic 列表换成外部 topic、limit
		// 换成空闲并发槽位数；领取后在同一短事务里逐行打租约（见
		// queryLeaseOutboxEvent），提交后 handler 才在事务外执行。
		// 同样强制 custom plan：外部 topic 事件可能排在大量 inline 积压之后，
		// generic plan 的主键扫描会让领取成本随积压线性增长。
		if err := queryable.Exec(txCtx, queryForceCustomPlan); err != nil {
			return err
		}
		claimed, err := claimOutboxRows(txCtx, queryable, queryClaimOutbox, reserved, topics)
		if err != nil {
			return err
		}
		for _, row := range claimed {
			if err := queryable.Exec(txCtx, queryLeaseOutboxEvent, row.eventID, externalLease.String()); err != nil {
				return err
			}
		}
		rows = claimed
		return nil
	})
	if err != nil {
		// 领取事务失败（DB 不可用等）：defer 释放预留槽位，下一轮重试。
		return 0, err
	}
	for _, row := range rows {
		d.startExternal(ctx, row) // 每个 handler 结束后自行释放自己占用的槽位
		dispatched++
	}
	return len(rows), nil
}

// reserveExternalSlots 预留事务外执行的空闲并发槽位，返回预留成功数（0 表示
// 已满）；领取到的事件数不足预留数时由调用方释放多余槽位，每个 handler 结束
// 后（runExternal）释放自己占用的槽位。
func (d *Dispatcher) reserveExternalSlots() int {
	d.externalMu.Lock()
	defer d.externalMu.Unlock()
	free := externalConcurrency - d.externalInFlight
	if free <= 0 {
		return 0
	}
	d.externalInFlight += free
	return free
}

func (d *Dispatcher) releaseExternalSlots(n int) {
	d.externalMu.Lock()
	d.externalInFlight -= n
	d.externalMu.Unlock()
}

// startExternal 启动一个事务外 handler（每事件一个 goroutine，并发已由
// drainExternal 的槽位预留限制）。goroutine 内再兜一层 recover：异步执行不在
// dispatcher 轮询循环的 GoRestartable 保护范围内，panic 逃逸会掀掉整个进程。
func (d *Dispatcher) startExternal(ctx context.Context, row outboxRow) {
	d.externalWG.Add(1)
	go func() {
		defer d.externalWG.Done()
		defer d.releaseExternalSlots(1)
		defer func() {
			if rec := recover(); rec != nil {
				d.logger.Error("outbox: external handler panic escaped",
					"event_id", row.eventID, "topic", row.topic, "panic", rec, "stack", string(debug.Stack()))
			}
		}()
		d.runExternal(ctx, row)
	}()
}

// runExternal 在领取事务之外执行单条事件并写回结果。handler 拿到带单事件超时
// 的 ctx（慢外部 IO 不会无限拖住执行 goroutine）；panic 由 invokeHandler 转错。
//
// 停机（父 ctx 取消）语义：不把「被中断」当作投递失败累计 attempts（否则反复
// 滚动重启会把合法事件推进死信），事件保持租约、重启后租约到期自动重投；结果
// 写回使用脱离父 ctx 取消的短超时 ctx，保证正常结束的执行结果仍然落盘。
func (d *Dispatcher) runExternal(ctx context.Context, row outboxRow) {
	handler := d.handlerFor(row.topic)
	handlerCtx, cancel := context.WithTimeout(ctx, externalHandlerTimeout)
	var handlerErr error
	if handler == nil {
		handlerErr = fmt.Errorf("outbox: no handler for topic %q", row.topic)
	} else {
		handlerErr = d.invokeHandler(handlerCtx, handler, row)
	}
	cancel()
	if ctx.Err() != nil {
		d.logger.Warn("outbox: external handler interrupted by shutdown, event kept for lease retry",
			"event_id", row.eventID, "topic", row.topic)
		return
	}
	finalCtx, finalCancel := context.WithTimeout(context.WithoutCancel(ctx), externalFinalizeTimeout)
	defer finalCancel()
	err := persistence.RunInTransaction(finalCtx, d.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableFromContext(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		return d.writeOutcome(txCtx, queryable, row, handlerErr, true)
	})
	if err != nil {
		// 结果没写回：行仍处于租约中，租约到期后重新投递（at-least-once）。
		d.logger.Warn("outbox: finalize external event failed, event will be retried after lease",
			"event_id", row.eventID, "topic", row.topic, "error", err)
	}
}

// Wait 等待当前已在执行的事务外 handler 全部结束（含结果写回）。仅供测试与
// 优雅停机编排使用：生产 dispatcher 循环不等待，等待会让慢 handler 重新阻塞。
func (d *Dispatcher) Wait() {
	if d == nil {
		return
	}
	d.externalWG.Wait()
}

// claimOutboxRows 执行领取 SQL 并扫描事件行。
func claimOutboxRows(ctx context.Context, queryable persistence.Queryable, query string, limit int, topics []string) ([]outboxRow, error) {
	if limit <= 0 {
		return nil, nil
	}
	if topics == nil {
		// 传空数组而不是 nil：nil []string 会被 pgx 编码为 NULL，让
		// "topic = ANY(NULL)" 恒为 NULL（依赖 NULL 三值语义才能得到空批）；
		// 显式空数组语义更直接，也不受 ANY/NOT IN 等改写影响。
		topics = []string{}
	}
	rows, err := queryable.Query(ctx, query, limit, topics)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	claimed := make([]outboxRow, 0, limit)
	for rows.Next() {
		var row outboxRow
		var payload []byte
		if err := rows.Scan(&row.eventID, &row.topic, &payload, &row.attempts); err != nil {
			return nil, err
		}
		row.payload = append(json.RawMessage(nil), payload...)
		claimed = append(claimed, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return claimed, nil
}

// writeOutcome 写回一条事件的投递结果：成功标记 processed；失败累计 attempts
// 并按指数退避重排，达到 maxAttempts 进死信。leased=true 表示结果来自事务外
// 执行的 handler，用 attempts 做比较交换（见 queryMarkLeased* 注释），旧结果
// 落空只记 Warn、不报错（事件会被重新投递）；leased=false 保持既有领取事务内
// 执行路径的无条件更新语义。
func (d *Dispatcher) writeOutcome(ctx context.Context, queryable persistence.Queryable, row outboxRow, handlerErr error, leased bool) error {
	nextAttempts := row.attempts + 1
	if handlerErr == nil {
		if !leased {
			return queryable.Exec(ctx, queryMarkOutboxProcessed, row.eventID, nextAttempts)
		}
		return d.execLeasedOutcome(ctx, queryable, queryMarkLeasedOutboxProcessed, row, row.eventID, nextAttempts, row.attempts)
	}
	if nextAttempts >= maxAttempts {
		d.logger.Error("outbox: event dead-lettered", "event_id", row.eventID, "topic", row.topic, "attempts", nextAttempts, "error", handlerErr)
		if !leased {
			return queryable.Exec(ctx, queryMarkOutboxDeadLetter, row.eventID, nextAttempts, handlerErr.Error())
		}
		return d.execLeasedOutcome(ctx, queryable, queryMarkLeasedOutboxDeadLetter, row, row.eventID, nextAttempts, handlerErr.Error(), row.attempts)
	}
	backoff := time.Duration(1<<uint(nextAttempts)) * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	if leased {
		if err := d.execLeasedOutcome(ctx, queryable, queryScheduleLeasedOutboxRetry, row, row.eventID, nextAttempts, backoff.String(), handlerErr.Error(), row.attempts); err != nil {
			return err
		}
	} else if err := queryable.Exec(ctx, queryScheduleOutboxRetry, row.eventID, nextAttempts, backoff.String(), handlerErr.Error()); err != nil {
		return err
	}
	d.logger.Warn("outbox: delivery failed, scheduled retry", "event_id", row.eventID, "topic", row.topic, "attempts", nextAttempts, "error", handlerErr)
	return nil
}

// execLeasedOutcome 执行带 attempts CAS 保护的结果写回并统计影响行数：0 行表示
// 租约已过期（行被重新领取/已由新一轮写回），旧结果丢弃，只记 Warn 便于排查。
func (d *Dispatcher) execLeasedOutcome(ctx context.Context, queryable persistence.Queryable, query string, row outboxRow, args ...any) error {
	var updated int64
	if err := queryable.QueryRow(ctx, query, args...).Scan(&updated); err != nil {
		return err
	}
	if updated == 0 {
		d.logger.Warn("outbox: external event result discarded (lease expired or already finalized)",
			"event_id", row.eventID, "topic", row.topic)
	}
	return nil
}
