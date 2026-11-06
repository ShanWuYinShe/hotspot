package system

import (
	"context"
	"fmt"

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
	bd := BootstrapData{
		BootstrapVersion: s.bootstrapVersion(ctx),
		Meta:             &meta,
		ClientConfig:     &clientConfig,
	}
	if bootstrapResolver != nil {
		if profile, ok := bootstrapResolver.ResolveProfile(ctx); ok {
			bd.Profile = profile
			// 仅登录态解析未读摘要；匿名时 profile / unread_summary 均为 null（恒定 nullable）
			summary := bootstrapResolver.ResolveUnreadSummary(ctx)
			bd.UnreadSummary = &summary
		}
	}
	return bd
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
func (s *Service) bootstrapVersion(ctx context.Context) string {
	annVersion, err := s.repo.AnnouncementsVersion(ctx)
	if err != nil {
		observability.L().Warn("system.bootstrap announcements version query failed", "error", err)
		annVersion = ""
	}
	return fmt.Sprintf("%s:%s:%s", buildinfo.Version, s.provider.AppConfigVersion(ctx), annVersion)
}

// ── 管理端公告管理（admin 经 AnnouncementAdminStore 消费）──────────────
// 公告数据归 system 模块所有：写路径全部经本 Service 转发到唯一 Repository，
// 复用 Repository 内的公告版本缓存失效，不再需要 admin 自建 repo 副本或
// bootstrap 的跨实例失效接线。

func (s *Service) SearchAnnouncements(ctx context.Context, filter AnnouncementSearchFilter) (AnnouncementPage, error) {
	return s.repo.SearchAnnouncements(ctx, filter)
}

func (s *Service) GetAnnouncement(ctx context.Context, announcementID string) (AnnouncementRecord, bool, error) {
	return s.repo.GetAnnouncement(ctx, announcementID)
}

func (s *Service) CreateAnnouncement(ctx context.Context, input AnnouncementCreateInput) (AnnouncementRecord, error) {
	return s.repo.CreateAnnouncement(ctx, input)
}

func (s *Service) UpdateAnnouncement(ctx context.Context, input AnnouncementUpdateInput) (AnnouncementRecord, error) {
	return s.repo.UpdateAnnouncement(ctx, input)
}

func (s *Service) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	return s.repo.DeleteAnnouncement(ctx, announcementID)
}

var _ AnnouncementAdminStore = (*Service)(nil)

// InvalidateAnnouncementsVersion 转发给 Repository 的公告版本读缓存失效。
// 说明：公告写路径经本 Service 转发到唯一 Repository，Repository 内部已自行
// 失效缓存；本方法保留给显式失效场景（如外部直接改库后的运维兜底）。
// 内存仓库无缓存时为 no-op。
func (s *Service) InvalidateAnnouncementsVersion() {
	if invalidator, ok := s.repo.(interface{ InvalidateAnnouncementsVersion() }); ok {
		invalidator.InvalidateAnnouncementsVersion()
	}
}
func (s *Service) GetBootstrapResolver() BootstrapResolver  { return s.bootstrap }
func (s *Service) SetBootstrapResolver(r BootstrapResolver) { s.bootstrap = r }
