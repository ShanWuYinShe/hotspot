package admin

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

func (s *Service) ListAuditLogs(ctx context.Context, req AuditLogListRequest) (PaginatedAuditLogs, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAuditRead)
	if apiErr != nil || err != nil {
		return PaginatedAuditLogs{}, apiErr, err
	}
	filter, apiErr := validateAuditFilter(req)
	if apiErr != nil {
		return PaginatedAuditLogs{}, apiErr, nil
	}
	page, err := s.repo.ListAuditLogs(ctx, filter, actor.HasPermission(permissionAuditReadIP))
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return PaginatedAuditLogs{}, apiErr, nil
		}
		return PaginatedAuditLogs{}, nil, err
	}
	resp := PaginatedAuditLogs{Items: page.Items}
	if page.Next != nil {
		resp.NextCursor = encodeAuditCursor(*page.Next)
	}
	return resp, nil, nil
}

func (s *Service) GetAuditLog(ctx context.Context, logID string) (AuditLogItem, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAuditRead)
	if apiErr != nil || err != nil {
		return AuditLogItem{}, apiErr, err
	}
	if strings.TrimSpace(logID) == "" {
		return AuditLogItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	item, found, err := s.repo.GetAuditLog(ctx, logID, actor.HasPermission(permissionAuditReadIP))
	if err != nil {
		return AuditLogItem{}, nil, err
	}
	if !found {
		return AuditLogItem{}, &httpx.APIError{Code: 10300, Message: "审计日志不存在"}, nil
	}
	return item, nil, nil
}

func (s *Service) GetAppConfig(ctx context.Context) (AppConfig, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionAppConfigRead); apiErr != nil || err != nil {
		return AppConfig{}, apiErr, err
	}
	cfg, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return AppConfig{}, nil, err
	}
	return cfg, nil, nil
}

func (s *Service) UpdateUpload(ctx context.Context, req UpdateUploadConfigRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.Upload.ImageMaxSize != nil {
			if *req.Upload.ImageMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.ImageMaxSize = *req.Upload.ImageMaxSize
		}
		if req.Upload.VideoMaxSize != nil {
			if *req.Upload.VideoMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.VideoMaxSize = *req.Upload.VideoMaxSize
		}
		if req.Upload.AudioMaxSize != nil {
			if *req.Upload.AudioMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.AudioMaxSize = *req.Upload.AudioMaxSize
		}
		if req.Upload.FileMaxSize != nil {
			if *req.Upload.FileMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.FileMaxSize = *req.Upload.FileMaxSize
		}
		if req.Upload.AllowedImageTypes != nil {
			cfg.Upload.AllowedImageTypes = sortAndUniqueStrings(req.Upload.AllowedImageTypes)
		}
		if req.Upload.AllowedVideoTypes != nil {
			cfg.Upload.AllowedVideoTypes = sortAndUniqueStrings(req.Upload.AllowedVideoTypes)
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "upload", before.Upload, updated.Upload)
}

func (s *Service) UpdateDefaults(ctx context.Context, req UpdateDefaultsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.DefaultLocation != nil {
			if req.DefaultLocation.Point != nil {
				cfg.Defaults.DefaultLocation.Point = *req.DefaultLocation.Point
			}
			if req.DefaultLocation.LocationName != nil {
				if strings.TrimSpace(*req.DefaultLocation.LocationName) == "" {
					return AppConfig{}, httpx.InvalidRequest("参数错误")
				}
				cfg.Defaults.DefaultLocation.LocationName = strings.TrimSpace(*req.DefaultLocation.LocationName)
			}
		}
		if req.DefaultGroupAvatar != nil {
			cfg.Defaults.DefaultGroupAvatar = strings.TrimSpace(*req.DefaultGroupAvatar)
		}
		if req.DefaultUserAvatar != nil {
			cfg.Defaults.DefaultUserAvatar = strings.TrimSpace(*req.DefaultUserAvatar)
		}
		if req.DefaultGroupSettings != nil {
			if req.DefaultGroupSettings.JoinMethod != nil {
				joinMethod := strings.TrimSpace(string(*req.DefaultGroupSettings.JoinMethod))
				if joinMethod == "" {
					return AppConfig{}, httpx.InvalidRequest("参数错误")
				}
				cfg.Defaults.DefaultGroupSettings.JoinMethod = *req.DefaultGroupSettings.JoinMethod
			}
			if req.DefaultGroupSettings.IsSearchable != nil {
				cfg.Defaults.DefaultGroupSettings.IsSearchable = *req.DefaultGroupSettings.IsSearchable
			}
			if req.DefaultGroupSettings.MaxMembers != nil {
				if *req.DefaultGroupSettings.MaxMembers <= 0 {
					return AppConfig{}, httpx.InvalidRequest("参数错误")
				}
				cfg.Defaults.DefaultGroupSettings.MaxMembers = *req.DefaultGroupSettings.MaxMembers
			}
			if req.DefaultGroupSettings.Location != nil {
				if req.DefaultGroupSettings.Location.Point != nil {
					cfg.Defaults.DefaultGroupSettings.Location.Point = *req.DefaultGroupSettings.Location.Point
				}
				if req.DefaultGroupSettings.Location.LocationName != nil {
					if strings.TrimSpace(*req.DefaultGroupSettings.Location.LocationName) == "" {
						return AppConfig{}, httpx.InvalidRequest("参数错误")
					}
					cfg.Defaults.DefaultGroupSettings.Location.LocationName = strings.TrimSpace(*req.DefaultGroupSettings.Location.LocationName)
				}
			}
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "defaults", before.Defaults, updated.Defaults)
}

func (s *Service) UpdateRateLimits(ctx context.Context, req UpdateRateLimitsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if err := applyPositiveIntPatch(&cfg.RateLimits.GlobalPerHour, req.RateLimits.GlobalPerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.UserPerHour, req.RateLimits.UserPerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthRegisterPer10Minutes, req.RateLimits.AuthRegisterPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthLoginPer10Minutes, req.RateLimits.AuthLoginPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthGuestPer10Minutes, req.RateLimits.AuthGuestPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthRefreshPer10Minutes, req.RateLimits.AuthRefreshPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthResetPer10Minutes, req.RateLimits.AuthResetPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.SearchPerMinute, req.RateLimits.SearchPerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.WritePerMinute, req.RateLimits.WritePerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.MomentsCreatePerMinute, req.RateLimits.MomentsCreatePerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.CommentsCreatePerMinute, req.RateLimits.CommentsCreatePerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.ReportsCreatePerHour, req.RateLimits.ReportsCreatePerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.UploadPresignPerHour, req.RateLimits.UploadPresignPerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.WSSubscribePerSecond, req.RateLimits.WSSubscribePerSecond); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AdminReadPerMinute, req.RateLimits.AdminReadPerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AdminWritePerMinute, req.RateLimits.AdminWritePerMinute); err != nil {
			return AppConfig{}, err
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "rate_limits", before.RateLimits, updated.RateLimits)
}

func (s *Service) UpdateResponseDefaults(ctx context.Context, req UpdateResponseDefaultsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.SearchSuggestCount, req.ResponseDefaults.SearchSuggestCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.TrendingTagsCount, req.ResponseDefaults.TrendingTagsCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.HotMomentsCount, req.ResponseDefaults.HotMomentsCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.ChatRecentMessagesCount, req.ResponseDefaults.ChatRecentMessagesCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.GroupAnnouncementsCount, req.ResponseDefaults.GroupAnnouncementsCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.SystemAnnouncementsCount, req.ResponseDefaults.SystemAnnouncementsCount); err != nil {
			return AppConfig{}, err
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "response_defaults", before.ResponseDefaults, updated.ResponseDefaults)
}

func (s *Service) UpdateFeatures(ctx context.Context, req UpdateFeaturesRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.EnablePublicRegistration != nil {
			cfg.Features.EnablePublicRegistration = *req.EnablePublicRegistration
		}
		if req.EnableGuestLogin != nil {
			cfg.Features.EnableGuestLogin = *req.EnableGuestLogin
		}
		if req.EnableMoments != nil {
			cfg.Features.EnableMoments = *req.EnableMoments
		}
		if req.EnableNearbyPeople != nil {
			cfg.Features.EnableNearbyPeople = *req.EnableNearbyPeople
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "features", before.Features, updated.Features)
}

func (s *Service) UpdateLimits(ctx context.Context, req UpdateLimitsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if err := applyPositiveIntPatch(&cfg.Limits.MaxGroupMembers, req.Limits.MaxGroupMembers); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.MaxMomentTags, req.Limits.MaxMomentTags); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.MaxMomentImages, req.Limits.MaxMomentImages); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.MomentDescriptionMaxLength, req.Limits.MomentDescriptionMaxLength); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.CommentMaxLength, req.Limits.CommentMaxLength); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.SearchRadiusMax, req.Limits.SearchRadiusMax); err != nil {
			return AppConfig{}, err
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "limits", before.Limits, updated.Limits)
}

func (s *Service) UpdateContact(ctx context.Context, req UpdateContactRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.SupportEmail != nil {
			if !strings.Contains(strings.TrimSpace(*req.SupportEmail), "@") {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Contact.SupportEmail = strings.TrimSpace(*req.SupportEmail)
		}
		if req.PrivacyPolicyURL != nil {
			cfg.Contact.PrivacyPolicyURL = strings.TrimSpace(*req.PrivacyPolicyURL)
		}
		if req.TermsOfServiceURL != nil {
			cfg.Contact.TermsOfServiceURL = strings.TrimSpace(*req.TermsOfServiceURL)
		}
		if req.OfficialWebsite != nil {
			cfg.Contact.OfficialWebsite = strings.TrimSpace(*req.OfficialWebsite)
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "contact", before.Contact, updated.Contact)
}

func validateAuditFilter(req AuditLogListRequest) (AuditLogFilter, *httpx.APIError) {
	filter := AuditLogFilter{Action: req.Action, OperatorID: strings.TrimSpace(req.OperatorID), TargetType: req.TargetType, TargetID: strings.TrimSpace(req.TargetID)}
	if req.Pagination != nil {
		filter.Pagination = *req.Pagination
	}
	if strings.TrimSpace(req.StartTime) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartTime))
		if err != nil {
			return AuditLogFilter{}, httpx.InvalidRequest("参数错误")
		}
		filter.StartTime = &parsed
	}
	if strings.TrimSpace(req.EndTime) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndTime))
		if err != nil {
			return AuditLogFilter{}, httpx.InvalidRequest("参数错误")
		}
		filter.EndTime = &parsed
	}
	if filter.StartTime != nil && filter.EndTime != nil && filter.StartTime.After(*filter.EndTime) {
		return AuditLogFilter{}, httpx.InvalidRequest("参数错误")
	}
	if _, apiErr := decodeAuditCursor(filter.Pagination.Cursor); apiErr != nil {
		return AuditLogFilter{}, apiErr
	}
	return filter, nil
}

func (s *Service) appendConfigAudit(ctx context.Context, actor AdminActor, configKey string, oldValue, newValue any) error {
	return s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
		Action:     AuditActionConfigUpdate,
		TargetType: AuditTargetTypeConfig,
		TargetID:   configKey,
		TargetName: configKey,
		Details: AuditDetails{
			"type":       string(AuditActionConfigUpdate),
			"config_key": configKey,
			"old_value":  oldValue,
			"new_value":  newValue,
		},
	}))
}
