// Package aftercommit 提供「事务提交后回调」：事务内注册的副作用（审计日志、
// 通知、实时广播）在 COMMIT 成功后才执行；ROLLBACK/panic 时丢弃，避免
// 「库已回滚但外部世界已看到副作用」。
//
// 用法：在 persistence.RunInTransaction 的 txCtx 内 Register；回调同步执行
// （在事务提交返回前），失败由调用方自行记录，不影响已提交事务。
package aftercommit

import (
	"context"
	"runtime/debug"
	"sync"

	"hotspot/backend-go/internal/platform/observability"
)

type hooksContextKey struct{}

// Hooks 是一次事务内注册的回调集合。
type Hooks struct {
	mu   sync.Mutex
	fns  []func(context.Context)
	done bool
	// base 是开启事务前的原始 ctx（不含已关闭的事务句柄）：提交后回调必须
	// 用它执行审计/通知，否则仓储会从 ctx 里拿到已 COMMIT 的事务连接。
	base context.Context
}

// WithHooks 把回调集合放入 ctx；由 RunInTransaction 在开启事务时用**事务前**
// 的 ctx 调用，回调最终以该 base ctx 执行。
func WithHooks(ctx context.Context) context.Context {
	return context.WithValue(ctx, hooksContextKey{}, &Hooks{base: ctx})
}

func hooksFromContext(ctx context.Context) (*Hooks, bool) {
	hooks, ok := ctx.Value(hooksContextKey{}).(*Hooks)
	return hooks, ok
}

// Register 注册提交后回调。ctx 不在事务内（没有 hooks 容器）时立即同步执行，
// 保证「非事务调用方」语义与直接调用一致。
func Register(ctx context.Context, fn func(context.Context)) {
	hooks, ok := hooksFromContext(ctx)
	if !ok {
		fn(ctx)
		return
	}
	hooks.mu.Lock()
	defer hooks.mu.Unlock()
	if hooks.done {
		// 事务已结束（回调阶段或已回滚）：不再接受注册，直接执行一次，
		// 避免调用方误以为副作用会被处理。
		fn(ctx)
		return
	}
	hooks.fns = append(hooks.fns, fn)
}

// Run 在 COMMIT 成功后按注册顺序同步执行全部回调（只执行一次）。
func Run(ctx context.Context) {
	hooks, ok := hooksFromContext(ctx)
	if !ok {
		return
	}
	hooks.mu.Lock()
	fns := hooks.fns
	hooks.fns = nil
	hooks.done = true
	base := hooks.base
	hooks.mu.Unlock()
	for _, fn := range fns {
		runHook(fn, base)
	}
}

// runHook 执行单个提交后回调并隔离 panic：回调（审计日志/通知/实时广播）本身
// 出错不应牵连同一事务的其余副作用，更不能把 panic 冒泡给调用方——执行到这里
// 时事务已经 COMMIT，向上抛出只会让调用方误判提交失败并触发错误的回滚/重试。
// panic 记 Error 日志（含堆栈）便于定位具体回调。
func runHook(fn func(context.Context), ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			observability.L().Error("aftercommit: hook panic recovered",
				"panic", rec,
				"stack", string(debug.Stack()),
			)
		}
	}()
	fn(ctx)
}

// Discard 在 ROLLBACK/panic 后丢弃全部回调（只丢弃一次）。
func Discard(ctx context.Context) {
	hooks, ok := hooksFromContext(ctx)
	if !ok {
		return
	}
	hooks.mu.Lock()
	hooks.fns = nil
	hooks.done = true
	hooks.mu.Unlock()
}
