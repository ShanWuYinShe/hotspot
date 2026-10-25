package system

import (
	"context"
	"fmt"
	"sync"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/buildinfo"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
)

type Service struct {
	repo      Repository
	provider  Provider
	bootstrap BootstrapResolver
}

func NewService(appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(appConfig))
}

func NewServiceWithFactory(factory *persistence.Factory, appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewPostgresRepository(factory), NewAppConfigProvider(appConfig))
}

func NewServiceWithDeps(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) GetMeta(ctx context.Context) ApiVersionMeta {
	return s.repo.GetMeta(ctx)
}

func (s *Service) Health(ctx context.Context) (int, string, *HealthStatus) {
	return s.repo.Health(ctx)
}

// SetHealthRedisPinger wires an optional redis liveness probe onto the
// underlying repository, when it supports one. No-op for in-memory repos.
func (s *Service) SetHealthRedisPinger(p func(ctx context.Context) error) {
	type redisPingerSetter interface {
		SetRedisPinger(func(ctx context.Context) error)
	}
	if setter, ok := s.repo.(redisPingerSetter); ok {
		setter.SetRedisPinger(p)
	}
}

func (s *Service) ListAnnouncements(ctx context.Context) (SystemAnnouncementList, error) {
	items, err := s.repo.ListAnnouncements(ctx, s.provider.GetSystemAnnouncementsCount(ctx))
	if err != nil {
		observability.L().Error("system.list_announcements failed", "error", err)
		return SystemAnnouncementList{}, err
	}
	resp := SystemAnnouncementList{Items: make([]SystemAnnouncementItem, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, item.PublicItem())
	}
	return resp, nil
}

func (s *Service) GetClientConfig(ctx context.Context) ClientConfig {
	return s.provider.GetClientConfig(ctx)
}

func (s *Service) Bootstrap(ctx context.Context, bootstrapResolver BootstrapResolver) BootstrapData {
	meta := s.GetMeta(ctx)
	clientConfig := s.GetClientConfig(ctx)

	// 三个互相独立的数据库读取并行化：公告版本、登录态画像、未读摘要。
	// 串行时 bootstrap 延迟 ≈ 3×DB RTT（压测 CPU profile 中 pgx Query 占累计 27%），
	// 并行后 ≈ max(RTT)。每个 goroutine 自带 recover：HTTP Recoverer 捕获不到
	// 其它 goroutine 的 panic，未恢复会直接 crash 进程；失败降级与串行版一致
	// （版本降级空串、profile 缺失则 profile/unread_summary 均为 null）。
	var annVersion string
	var profile any
	var profileOK bool
	var summary UnreadSummary

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer recoverBootstrapPanic("announcements_version")
		annVersion = s.announcementsVersion(ctx)
	}()
	if bootstrapResolver != nil {
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer recoverBootstrapPanic("resolve_profile")
			profile, profileOK = bootstrapResolver.ResolveProfile(ctx)
		}()
		go func() {
			defer wg.Done()
			defer recoverBootstrapPanic("resolve_unread_summary")
			summary = bootstrapResolver.ResolveUnreadSummary(ctx)
		}()
	}
	wg.Wait()

	bd := BootstrapData{
		BootstrapVersion: fmt.Sprintf("%s:%s:%s", buildinfo.Version, s.provider.AppConfigVersion(ctx), annVersion),
		Meta:             &meta,
		ClientConfig:     &clientConfig,
	}
	if bootstrapResolver != nil && profileOK {
		bd.Profile = profile
		// 仅登录态返回未读摘要；匿名时恒定 nullable（与串行实现一致）。
		bd.UnreadSummary = &summary
	}
	return bd
}

// recoverBootstrapPanic 记录并吞掉 bootstrap 并行分支的 panic（避免单个读路径
// 异常打挂整个进程；该请求按降级数据返回）。
func recoverBootstrapPanic(stage string) {
	if recovered := recover(); recovered != nil {
		observability.L().Error("system.bootstrap goroutine panic", "stage", stage, "panic", fmt.Sprint(recovered))
	}
}

// bootstrapVersion 组装客户端缓存版本（spec api_spec_system.md 缓存契约：
// 「bootstrap_version 由服务端在平台管理员修改 AppConfig 或发布新版本/
// 公告时主动递增」）。三个维度任一变化即整体变化：
//   - buildinfo.Version：发布新版本；
//   - AppConfig.UpdatedAt（Provider.AppConfigVersion）：管理员修改配置；
//   - 公告域指纹（Repository.AnnouncementsVersion）：发布/修改/删除公告。
//
// profile/unread_summary 为用户私有数据，不参与版本计算（spec 明示
// 「bootstrap_version 不因用户资料/未读数变化而递增」）。此前实现恒取
// 静态 meta.Version（永为 v1、永不变化），客户端接入 If-None-Match 后
// 缓存将永不失效（审查 P1-2）。单一维度查询失败记 Warn 并降级为空串
// 参与拼接（其余两个维度仍能触发失效）。
func (s *Service) announcementsVersion(ctx context.Context) string {
	annVersion, err := s.repo.AnnouncementsVersion(ctx)
	if err != nil {
		observability.L().Warn("system.bootstrap announcements version query failed", "error", err)
		return ""
	}
	return annVersion
}

func (s *Service) GetBootstrapResolver() BootstrapResolver  { return s.bootstrap }
func (s *Service) SetBootstrapResolver(r BootstrapResolver) { s.bootstrap = r }
