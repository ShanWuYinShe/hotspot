package admin

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
)

// stubMutableSource 模拟 MutableSource：Update 回调可在 apply 前对 cur
// 施加「并发写者」的改动（Get 返回的快照落后于回调入参）。
type stubMutableSource struct {
	cfg appconfig.AppConfig
	// onBeforeApply 在 Update 回调内、apply 之前修改 cur，模拟另一管理员
	// 在本次 Get 与 Update 之间落库的并发改动。
	onBeforeApply func(cur appconfig.AppConfig) appconfig.AppConfig
	saved         appconfig.AppConfig
}

func (s *stubMutableSource) Get(context.Context) appconfig.AppConfig { return s.cfg }

func (s *stubMutableSource) Update(_ context.Context, apply func(appconfig.AppConfig) appconfig.AppConfig) (appconfig.AppConfig, error) {
	cur := s.cfg
	if s.onBeforeApply != nil {
		cur = s.onBeforeApply(cur)
	}
	next := apply(cur)
	s.saved = next
	return next, nil
}

// 审查 P1-4 防回归：UpdateAppConfig 必须在 Update 回调内基于最新 cur 增量
// 合并。此前 provider 写回 Get 时刻的整份旧快照——并发管理员改掉的分区
// （此处为 features.enable_moments）会被本次更新整份回滚。
func TestUpdateAppConfigMergesAgainstConcurrentWrite(t *testing.T) {
	source := &stubMutableSource{
		cfg: appconfig.NewStaticSource(config.Load()).Get(context.Background()),
		onBeforeApply: func(cur appconfig.AppConfig) appconfig.AppConfig {
			// 并发写者：在本次 UpdateAppConfig 的 Get 之后关闭 moments。
			cur.Features.EnableMoments = false
			return cur
		},
	}
	provider := NewAppConfigProvider(source)

	newLimit := 777
	_, apiErr, err := provider.UpdateAppConfig(context.Background(), "admin_b", func(cfg AdminAppConfig) (AdminAppConfig, *httpx.APIError) {
		cfg.Limits.MaxGroupMembers = newLimit
		return cfg, nil
	})
	if err != nil || apiErr != nil {
		t.Fatalf("unexpected error: apiErr=%v err=%v", apiErr, err)
	}
	if source.saved.Limits.MaxGroupMembers != newLimit {
		t.Fatalf("expected own patch applied, got max_group_members=%d", source.saved.Limits.MaxGroupMembers)
	}
	if source.saved.Features.EnableMoments {
		t.Fatal("concurrent feature change was rolled back by full-snapshot write")
	}
}
