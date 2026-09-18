package system

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
)

type Provider interface {
	GetClientConfig(context.Context) ClientConfig
	GetSystemAnnouncementsCount(context.Context) int
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
