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
