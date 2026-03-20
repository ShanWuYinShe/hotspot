package auth

import (
	"context"
	"time"
)

type Principal struct {
	UserID    string
	SessionID string
	Platform  string
	IsAdmin   bool
}

type contextKey string

const (
	principalKey    contextKey = "principal"
	platformKey     contextKey = "platform"
	accessExpiryKey contextKey = "access_expiry"
)

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey).(Principal)
	return principal, ok
}

func WithPlatform(ctx context.Context, platform string) context.Context {
	return context.WithValue(ctx, platformKey, platform)
}

func platformFromContext(ctx context.Context) string {
	platform, _ := ctx.Value(platformKey).(string)
	return platform
}

func WithAccessExpiry(ctx context.Context, expiresAt time.Time) context.Context {
	return context.WithValue(ctx, accessExpiryKey, expiresAt)
}

func AccessExpiryFromContext(ctx context.Context) (time.Time, bool) {
	expiresAt, ok := ctx.Value(accessExpiryKey).(time.Time)
	return expiresAt, ok
}
