package admin

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
)

type Provider interface {
	GetAppConfig(context.Context) (AppConfig, error)
}

type AppConfigProvider struct {
	source appconfig.Source
}

func NewAppConfigProvider(source appconfig.Source) *AppConfigProvider {
	return &AppConfigProvider{source: source}
}

func (p *AppConfigProvider) GetAppConfig(ctx context.Context) (AppConfig, error) {
	cfg := p.source.Get(ctx)
	appCfg := AppConfig{
		Features: FeatureSwitches{
			EnablePublicRegistration: cfg.Features.EnablePublicRegistration,
			EnableGuestLogin:         cfg.Features.EnableGuestLogin,
			EnableMoments:            cfg.Features.EnableMoments,
			EnableNearbyPeople:       cfg.Features.EnableNearbyPeople,
		},
		Limits: AppLimits{
			MaxGroupMembers:            cfg.Limits.MaxGroupMembers,
			MaxMomentTags:              cfg.Limits.MaxMomentTags,
			MaxMomentImages:            cfg.Limits.MaxMomentImages,
			MomentDescriptionMaxLength: cfg.Limits.MomentDescriptionMaxLength,
			CommentMaxLength:           cfg.Limits.CommentMaxLength,
			SearchRadiusMax:            cfg.Limits.SearchRadiusMax,
		},
		Contact: ContactInfo{
			SupportEmail:      cfg.Contact.SupportEmail,
			PrivacyPolicyURL:  cfg.Contact.PrivacyPolicyURL,
			TermsOfServiceURL: cfg.Contact.TermsOfServiceURL,
			OfficialWebsite:   cfg.Contact.OfficialWebsite,
		},
		Upload: UploadConfig{
			ImageMaxSize:      10 * 1024 * 1024,
			VideoMaxSize:      100 * 1024 * 1024,
			AudioMaxSize:      20 * 1024 * 1024,
			FileMaxSize:       50 * 1024 * 1024,
			AllowedImageTypes: []string{"image/jpeg", "image/png", "image/webp"},
			AllowedVideoTypes: []string{"video/mp4", "video/quicktime"},
		},
		Defaults: ContentDefaults{
			DefaultLocation: Location{
				Point: GeoPoint{
					Latitude:  cfg.Defaults.DefaultLocation.Point.Latitude,
					Longitude: cfg.Defaults.DefaultLocation.Point.Longitude,
				},
				LocationName: cfg.Defaults.DefaultLocation.LocationName,
			},
			DefaultGroupAvatar: cfg.Defaults.DefaultGroupAvatar,
			DefaultUserAvatar:  cfg.Defaults.DefaultUserAvatar,
			DefaultGroupSettings: GroupSettings{
				JoinMethod:   "open",
				IsSearchable: true,
				MaxMembers:   cfg.Limits.MaxGroupMembers,
				Location: Location{
					Point: GeoPoint{
						Latitude:  cfg.Defaults.DefaultLocation.Point.Latitude,
						Longitude: cfg.Defaults.DefaultLocation.Point.Longitude,
					},
					LocationName: cfg.Defaults.DefaultLocation.LocationName,
				},
			},
		},
		RateLimits: RateLimitConfig{
			GlobalPerHour:            10000,
			UserPerHour:              1000,
			AuthRegisterPer10Minutes: 20,
			AuthLoginPer10Minutes:    60,
			AuthGuestPer10Minutes:    60,
			AuthRefreshPer10Minutes:  120,
			AuthResetPer10Minutes:    20,
			SearchPerMinute:          60,
			WritePerMinute:           30,
			MomentsCreatePerMinute:   10,
			CommentsCreatePerMinute:  20,
			ReportsCreatePerHour:     20,
			UploadPresignPerHour:     100,
			WSSubscribePerSecond:     10,
			AdminReadPerMinute:       120,
			AdminWritePerMinute:      60,
		},
		ResponseDefaults: ResponseDefaults{
			SearchSuggestCount:        5,
			TrendingTagsCount:         10,
			HotMomentsCount:           20,
			ChatRecentMessagesCount:   cfg.ResponseDefaults.ChatRecentMessagesCount,
			GroupAnnouncementsCount:   10,
			SystemAnnouncementsCount:  cfg.ResponseDefaults.SystemAnnouncementsCount,
		},
	}
	return appCfg, nil
}
