package idempotency

import "context"

type contextKey string

const keyContext contextKey = "idempotency-key"

func WithKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, keyContext, key)
}
func KeyFromContext(ctx context.Context) string { key, _ := ctx.Value(keyContext).(string); return key }

const storeContext contextKey = "idempotency-store"

// WithStoreContext 把 Store 注入 ctx：根中间件（WithStore）在装配期注入一次，
// ReplayOrStore 从 ctx 取实例（不再有进程级 SetStore 可变全局）。
func WithStoreContext(ctx context.Context, store Store) context.Context {
	if store == nil {
		return ctx
	}
	return context.WithValue(ctx, storeContext, store)
}

func storeFromContext(ctx context.Context) Store {
	if ctx == nil {
		return nil
	}
	store, _ := ctx.Value(storeContext).(Store)
	return store
}
