package system

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
)

type Provider interface {
	GetClientConfig(context.Context) ClientConfig
	GetSystemAnnouncementsCount(context.Context) int
	// AppConfigVersion 返回 AppConfig 的版本指纹（如 updated_at）：管理员
	// 每次修改配置都会变化，供 bootstrap_version 组装缓存失效信号
	//（spec api_spec_system.md 缓存契约：修改 AppConfig 时递增）。
	AppConfigVersion(context.Context) string
}

type BootstrapResolver interface {
	ResolveProfile(ctx context.Context) (any, bool)
	ResolveUnreadSummary(ctx context.Context) UnreadSummary
}

type AppConfigProvider struct {
	source appconfig.Source
}

func NewAppConfigProvider(source appconfig.Source) *AppConfigProvider {
	return &AppConfigProvider{source: source}
}

func (p *AppConfigProvider) GetClientConfig(ctx context.Context) ClientConfig {
	cfg := p.source.Get(ctx)
	return ClientConfig{
		Features: FeatureSwitches{
			EnablePublicRegistration: cfg.Features.EnablePublicRegistration,
			EnableGuestRegister:      cfg.Features.EnableGuestRegister,
			EnableMoments:            cfg.Features.EnableMoments,
			EnableNearbyPeople:       cfg.Features.EnableNearbyPeople,
		},
		// 投影必须覆盖 AppLimits 全部 12 字段（spec api_spec_types.md）：
		// 客户端撤回窗口/陌生人配额/表情白名单等校验依赖这些限额。
		Limits: AppLimits{
			MaxGroupMembers:            cfg.Limits.MaxGroupMembers,
			MaxMomentTags:              cfg.Limits.MaxMomentTags,
			MaxMomentImages:            cfg.Limits.MaxMomentImages,
			MaxChatFiles:               cfg.Limits.MaxChatFiles,
			MomentDescriptionMaxLength: cfg.Limits.MomentDescriptionMaxLength,
			CommentMaxLength:           cfg.Limits.CommentMaxLength,
			SearchRadiusMax:            cfg.Limits.SearchRadiusMax,
			MessageRecallWindowSeconds: cfg.Limits.MessageRecallWindowSeconds,
			StrangerMessageDailyQuota:  cfg.Limits.StrangerMessageDailyQuota,
			ReactionEmojis:             cfg.Limits.ReactionEmojis,
			GroupInviteTTLHours:        cfg.Limits.GroupInviteTTLHours,
			GroupApplicationTTLHours:   cfg.Limits.GroupApplicationTTLHours,
		},
		Contact: ContactInfo{
			SupportEmail:      cfg.Contact.SupportEmail,
			PrivacyPolicyURL:  cfg.Contact.PrivacyPolicyURL,
			TermsOfServiceURL: cfg.Contact.TermsOfServiceURL,
			OfficialWebsite:   cfg.Contact.OfficialWebsite,
		},
		Defaults: ClientDefaults{
			DefaultLocation: Location{
				Point: GeoPoint{
					Latitude:  cfg.Defaults.DefaultLocation.Point.Latitude,
					Longitude: cfg.Defaults.DefaultLocation.Point.Longitude,
				},
				LocationName: cfg.Defaults.DefaultLocation.LocationName,
			},
			DefaultUserAvatar:  cfg.Defaults.DefaultUserAvatar,
			DefaultGroupAvatar: cfg.Defaults.DefaultGroupAvatar,
		},
	}
}

func (p *AppConfigProvider) GetSystemAnnouncementsCount(ctx context.Context) int {
	cfg := p.source.Get(ctx)
	if cfg.ResponseDefaults.SystemAnnouncementsCount <= 0 {
		return 10
	}
	return cfg.ResponseDefaults.SystemAnnouncementsCount
}

// AppConfigVersion 以 AppConfig.UpdatedAt 为指纹：PersistenceSource 每次
// Update 都会顶新该时间戳（platform/appconfig/source.go），未加载/静态
// 源为空串（该维度恒定，不影响其余维度的失效检测）。
func (p *AppConfigProvider) AppConfigVersion(ctx context.Context) string {
	return p.source.Get(ctx).UpdatedAt
}
