package admin

import (
	"context"
	"time"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/httpx"
)

type Provider interface {
	GetAppConfig(context.Context) (AppConfig, error)
	UpdateAppConfig(context.Context, string, func(AppConfig) (AppConfig, *httpx.APIError)) (AppConfig, *httpx.APIError, error)
}

type AppConfigProvider struct {
	source appconfig.MutableSource
}

func NewAppConfigProvider(source appconfig.MutableSource) *AppConfigProvider {
	return &AppConfigProvider{source: source}
}

func (p *AppConfigProvider) GetAppConfig(ctx context.Context) (AppConfig, error) {
	return adminAppConfigFromSource(p.source.Get(ctx)), nil
}

func (p *AppConfigProvider) UpdateAppConfig(ctx context.Context, updatedByUserID string, apply func(AppConfig) (AppConfig, *httpx.APIError)) (AppConfig, *httpx.APIError, error) {
	current := adminAppConfigFromSource(p.source.Get(ctx))
	next, apiErr := apply(current)
	if apiErr != nil {
		return AppConfig{}, apiErr, nil
	}
	next.UpdatedByUserID = updatedByUserID
	if next.UpdatedAt == "" {
		next.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	updated, err := p.source.Update(ctx, func(appconfig.AppConfig) appconfig.AppConfig {
		return sourceAppConfigFromAdmin(next)
	})
	if err != nil {
		return AppConfig{}, nil, err
	}
	return adminAppConfigFromSource(updated), nil, nil
}

func adminAppConfigFromSource(cfg appconfig.AppConfig) AppConfig {
	return AppConfig{
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
			ImageMaxSize:      cfg.Upload.ImageMaxSize,
			VideoMaxSize:      cfg.Upload.VideoMaxSize,
			AudioMaxSize:      cfg.Upload.AudioMaxSize,
			FileMaxSize:       cfg.Upload.FileMaxSize,
			AllowedImageTypes: append([]string(nil), cfg.Upload.AllowedImageTypes...),
			AllowedVideoTypes: append([]string(nil), cfg.Upload.AllowedVideoTypes...),
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
				JoinMethod:   groupsmodule.GroupJoinMethod(cfg.Defaults.DefaultGroupSettings.JoinMethod),
				IsSearchable: cfg.Defaults.DefaultGroupSettings.IsSearchable,
				MaxMembers:   cfg.Defaults.DefaultGroupSettings.MaxMembers,
				Location: Location{
					Point: GeoPoint{
						Latitude:  cfg.Defaults.DefaultGroupSettings.Location.Point.Latitude,
						Longitude: cfg.Defaults.DefaultGroupSettings.Location.Point.Longitude,
					},
					LocationName: cfg.Defaults.DefaultGroupSettings.Location.LocationName,
				},
			},
		},
		RateLimits: RateLimitConfig{
			GlobalPerHour:            cfg.RateLimits.GlobalPerHour,
			UserPerHour:              cfg.RateLimits.UserPerHour,
			AuthRegisterPer10Minutes: cfg.RateLimits.AuthRegisterPer10Minutes,
			AuthLoginPer10Minutes:    cfg.RateLimits.AuthLoginPer10Minutes,
			AuthGuestPer10Minutes:    cfg.RateLimits.AuthGuestPer10Minutes,
			AuthRefreshPer10Minutes:  cfg.RateLimits.AuthRefreshPer10Minutes,
			AuthResetPer10Minutes:    cfg.RateLimits.AuthResetPer10Minutes,
			SearchPerMinute:          cfg.RateLimits.SearchPerMinute,
			WritePerMinute:           cfg.RateLimits.WritePerMinute,
			MomentsCreatePerMinute:   cfg.RateLimits.MomentsCreatePerMinute,
			CommentsCreatePerMinute:  cfg.RateLimits.CommentsCreatePerMinute,
			ReportsCreatePerHour:     cfg.RateLimits.ReportsCreatePerHour,
			UploadPresignPerHour:     cfg.RateLimits.UploadPresignPerHour,
			WSSubscribePerSecond:     cfg.RateLimits.WSSubscribePerSecond,
			AdminReadPerMinute:       cfg.RateLimits.AdminReadPerMinute,
			AdminWritePerMinute:      cfg.RateLimits.AdminWritePerMinute,
		},
		ResponseDefaults: ResponseDefaults{
			SearchSuggestCount:       cfg.ResponseDefaults.SearchSuggestCount,
			HotMomentsCount:          cfg.ResponseDefaults.HotMomentsCount,
			ChatRecentMessagesCount:  cfg.ResponseDefaults.ChatRecentMessagesCount,
			GroupAnnouncementsCount:  cfg.ResponseDefaults.GroupAnnouncementsCount,
			SystemAnnouncementsCount: cfg.ResponseDefaults.SystemAnnouncementsCount,
			CommentTopRepliesCount:   cfg.ResponseDefaults.CommentTopRepliesCount,
			UserProfileMomentsCount:  cfg.ResponseDefaults.UserProfileMomentsCount,
		},
		UpdatedAt:       cfg.UpdatedAt,
		UpdatedByUserID: cfg.UpdatedByUserID,
	}
}

func sourceAppConfigFromAdmin(cfg AppConfig) appconfig.AppConfig {
	return appconfig.AppConfig{
		Features: appconfig.Features{
			EnablePublicRegistration: cfg.Features.EnablePublicRegistration,
			EnableGuestLogin:         cfg.Features.EnableGuestLogin,
			EnableMoments:            cfg.Features.EnableMoments,
			EnableNearbyPeople:       cfg.Features.EnableNearbyPeople,
		},
		Limits: appconfig.Limits{
			MaxGroupMembers:            cfg.Limits.MaxGroupMembers,
			MaxMomentTags:              cfg.Limits.MaxMomentTags,
			MaxMomentImages:            cfg.Limits.MaxMomentImages,
			MomentDescriptionMaxLength: cfg.Limits.MomentDescriptionMaxLength,
			CommentMaxLength:           cfg.Limits.CommentMaxLength,
			SearchRadiusMax:            cfg.Limits.SearchRadiusMax,
		},
		Contact: appconfig.Contact{
			SupportEmail:      cfg.Contact.SupportEmail,
			PrivacyPolicyURL:  cfg.Contact.PrivacyPolicyURL,
			TermsOfServiceURL: cfg.Contact.TermsOfServiceURL,
			OfficialWebsite:   cfg.Contact.OfficialWebsite,
		},
		Upload: appconfig.UploadConfig{
			ImageMaxSize:      cfg.Upload.ImageMaxSize,
			VideoMaxSize:      cfg.Upload.VideoMaxSize,
			AudioMaxSize:      cfg.Upload.AudioMaxSize,
			FileMaxSize:       cfg.Upload.FileMaxSize,
			AllowedImageTypes: append([]string(nil), cfg.Upload.AllowedImageTypes...),
			AllowedVideoTypes: append([]string(nil), cfg.Upload.AllowedVideoTypes...),
		},
		Defaults: appconfig.Defaults{
			DefaultLocation: appconfig.Location{
				Point: appconfig.GeoPoint{
					Latitude:  cfg.Defaults.DefaultLocation.Point.Latitude,
					Longitude: cfg.Defaults.DefaultLocation.Point.Longitude,
				},
				LocationName: cfg.Defaults.DefaultLocation.LocationName,
			},
			DefaultUserAvatar:  cfg.Defaults.DefaultUserAvatar,
			DefaultGroupAvatar: cfg.Defaults.DefaultGroupAvatar,
			DefaultGroupSettings: appconfig.GroupSettings{
				JoinMethod:   string(cfg.Defaults.DefaultGroupSettings.JoinMethod),
				IsSearchable: cfg.Defaults.DefaultGroupSettings.IsSearchable,
				MaxMembers:   cfg.Defaults.DefaultGroupSettings.MaxMembers,
				Location: appconfig.Location{
					Point: appconfig.GeoPoint{
						Latitude:  cfg.Defaults.DefaultGroupSettings.Location.Point.Latitude,
						Longitude: cfg.Defaults.DefaultGroupSettings.Location.Point.Longitude,
					},
					LocationName: cfg.Defaults.DefaultGroupSettings.Location.LocationName,
				},
			},
		},
		RateLimits: appconfig.RateLimitConfig{
			GlobalPerHour:            cfg.RateLimits.GlobalPerHour,
			UserPerHour:              cfg.RateLimits.UserPerHour,
			AuthRegisterPer10Minutes: cfg.RateLimits.AuthRegisterPer10Minutes,
			AuthLoginPer10Minutes:    cfg.RateLimits.AuthLoginPer10Minutes,
			AuthGuestPer10Minutes:    cfg.RateLimits.AuthGuestPer10Minutes,
			AuthRefreshPer10Minutes:  cfg.RateLimits.AuthRefreshPer10Minutes,
			AuthResetPer10Minutes:    cfg.RateLimits.AuthResetPer10Minutes,
			SearchPerMinute:          cfg.RateLimits.SearchPerMinute,
			WritePerMinute:           cfg.RateLimits.WritePerMinute,
			MomentsCreatePerMinute:   cfg.RateLimits.MomentsCreatePerMinute,
			CommentsCreatePerMinute:  cfg.RateLimits.CommentsCreatePerMinute,
			ReportsCreatePerHour:     cfg.RateLimits.ReportsCreatePerHour,
			UploadPresignPerHour:     cfg.RateLimits.UploadPresignPerHour,
			WSSubscribePerSecond:     cfg.RateLimits.WSSubscribePerSecond,
			AdminReadPerMinute:       cfg.RateLimits.AdminReadPerMinute,
			AdminWritePerMinute:      cfg.RateLimits.AdminWritePerMinute,
		},
		ResponseDefaults: appconfig.ResponseDefaults{
			SearchSuggestCount:       cfg.ResponseDefaults.SearchSuggestCount,
			HotMomentsCount:          cfg.ResponseDefaults.HotMomentsCount,
			ChatRecentMessagesCount:  cfg.ResponseDefaults.ChatRecentMessagesCount,
			GroupAnnouncementsCount:  cfg.ResponseDefaults.GroupAnnouncementsCount,
			SystemAnnouncementsCount: cfg.ResponseDefaults.SystemAnnouncementsCount,
			CommentTopRepliesCount:   cfg.ResponseDefaults.CommentTopRepliesCount,
			UserProfileMomentsCount:  cfg.ResponseDefaults.UserProfileMomentsCount,
		},
		UpdatedAt:       cfg.UpdatedAt,
		UpdatedByUserID: cfg.UpdatedByUserID,
	}
}
