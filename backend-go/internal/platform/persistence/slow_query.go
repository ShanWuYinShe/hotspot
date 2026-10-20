package persistence

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// queryStartKey 是 context 里携带"查询开始信息"的键（未导出，避免与其它 tracer 冲突）。
type queryStartKey struct{}

// queryStartInfo 记录一次查询的开始时间与 SQL 文本。
// pgx 的 TraceQueryStart 能拿到 SQL，TraceQueryEnd 只有 CommandTag ——
// 想在结束时打出慢 SQL，必须在开始时把 SQL 存进 context。
type queryStartInfo struct {
	at  time.Time
	sql string
}

// SlowQueryTracer 是 pgx.QueryTracer 实现：执行超过阈值的 SQL 通过回调上报。
//
// 设计取舍：
//   - 只记录**耗时**，不做采样/聚合 —— 本地与生产都够用，且不会引入额外状态；
//   - 阈值 <= 0 时完全关闭（零开销，不污染 context）；
//   - 日志函数由调用方注入，便于单测断言，也避免 persistence 包直接依赖 observability。
//
// 阈值来源：config.SlowQueryThreshold（env SLOW_QUERY_THRESHOLD，默认 200ms）。
type SlowQueryTracer struct {
	threshold time.Duration
	logf      func(sql string, duration time.Duration, err error)
}

// NewSlowQueryTracer 创建慢查询 tracer。logf 为 nil 或 threshold <= 0 时等价于关闭。
func NewSlowQueryTracer(threshold time.Duration, logf func(sql string, duration time.Duration, err error)) *SlowQueryTracer {
	if threshold <= 0 || logf == nil {
		return nil
	}
	return &SlowQueryTracer{threshold: threshold, logf: logf}
}

// TraceQueryStart 记录开始时间与 SQL。
func (t *SlowQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if t == nil {
		return ctx
	}
	return context.WithValue(ctx, queryStartKey{}, queryStartInfo{at: time.Now(), sql: data.SQL})
}

// TraceQueryEnd 计算耗时，超过阈值时回调上报（含错误信息）。
func (t *SlowQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if t == nil {
		return
	}
	info, ok := ctx.Value(queryStartKey{}).(queryStartInfo)
	if !ok {
		return
	}
	elapsed := time.Since(info.at)
	if elapsed < t.threshold {
		return
	}
	t.logf(info.sql, elapsed, data.Err)
}
