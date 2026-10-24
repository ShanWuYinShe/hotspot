package admin

import (
	"context"
	"time"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

type Provider interface {
	GetAppConfig(context.Context) (AdminAppConfig, error)
	UpdateAppConfig(context.Context, string, func(AdminAppConfig) (AdminAppConfig, *httpx.APIError)) (AdminAppConfig, *httpx.APIError, error)
}

type AppConfigProvider struct {
	source appconfig.MutableSource
}

func NewAppConfigProvider(source appconfig.MutableSource) *AppConfigProvider {
	return &AppConfigProvider{source: source}
}

func (p *AppConfigProvider) GetAppConfig(ctx context.Context) (AdminAppConfig, error) {
	return adminAppConfigFromSource(p.source.Get(ctx)), nil
}

func (p *AppConfigProvider) UpdateAppConfig(ctx context.Context, updatedByUserID string, apply func(AdminAppConfig) (AdminAppConfig, *httpx.APIError)) (AdminAppConfig, *httpx.APIError, error) {
	// 先对当前快照跑一次校验：apply 内的校验全部是绝对值校验（正数/非空/
	// 枚举），不依赖现有配置取值——失败在触碰 Update 写路径前直接拒绝，
	// 避免无效请求也把 payload 原样写库（PersistenceSource.Update 无条件
	// upsert，会把 updated_at 一并顶新）。
	if _, apiErr := apply(adminAppConfigFromSource(p.source.Get(ctx))); apiErr != nil {
		return AdminAppConfig{}, apiErr, nil
	}
	var applyErr *httpx.APIError
	updated, err := p.source.Update(ctx, func(cur appconfig.AppConfig) appconfig.AppConfig {
		// 增量合并必须在 Update 回调内基于传入的最新 cur 进行：此前这里
		// 写回的是 Get 时刻的整份旧快照，两个管理员并发改不同分区时后写
		// 者会把先写者的改动整份回滚（审查 P1-4）。
		next, apiErr := apply(adminAppConfigFromSource(cur))
		if apiErr != nil {
			// 理论不可达（校验已在上方完成，且均为绝对值校验）；
			// 防御性回退：原样返回 cur，不落任何改动。
			applyErr = apiErr
			return cur
		}
		now := time.Now().UTC().Format(time.RFC3339)
		next.UpdatedAt = &now
		next.UpdatedByUserID = &updatedByUserID
		return sourceAppConfigFromAdmin(next)
	})
	if applyErr != nil {
		return AdminAppConfig{}, applyErr, nil
	}
	if err != nil {
		return AdminAppConfig{}, nil, err
	}
	return adminAppConfigFromSource(updated), nil, nil
}

func adminAppConfigFromSource(cfg appconfig.AppConfig) AdminAppConfig {
	return AdminAppConfig{
		Features: FeatureSwitches{
			EnablePublicRegistration: cfg.Features.EnablePublicRegistration,
			EnableGuestRegister:      cfg.Features.EnableGuestRegister,
			EnableMoments:            cfg.Features.EnableMoments,
			EnableNearbyPeople:       cfg.Features.EnableNearbyPeople,
		},
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
			ReactionEmojis:             append([]string(nil), cfg.Limits.ReactionEmojis...),
			GroupInviteTTLHours:        cfg.Limits.GroupInviteTTLHours,
			GroupApplicationTTLHours:   cfg.Limits.GroupApplicationTTLHours,
		},
		Contact: ContactInfo{
			SupportEmail:      cfg.Contact.SupportEmail,
			PrivacyPolicyURL:  cfg.Contact.PrivacyPolicyURL,
			TermsOfServiceURL: cfg.Contact.TermsOfServiceURL,
			OfficialWebsite:   cfg.Contact.OfficialWebsite,
		},
		Upload: AdminUploadConfig{
			ImageMaxSize:      cfg.Upload.ImageMaxSize,
			VideoMaxSize:      cfg.Upload.VideoMaxSize,
			AudioMaxSize:      cfg.Upload.AudioMaxSize,
			FileMaxSize:       cfg.Upload.FileMaxSize,
			AvatarMaxSize:     cfg.Upload.AvatarMaxSize,
			ChatFileMaxSize:   cfg.Upload.ChatFileMaxSize,
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
			DefaultGroupSettings: AdminGroupSettings{
				JoinMethod:        groupsmodule.GroupJoinMethod(cfg.Defaults.DefaultGroupSettings.JoinMethod),
				RequiresApproval:  cfg.Defaults.DefaultGroupSettings.RequiresApproval,
				HasPassword:       cfg.Defaults.DefaultGroupSettings.HasPassword,
				IsSearchable:      cfg.Defaults.DefaultGroupSettings.IsSearchable,
				MembersSearchable: cfg.Defaults.DefaultGroupSettings.MembersSearchable,
				MaxMembers:        cfg.Defaults.DefaultGroupSettings.MaxMembers,
				Location: Location{
					Point: GeoPoint{
						Latitude:  cfg.Defaults.DefaultGroupSettings.Location.Point.Latitude,
						Longitude: cfg.Defaults.DefaultGroupSettings.Location.Point.Longitude,
					},
					LocationName: cfg.Defaults.DefaultGroupSettings.Location.LocationName,
				},
			},
		},
		RateLimits: AdminRateLimitConfig{
			GlobalPerHour:            cfg.RateLimits.GlobalPerHour,
			UserPerHour:              cfg.RateLimits.UserPerHour,
			AuthRegisterPer10Minutes: cfg.RateLimits.AuthRegisterPer10Minutes,
			AuthSigninPer10Minutes:   cfg.RateLimits.AuthSigninPer10Minutes,
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
			WSHandshakePerMinute:     cfg.RateLimits.WSHandshakePerMinute,
			AdminReadPerMinute:       cfg.RateLimits.AdminReadPerMinute,
			AdminWritePerMinute:      cfg.RateLimits.AdminWritePerMinute,
		},
		ResponseDefaults: AdminResponseDefaults{
			SearchSuggestCount:       cfg.ResponseDefaults.SearchSuggestCount,
			HotMomentsCount:          cfg.ResponseDefaults.HotMomentsCount,
			ChatRecentMessagesCount:  cfg.ResponseDefaults.ChatRecentMessagesCount,
			GroupAnnouncementsCount:  cfg.ResponseDefaults.GroupAnnouncementsCount,
			SystemAnnouncementsCount: cfg.ResponseDefaults.SystemAnnouncementsCount,
			CommentTopRepliesCount:   cfg.ResponseDefaults.CommentTopRepliesCount,
			UserProfileMomentsCount:  cfg.ResponseDefaults.UserProfileMomentsCount,
		},
		Auth: AdminAuthConfig{
			VerificationCodeTTLMinutes: cfg.Auth.VerificationCodeTTLMinutes,
			LoginLockoutThreshold:      cfg.Auth.LoginLockoutThreshold,
			LoginLockoutMinutes:        cfg.Auth.LoginLockoutMinutes,
		},
		UpdatedAt:       types.StrPtr(cfg.UpdatedAt),
		UpdatedByUserID: types.StrPtr(cfg.UpdatedByUserID),
	}
}

func sourceAppConfigFromAdmin(cfg AdminAppConfig) appconfig.AppConfig {
	return appconfig.AppConfig{
		Features: appconfig.Features{
			EnablePublicRegistration: cfg.Features.EnablePublicRegistration,
			EnableGuestRegister:      cfg.Features.EnableGuestRegister,
			EnableMoments:            cfg.Features.EnableMoments,
			EnableNearbyPeople:       cfg.Features.EnableNearbyPeople,
		},
		Limits: appconfig.Limits{
			MaxGroupMembers:            cfg.Limits.MaxGroupMembers,
			MaxMomentTags:              cfg.Limits.MaxMomentTags,
			MaxMomentImages:            cfg.Limits.MaxMomentImages,
			MaxChatFiles:               cfg.Limits.MaxChatFiles,
			MomentDescriptionMaxLength: cfg.Limits.MomentDescriptionMaxLength,
			CommentMaxLength:           cfg.Limits.CommentMaxLength,
			SearchRadiusMax:            cfg.Limits.SearchRadiusMax,
			MessageRecallWindowSeconds: cfg.Limits.MessageRecallWindowSeconds,
			StrangerMessageDailyQuota:  cfg.Limits.StrangerMessageDailyQuota,
			ReactionEmojis:             append([]string(nil), cfg.Limits.ReactionEmojis...),
			GroupInviteTTLHours:        cfg.Limits.GroupInviteTTLHours,
			GroupApplicationTTLHours:   cfg.Limits.GroupApplicationTTLHours,
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
			AvatarMaxSize:     cfg.Upload.AvatarMaxSize,
			ChatFileMaxSize:   cfg.Upload.ChatFileMaxSize,
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
				JoinMethod:        string(cfg.Defaults.DefaultGroupSettings.JoinMethod),
				RequiresApproval:  cfg.Defaults.DefaultGroupSettings.RequiresApproval,
				HasPassword:       cfg.Defaults.DefaultGroupSettings.HasPassword,
				IsSearchable:      cfg.Defaults.DefaultGroupSettings.IsSearchable,
				MembersSearchable: cfg.Defaults.DefaultGroupSettings.MembersSearchable,
				MaxMembers:        cfg.Defaults.DefaultGroupSettings.MaxMembers,
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
			AuthSigninPer10Minutes:   cfg.RateLimits.AuthSigninPer10Minutes,
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
			WSHandshakePerMinute:     cfg.RateLimits.WSHandshakePerMinute,
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
		Auth: appconfig.AuthConfig{
			VerificationCodeTTLMinutes: cfg.Auth.VerificationCodeTTLMinutes,
			LoginLockoutThreshold:      cfg.Auth.LoginLockoutThreshold,
			LoginLockoutMinutes:        cfg.Auth.LoginLockoutMinutes,
		},
		UpdatedAt:       strDeref(cfg.UpdatedAt),
		UpdatedByUserID: strDeref(cfg.UpdatedByUserID),
	}
}

func strDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
