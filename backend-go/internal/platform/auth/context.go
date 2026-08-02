package auth

import (
	"context"
	"strings"
	"time"
)

// AdminMembership is a marker carried on Principal for the AdminAccess
// middleware's belt-and-suspenders pre-check only. It is NOT the authorization
// source of truth: every admin Service method calls requirePermission, which
// queries the admin_users / admin_role_permissions tables (the real RBAC).
// user_id is server-generated as "user_<random>" (see idgen), so the prefix
// heuristic below cannot be triggered by any client-controlled input — it is
// a redundant backstop, not a gate that can be bypassed.
type AdminMembership struct {
	Permissions []string
	Roles       []string
}

type Principal struct {
	UserID          string
	SessionID       string
	Platform        string
	AdminMembership *AdminMembership
}

func (p Principal) IsAdmin() bool {
	return p.AdminMembership != nil
}

func newPrincipal(userID, sessionID, platform string) Principal {
	principal := Principal{UserID: userID, SessionID: sessionID, Platform: platform}
	// Redundant backstop: user_id is always "user_<random>" (server-generated),
	// so this branch is effectively dead in production. Real admin authorization
	// happens in admin.Service.requirePermission via a DB lookup. Kept so the
	// AdminAccess middleware can short-circuit obvious non-admins cheaply.
	if userID == "admin" || strings.HasPrefix(userID, "admin_") {
		principal.AdminMembership = &AdminMembership{}
	}
	return principal
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
