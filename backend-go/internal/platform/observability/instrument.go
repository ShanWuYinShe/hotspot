package observability

import "time"

// instrument.go — 服务层 started/failed/completed 三段式日志样板收敛。
//
// 此前各 Service 方法手写 6-8 行成对样板（startedAt 计时 + 三个阶段日志 +
// elapsed_ms 换算），groups/auth 等模块合计 60+ 处，一处字段改动需多处同步。
// 用法（对齐既有日志键与消息格式，仅字段顺序规范化）：
//
//	op := observability.Begin("groups.update", "group_id", req.GroupID)
//	// 失败路径
//	op.Fail(err)
//	// 预期业务拒绝（Warn 级）
//	op.Warn("not found", "group_id", req.GroupID)
//	// 成功路径（extra 为该阶段特有字段，如 count）
//	op.Complete("count", len(items))

// Operation 单次服务操作的计时日志器。
type Operation struct {
	op        string
	startedAt time.Time
	kv        []any
}

// Begin 记录 "<op> started"（kv 为该操作全程携带的公共字段）并开始计时。
func Begin(op string, kv ...any) *Operation {
	L().Info(op+" started", kv...)
	return &Operation{op: op, startedAt: time.Now(), kv: kv}
}

func (o *Operation) elapsedMs() int64 {
	return time.Since(o.startedAt).Milliseconds()
}

// base 组装公共字段 + 额外字段 + 自动 elapsed_ms。
func (o *Operation) base(extra ...any) []any {
	fields := make([]any, 0, len(o.kv)+len(extra)+2)
	fields = append(fields, o.kv...)
	fields = append(fields, extra...)
	return append(fields, "elapsed_ms", o.elapsedMs())
}

// Fail 记录 Error 级 "<op> failed"（error 字段在首，elapsed_ms 收尾）。
func (o *Operation) Fail(err error, extra ...any) {
	fields := make([]any, 0, len(extra)+2)
	fields = append(fields, "error", err)
	fields = append(fields, extra...)
	L().Error(o.op+" failed", append(fields, "elapsed_ms", o.elapsedMs())...)
}

// Warn 记录 Warn 级 "<op> <suffix>"（预期业务拒绝走 Warn，见 AGENTS.md 日志纪律），
// 自动附加 elapsed_ms。
func (o *Operation) Warn(suffix string, kv ...any) {
	L().Warn(o.op+" "+suffix, append(kv, "elapsed_ms", o.elapsedMs())...)
}

// Complete 记录 Info 级 "<op> completed"（extra 为结果字段，elapsed_ms 收尾）。
func (o *Operation) Complete(extra ...any) {
	L().Info(o.op+" completed", o.base(extra...)...)
}

// ElapsedMillis 暴露已计时长（少数非标准阶段日志直接引用时使用）。
func (o *Operation) ElapsedMillis() int64 {
	return o.elapsedMs()
}
