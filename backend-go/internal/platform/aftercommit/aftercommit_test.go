package aftercommit

import (
	"context"
	"testing"
)

func TestRegisterRunsAfterCommit(t *testing.T) {
	ctx := WithHooks(context.Background())
	var order []string
	Register(ctx, func(context.Context) { order = append(order, "hook1") })
	Register(ctx, func(context.Context) { order = append(order, "hook2") })
	if len(order) != 0 {
		t.Fatalf("hooks must not run before commit: %v", order)
	}
	Run(ctx)
	if len(order) != 2 || order[0] != "hook1" || order[1] != "hook2" {
		t.Fatalf("unexpected hook order: %v", order)
	}
	// 只执行一次
	Run(ctx)
	if len(order) != 2 {
		t.Fatalf("hooks must run exactly once, got %v", order)
	}
}

func TestDiscardRollsBackHooks(t *testing.T) {
	ctx := WithHooks(context.Background())
	ran := false
	Register(ctx, func(context.Context) { ran = true })
	Discard(ctx)
	Run(ctx)
	if ran {
		t.Fatal("discarded hooks must not run")
	}
}

func TestRegisterOutsideTransactionRunsImmediately(t *testing.T) {
	ran := false
	Register(context.Background(), func(context.Context) { ran = true })
	if !ran {
		t.Fatal("hook without transaction must run immediately")
	}
}

// TestRunIsolatesHookPanic 回归审计 A 平台层 P2-1：单个提交后回调 panic 不得中断同事务
// 其余回调，也不得冒泡给调用方——执行到 Run 时事务已经 COMMIT，向上抛出会被
// 误判为提交失败（触发错误的回滚/重试语义）。
func TestRunIsolatesHookPanic(t *testing.T) {
	ctx := WithHooks(context.Background())
	var order []string
	Register(ctx, func(context.Context) { order = append(order, "before") })
	Register(ctx, func(context.Context) { panic("hook boom") })
	Register(ctx, func(context.Context) { order = append(order, "after") })

	Run(ctx) // panic 被隔离：不 panic 到本测试
	if len(order) != 2 || order[0] != "before" || order[1] != "after" {
		t.Fatalf("panic hook 后其余回调仍应按注册顺序执行, got %v", order)
	}

	// panic 不影响「只执行一次」语义：再次 Run 不会重放回调。
	Run(ctx)
	if len(order) != 2 {
		t.Fatalf("hooks 只应执行一次, got %v", order)
	}
}
