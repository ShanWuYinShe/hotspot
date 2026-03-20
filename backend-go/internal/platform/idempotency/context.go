package idempotency

import "context"

type contextKey string

const keyContext contextKey = "idempotency-key"

func WithKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, keyContext, key)
}
func KeyFromContext(ctx context.Context) string { key, _ := ctx.Value(keyContext).(string); return key }
