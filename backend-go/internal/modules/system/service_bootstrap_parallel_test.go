package system

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

// panicProfileResolver 在画像解析时 panic，用于验证并行分支的 recover 不会
// 打挂进程（HTTP Recoverer 捕获不到其它 goroutine 的 panic）。
type panicProfileResolver struct{}

func (panicProfileResolver) ResolveProfile(context.Context) (any, bool) {
	panic("resolver boom")
}

func (panicProfileResolver) ResolveUnreadSummary(context.Context) UnreadSummary {
	return UnreadSummary{NotifyTotal: 9, TotalBadge: 9}
}

// panicUnreadResolver 在未读摘要解析时 panic：profile 仍应正常返回，摘要降级为空。
type panicUnreadResolver struct{}

func (panicUnreadResolver) ResolveProfile(context.Context) (any, bool) {
	return map[string]any{"user_id": "u1"}, true
}

func (panicUnreadResolver) ResolveUnreadSummary(context.Context) UnreadSummary {
	panic("unread boom")
}

func TestServiceBootstrapRecoversProfilePanic(t *testing.T) {
	svc := NewService(appconfig.NewStaticSource(config.Load()))
	bd := svc.Bootstrap(context.Background(), panicProfileResolver{})
	if bd.BootstrapVersion == "" || bd.Meta.Version == "" {
		t.Fatalf("bootstrap version/meta must still be computed, got %#v", bd)
	}
	if bd.Profile != nil {
		t.Fatalf("panicked profile resolver must degrade to nil profile, got %#v", bd.Profile)
	}
	// 与串行实现一致：profile miss 时不返回 unread_summary。
	if bd.UnreadSummary != nil {
		t.Fatalf("profile panic must leave unread_summary nil, got %#v", bd.UnreadSummary)
	}
}

func TestServiceBootstrapRecoversUnreadPanic(t *testing.T) {
	svc := NewService(appconfig.NewStaticSource(config.Load()))
	bd := svc.Bootstrap(context.Background(), panicUnreadResolver{})
	if bd.Profile == nil {
		t.Fatal("profile must survive unread-summary panic")
	}
	if bd.UnreadSummary == nil || bd.UnreadSummary.NotifyTotal != 0 || bd.UnreadSummary.TotalBadge != 0 {
		t.Fatalf("panicked unread resolver must degrade to zero summary, got %#v", bd.UnreadSummary)
	}
}
