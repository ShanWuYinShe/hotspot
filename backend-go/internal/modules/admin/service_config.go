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
		if apiErr := asAPIError(err); apiErr != nil {
			return PaginatedAuditLogs{}, apiErr, nil
		}
		return PaginatedAuditLogs{}, nil, err
	}
	resp := PaginatedAuditLogs{Items: page.Items}
	if page.Next != nil {
		c := encodeAuditCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

func (s *Service) GetAppConfig(ctx context.Context) (AdminAppConfig, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionAppConfigRead); apiErr != nil || err != nil {
		return AdminAppConfig{}, apiErr, err
	}
	cfg, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return AdminAppConfig{}, nil, err
	}
	return cfg, nil, nil
}

func (s *Service) UpdateAppConfig(ctx context.Context, req UpdateAppConfigRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	// At least one section must be provided
	if req.Upload == nil && req.Defaults == nil && req.RateLimits == nil && req.ResponseDefaults == nil && req.Features == nil && req.Limits == nil && req.Contact == nil {
		return httpx.InvalidRequest("参数错误"), nil
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AdminAppConfig) (AdminAppConfig, *httpx.APIError) {
		if req.Upload != nil {
			if err := applyUploadConfigPatch(&cfg.Upload, *req.Upload); err != nil {
				return AdminAppConfig{}, err
			}
		}
		if req.Defaults != nil {
			if err := applyDefaultsPatch(&cfg.Defaults, *req.Defaults); err != nil {
				return AdminAppConfig{}, err
			}
		}
		if req.RateLimits != nil {
			if err := applyRateLimitsPatch(&cfg.RateLimits, *req.RateLimits); err != nil {
				return AdminAppConfig{}, err
			}
		}
		if req.ResponseDefaults != nil {
			if err := applyResponseDefaultsPatch(&cfg.ResponseDefaults, *req.ResponseDefaults); err != nil {
				return AdminAppConfig{}, err
			}
		}
		if req.Features != nil {
			applyFeaturesPatch(&cfg.Features, *req.Features)
		}
		if req.Limits != nil {
			if err := applyLimitsPatch(&cfg.Limits, *req.Limits); err != nil {
				return AdminAppConfig{}, err
			}
		}
		if req.Contact != nil {
			if err := applyContactPatch(&cfg.Contact, *req.Contact); err != nil {
				return AdminAppConfig{}, err
			}
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "app_config", before, updated)
}

// --- Patch helper functions for unified UpdateAppConfig ---

func applyUploadConfigPatch(target *AdminUploadConfig, patch PartialUploadConfig) *httpx.APIError {
	if patch.ImageMaxSize != nil {
		if *patch.ImageMaxSize <= 0 {
			return httpx.InvalidRequest("参数错误")
		}
		target.ImageMaxSize = *patch.ImageMaxSize
	}
	if patch.VideoMaxSize != nil {
		if *patch.VideoMaxSize <= 0 {
			return httpx.InvalidRequest("参数错误")
		}
		target.VideoMaxSize = *patch.VideoMaxSize
	}
	if patch.AudioMaxSize != nil {
		if *patch.AudioMaxSize <= 0 {
			return httpx.InvalidRequest("参数错误")
		}
		target.AudioMaxSize = *patch.AudioMaxSize
	}
	if patch.FileMaxSize != nil {
		if *patch.FileMaxSize <= 0 {
			return httpx.InvalidRequest("参数错误")
		}
		target.FileMaxSize = *patch.FileMaxSize
	}
	if patch.AllowedImageTypes != nil {
		target.AllowedImageTypes = sortAndUniqueStrings(patch.AllowedImageTypes)
	}
	if patch.AllowedVideoTypes != nil {
		target.AllowedVideoTypes = sortAndUniqueStrings(patch.AllowedVideoTypes)
	}
	return nil
}

func applyDefaultsPatch(target *ContentDefaults, patch UpdateDefaultsPayload) *httpx.APIError {
	if patch.DefaultLocation != nil {
		if patch.DefaultLocation.Point != nil {
			target.DefaultLocation.Point = *patch.DefaultLocation.Point
		}
		if patch.DefaultLocation.LocationName != nil {
			if strings.TrimSpace(*patch.DefaultLocation.LocationName) == "" {
				return httpx.InvalidRequest("参数错误")
			}
			target.DefaultLocation.LocationName = strings.TrimSpace(*patch.DefaultLocation.LocationName)
		}
	}
	if patch.DefaultGroupAvatar != nil {
		target.DefaultGroupAvatar = strings.TrimSpace(*patch.DefaultGroupAvatar)
	}
	if patch.DefaultUserAvatar != nil {
		target.DefaultUserAvatar = strings.TrimSpace(*patch.DefaultUserAvatar)
	}
	if patch.DefaultGroupSettings != nil {
		gs := patch.DefaultGroupSettings
		if gs.JoinMethod != nil {
			if strings.TrimSpace(string(*gs.JoinMethod)) == "" {
				return httpx.InvalidRequest("参数错误")
			}
			target.DefaultGroupSettings.JoinMethod = *gs.JoinMethod
		}
		if gs.IsSearchable != nil {
			target.DefaultGroupSettings.IsSearchable = *gs.IsSearchable
		}
		if gs.MaxMembers != nil {
			if *gs.MaxMembers <= 0 {
				return httpx.InvalidRequest("参数错误")
			}
			target.DefaultGroupSettings.MaxMembers = *gs.MaxMembers
		}
		if gs.Location != nil {
			if gs.Location.Point != nil {
				target.DefaultGroupSettings.Location.Point = *gs.Location.Point
			}
			if gs.Location.LocationName != nil {
				if strings.TrimSpace(*gs.Location.LocationName) == "" {
					return httpx.InvalidRequest("参数错误")
				}
				target.DefaultGroupSettings.Location.LocationName = strings.TrimSpace(*gs.Location.LocationName)
			}
		}
	}
	return nil
}

func applyRateLimitsPatch(target *AdminRateLimitConfig, patch PartialRateLimitConfig) *httpx.APIError {
	fields := []struct {
		target *int
		patch  *int
	}{
		{&target.GlobalPerHour, patch.GlobalPerHour},
		{&target.UserPerHour, patch.UserPerHour},
		{&target.AuthRegisterPer10Minutes, patch.AuthRegisterPer10Minutes},
		{&target.AuthSigninPer10Minutes, patch.AuthSigninPer10Minutes},
		{&target.AuthGuestPer10Minutes, patch.AuthGuestPer10Minutes},
		{&target.AuthRefreshPer10Minutes, patch.AuthRefreshPer10Minutes},
		{&target.AuthResetPer10Minutes, patch.AuthResetPer10Minutes},
		{&target.SearchPerMinute, patch.SearchPerMinute},
		{&target.WritePerMinute, patch.WritePerMinute},
		{&target.MomentsCreatePerMinute, patch.MomentsCreatePerMinute},
		{&target.CommentsCreatePerMinute, patch.CommentsCreatePerMinute},
		{&target.ReportsCreatePerHour, patch.ReportsCreatePerHour},
		{&target.UploadPresignPerHour, patch.UploadPresignPerHour},
		{&target.WSSubscribePerSecond, patch.WSSubscribePerSecond},
		{&target.WSHandshakePerMinute, patch.WSHandshakePerMinute},
		{&target.AdminReadPerMinute, patch.AdminReadPerMinute},
		{&target.AdminWritePerMinute, patch.AdminWritePerMinute},
	}
	for _, f := range fields {
		if apiErr := applyPositiveIntPatch(f.target, f.patch); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func applyResponseDefaultsPatch(target *AdminResponseDefaults, patch PartialResponseDefaults) *httpx.APIError {
	fields := []struct {
		target *int
		patch  *int
	}{
		{&target.SearchSuggestCount, patch.SearchSuggestCount},
		{&target.HotMomentsCount, patch.HotMomentsCount},
		{&target.ChatRecentMessagesCount, patch.ChatRecentMessagesCount},
		{&target.GroupAnnouncementsCount, patch.GroupAnnouncementsCount},
		{&target.SystemAnnouncementsCount, patch.SystemAnnouncementsCount},
		{&target.CommentTopRepliesCount, patch.CommentTopRepliesCount},
		{&target.UserProfileMomentsCount, patch.UserProfileMomentsCount},
	}
	for _, f := range fields {
		if apiErr := applyPositiveIntPatch(f.target, f.patch); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func applyFeaturesPatch(target *FeatureSwitches, patch PartialFeatureSwitches) {
	if patch.EnablePublicRegistration != nil {
		target.EnablePublicRegistration = *patch.EnablePublicRegistration
	}
	if patch.EnableGuestRegister != nil {
		target.EnableGuestRegister = *patch.EnableGuestRegister
	}
	if patch.EnableMoments != nil {
		target.EnableMoments = *patch.EnableMoments
	}
	if patch.EnableNearbyPeople != nil {
		target.EnableNearbyPeople = *patch.EnableNearbyPeople
	}
}

func applyLimitsPatch(target *AppLimits, patch PartialAppLimits) *httpx.APIError {
	fields := []struct {
		target *int
		patch  *int
	}{
		{&target.MaxGroupMembers, patch.MaxGroupMembers},
		{&target.MaxMomentTags, patch.MaxMomentTags},
		{&target.MaxMomentImages, patch.MaxMomentImages},
		{&target.MomentDescriptionMaxLength, patch.MomentDescriptionMaxLength},
		{&target.CommentMaxLength, patch.CommentMaxLength},
		{&target.SearchRadiusMax, patch.SearchRadiusMax},
		{&target.MessageRecallWindowSeconds, patch.MessageRecallWindowSeconds},
		{&target.StrangerMessageDailyQuota, patch.StrangerMessageDailyQuota},
		{&target.GroupInviteTTLHours, patch.GroupInviteTTLHours},
		{&target.GroupApplicationTTLHours, patch.GroupApplicationTTLHours},
	}
	for _, f := range fields {
		if apiErr := applyPositiveIntPatch(f.target, f.patch); apiErr != nil {
			return apiErr
		}
	}
	if patch.ReactionEmojis != nil {
		target.ReactionEmojis = patch.ReactionEmojis
	}
	return nil
}

func applyContactPatch(target *ContactInfo, patch PartialContactInfo) *httpx.APIError {
	if patch.SupportEmail != nil {
		if !strings.Contains(strings.TrimSpace(*patch.SupportEmail), "@") {
			return httpx.InvalidRequest("参数错误")
		}
		target.SupportEmail = strings.TrimSpace(*patch.SupportEmail)
	}
	if patch.PrivacyPolicyURL != nil {
		target.PrivacyPolicyURL = strings.TrimSpace(*patch.PrivacyPolicyURL)
	}
	if patch.TermsOfServiceURL != nil {
		target.TermsOfServiceURL = strings.TrimSpace(*patch.TermsOfServiceURL)
	}
	if patch.OfficialWebsite != nil {
		target.OfficialWebsite = strings.TrimSpace(*patch.OfficialWebsite)
	}
	return nil
}

func validateAuditFilter(req AuditLogListRequest) (AuditLogFilter, *httpx.APIError) {
	filter := AuditLogFilter{Action: req.Action, OperatorID: strings.TrimSpace(req.OperatorID), TargetType: req.TargetType, TargetID: strings.TrimSpace(req.TargetID)}
	if req.Pagination != nil {
		filter.Pagination = *req.Pagination
	}
	if strings.TrimSpace(req.StartTime) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.StartTime))
		if err != nil {
			return AuditLogFilter{}, httpx.InvalidRequest("参数错误")
		}
		filter.StartTime = &parsed
	}
	if strings.TrimSpace(req.EndTime) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.EndTime))
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

// AppendGroupDismissAudit records a group-dismiss audit entry (spec
// api_spec_group.md: the dismiss reason is audit-only and never surfaced to
// members). Implemented here so the groups module can write dismiss audits
// through the injected groups.GroupDismissAuditWriter without a groups→admin
// import cycle.
func (s *Service) AppendGroupDismissAudit(ctx context.Context, operatorID string, groupID string, groupName string, reason string) error {
	details := AuditDetails{"type": string(AuditActionGroupDismiss), "group_id": groupID}
	if reason != "" {
		details["reason"] = reason
	}
	return s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, AdminActor{UserID: operatorID}, AuditLogEntry{
		Action:     AuditActionGroupDismiss,
		TargetType: AuditTargetTypeGroup,
		TargetID:   groupID,
		TargetName: groupName,
		Details:    details,
	}))
}
