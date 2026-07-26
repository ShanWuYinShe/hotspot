package bootstrap

import (
	"context"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type systemBootstrapResolver struct {
	users  *users.Service
	notify *notify.Service
}

func (r systemBootstrapResolver) ResolveProfile(ctx context.Context) (any, bool) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return nil, false
	}
	profile, err := r.users.GetMyProfile(ctx)
	if err != nil {
		return nil, false
	}
	_ = principal.UserID
	return profile, true
}

func (r systemBootstrapResolver) ResolveUnreadSummary(ctx context.Context) system.UnreadSummary {
	summary := system.UnreadSummary{}
	if result, err := r.notify.UnreadCount(ctx); err == nil {
		summary.NotificationUnreadCount = result.Total
		summary.TotalBadge = result.Total
	}
	return summary
}
