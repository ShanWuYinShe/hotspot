package observability

import (
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// observedDefault 把包级默认日志器换成 observer 收集器并返回日志句柄，
// 测试结束恢复原默认（L() 惰性初始化的实例可直接还原）。
func observedDefault(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	prev := L()
	core, logs := observer.New(zapcore.InfoLevel)
	SetDefault(&Logger{base: zap.New(core).Sugar()})
	t.Cleanup(func() { SetDefault(prev) })
	return logs
}

// Begin kv 是操作全程公共字段：started/failed/warned/completed 四阶段日志
// 必须全部自动携带，调用点不得重复传参（重复会输出两个同名键）。
func TestOperationCarriesBeginKVAcrossPhases(t *testing.T) {
	logs := observedDefault(t)
	op := Begin("test.op", "group_id", "g1", "keyword", "kw")

	op.Complete("count", 3)
	op.Warn("not found", "user_id", "u1")
	op.Fail(errors.New("boom"), "stage", "refresh")

	started := logs.FilterMessage("test.op started").All()
	if len(started) != 1 {
		t.Fatalf("started 条数 = %d, 期望 1", len(started))
	}
	if ctx := started[0].ContextMap(); ctx["group_id"] != "g1" || ctx["keyword"] != "kw" {
		t.Fatalf("started 缺少 Begin kv: %v", ctx)
	}

	completed := logs.FilterMessage("test.op completed").All()
	if len(completed) != 1 {
		t.Fatalf("completed 条数 = %d, 期望 1", len(completed))
	}
	if ctx := completed[0].ContextMap(); ctx["group_id"] != "g1" || ctx["count"] != int64(3) {
		t.Fatalf("completed 缺少 Begin kv/extra: %v", ctx)
	}

	warned := logs.FilterMessage("test.op not found").All()
	if len(warned) != 1 {
		t.Fatalf("warned 条数 = %d, 期望 1", len(warned))
	}
	if ctx := warned[0].ContextMap(); ctx["group_id"] != "g1" || ctx["user_id"] != "u1" {
		t.Fatalf("Warn 缺少 Begin kv/extra: %v", ctx)
	}

	failed := logs.FilterMessage("test.op failed").All()
	if len(failed) != 1 {
		t.Fatalf("failed 条数 = %d, 期望 1", len(failed))
	}
	if ctx := failed[0].ContextMap(); ctx["group_id"] != "g1" || ctx["stage"] != "refresh" {
		t.Fatalf("Fail 缺少 Begin kv/extra（此前 Fail 丢弃 kv 的回归点）: %v", ctx)
	}
	// error 字段在首、elapsed_ms 收尾
	fields := failed[0].Context
	if fields[0].Key != "error" {
		t.Fatalf("首字段 = %q, 期望 error", fields[0].Key)
	}
	if last := fields[len(fields)-1]; last.Key != "elapsed_ms" {
		t.Fatalf("收尾字段 = %q, 期望 elapsed_ms", last.Key)
	}
}

// 空 Begin（无 kv）时各阶段日志不得输出空键或 panic。
func TestOperationWithoutKV(t *testing.T) {
	logs := observedDefault(t)
	op := Begin("test.bare")

	op.Complete()
	op.Warn("blocked")
	op.Fail(errors.New("boom"))

	for _, m := range []string{"test.bare started", "test.bare completed", "test.bare blocked", "test.bare failed"} {
		entries := logs.FilterMessage(m).All()
		if len(entries) != 1 {
			t.Fatalf("%s 条数 = %d, 期望 1", m, len(entries))
		}
		for _, f := range entries[0].Context {
			if f.Key == "" {
				t.Fatalf("%s 出现空键字段: %v", m, entries[0].Context)
			}
		}
	}
}
