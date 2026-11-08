package admin

import (
	"context"
	"errors"
	"strings"
	"testing"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

// 本文件覆盖应用配置更新链路的补丁辅助函数与错误出口：
// 单段 patch 的校验/赋值、审计过滤器解析、Provider 的防御性分支，
// 以及带权限校验的 UpdateAppConfig 各 section 的失败路径。

func covStrPtr(v string) *string { return &v }
func covBoolPtr(v bool) *bool    { return &v }

func TestApplyUploadConfigPatch(t *testing.T) {
	target := AdminUploadConfig{}
	patch := PartialUploadConfig{
		ImageMaxSize:      intPtr(10),
		VideoMaxSize:      intPtr(20),
		AudioMaxSize:      intPtr(30),
		FileMaxSize:       intPtr(40),
		AvatarMaxSize:     intPtr(50),
		ChatFileMaxSize:   intPtr(60),
		AllowedImageTypes: []string{"png", "jpg", "png", ""},
		AllowedVideoTypes: []string{"mp4", "mov"},
	}
	if apiErr := applyUploadConfigPatch(&target, patch); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if target.ImageMaxSize != 10 || target.VideoMaxSize != 20 || target.AudioMaxSize != 30 ||
		target.FileMaxSize != 40 || target.AvatarMaxSize != 50 || target.ChatFileMaxSize != 60 {
		t.Fatalf("size fields not applied: %#v", target)
	}
	if len(target.AllowedImageTypes) != 2 || target.AllowedImageTypes[0] != "jpg" || target.AllowedImageTypes[1] != "png" {
		t.Fatalf("allowed image types not sorted/unique: %#v", target.AllowedImageTypes)
	}
	if len(target.AllowedVideoTypes) != 2 {
		t.Fatalf("allowed video types not applied: %#v", target.AllowedVideoTypes)
	}

	invalidCases := []struct {
		name  string
		patch PartialUploadConfig
	}{
		{"image", PartialUploadConfig{ImageMaxSize: intPtr(0)}},
		{"video", PartialUploadConfig{VideoMaxSize: intPtr(-1)}},
		{"audio", PartialUploadConfig{AudioMaxSize: intPtr(0)}},
		{"file", PartialUploadConfig{FileMaxSize: intPtr(0)}},
		{"avatar", PartialUploadConfig{AvatarMaxSize: intPtr(0)}},
		{"chat_file", PartialUploadConfig{ChatFileMaxSize: intPtr(0)}},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			target := AdminUploadConfig{}
			if apiErr := applyUploadConfigPatch(&target, tc.patch); apiErr == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestApplyDefaultsPatch(t *testing.T) {
	target := ContentDefaults{}
	point := GeoPoint{Latitude: 31.23, Longitude: 121.47}
	patch := UpdateDefaultsPayload{
		DefaultLocation:    &PartialLocation{Point: &point, LocationName: covStrPtr("  上海  ")},
		DefaultGroupAvatar: covStrPtr(" g_avatar "),
		DefaultUserAvatar:  covStrPtr(" u_avatar "),
		DefaultGroupSettings: &PartialGroupSettings{
			JoinMethod:        covJoinMethodPtr(groupsmodule.GroupJoinMethod("approval")),
			RequiresApproval:  covBoolPtr(true),
			HasPassword:       covBoolPtr(false),
			IsSearchable:      covBoolPtr(true),
			MembersSearchable: covBoolPtr(false),
			MaxMembers:        intPtr(300),
			Location:          &PartialLocation{Point: &point, LocationName: covStrPtr("  北京  ")},
		},
	}
	if apiErr := applyDefaultsPatch(&target, patch); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if target.DefaultLocation.Point != point || target.DefaultLocation.LocationName != "上海" {
		t.Fatalf("default location not applied: %#v", target.DefaultLocation)
	}
	if target.DefaultGroupAvatar != "g_avatar" || target.DefaultUserAvatar != "u_avatar" {
		t.Fatalf("default avatars not trimmed: %#v", target)
	}
	gs := target.DefaultGroupSettings
	if gs.JoinMethod != groupsmodule.GroupJoinMethod("approval") || !gs.RequiresApproval || gs.HasPassword ||
		!gs.IsSearchable || gs.MembersSearchable || gs.MaxMembers != 300 {
		t.Fatalf("group settings not applied: %#v", gs)
	}
	if gs.Location.LocationName != "北京" || gs.Location.Point != point {
		t.Fatalf("group location not applied: %#v", gs.Location)
	}

	invalidCases := []struct {
		name  string
		patch UpdateDefaultsPayload
	}{
		{"empty_location_name", UpdateDefaultsPayload{DefaultLocation: &PartialLocation{LocationName: covStrPtr("   ")}}},
		{"empty_join_method", UpdateDefaultsPayload{DefaultGroupSettings: &PartialGroupSettings{JoinMethod: covJoinMethodPtr(groupsmodule.GroupJoinMethod(" "))}}},
		{"bad_max_members", UpdateDefaultsPayload{DefaultGroupSettings: &PartialGroupSettings{MaxMembers: intPtr(0)}}},
		{"empty_group_location", UpdateDefaultsPayload{DefaultGroupSettings: &PartialGroupSettings{Location: &PartialLocation{LocationName: covStrPtr(" ")}}}},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			target := ContentDefaults{}
			if apiErr := applyDefaultsPatch(&target, tc.patch); apiErr == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestApplyRateLimitsPatch(t *testing.T) {
	target := AdminRateLimitConfig{}
	patch := PartialRateLimitConfig{
		GlobalPerHour:            intPtr(1),
		UserPerHour:              intPtr(2),
		AuthRegisterPer10Minutes: intPtr(3),
		AuthSigninPer10Minutes:   intPtr(4),
		AuthGuestPer10Minutes:    intPtr(5),
		AuthRefreshPer10Minutes:  intPtr(6),
		AuthResetPer10Minutes:    intPtr(7),
		SearchPerMinute:          intPtr(8),
		WritePerMinute:           intPtr(9),
		MomentsCreatePerMinute:   intPtr(10),
		CommentsCreatePerMinute:  intPtr(11),
		ReportsCreatePerHour:     intPtr(12),
		UploadPresignPerHour:     intPtr(13),
		WSSubscribePerSecond:     intPtr(14),
		WSHandshakePerMinute:     intPtr(15),
		AdminReadPerMinute:       intPtr(16),
		AdminWritePerMinute:      intPtr(17),
	}
	if apiErr := applyRateLimitsPatch(&target, patch); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if target.GlobalPerHour != 1 || target.AdminWritePerMinute != 17 {
		t.Fatalf("rate limit fields not applied: %#v", target)
	}
	if apiErr := applyRateLimitsPatch(&target, PartialRateLimitConfig{GlobalPerHour: intPtr(0)}); apiErr == nil {
		t.Fatal("expected validation error for non-positive rate limit")
	}
}

func TestApplyResponseDefaultsPatch(t *testing.T) {
	target := AdminResponseDefaults{}
	patch := PartialResponseDefaults{
		SearchSuggestCount:       intPtr(1),
		HotMomentsCount:          intPtr(2),
		ChatRecentMessagesCount:  intPtr(3),
		GroupAnnouncementsCount:  intPtr(4),
		SystemAnnouncementsCount: intPtr(5),
		CommentTopRepliesCount:   intPtr(6),
		UserProfileMomentsCount:  intPtr(7),
	}
	if apiErr := applyResponseDefaultsPatch(&target, patch); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if target.SearchSuggestCount != 1 || target.UserProfileMomentsCount != 7 {
		t.Fatalf("response defaults not applied: %#v", target)
	}
	if apiErr := applyResponseDefaultsPatch(&target, PartialResponseDefaults{HotMomentsCount: intPtr(-1)}); apiErr == nil {
		t.Fatal("expected validation error for non-positive response default")
	}
}

func TestApplyFeaturesPatch(t *testing.T) {
	target := FeatureSwitches{}
	applyFeaturesPatch(&target, PartialFeatureSwitches{
		EnablePublicRegistration: covBoolPtr(true),
		EnableGuestRegister:      covBoolPtr(true),
		EnableMoments:            covBoolPtr(false),
		EnableNearbyPeople:       covBoolPtr(true),
	})
	if !target.EnablePublicRegistration || !target.EnableGuestRegister || target.EnableMoments || !target.EnableNearbyPeople {
		t.Fatalf("features not applied: %#v", target)
	}
}

func TestApplyLimitsPatch(t *testing.T) {
	target := AppLimits{}
	patch := PartialAppLimits{
		MaxGroupMembers:            intPtr(1),
		MaxMomentTags:              intPtr(2),
		MaxMomentImages:            intPtr(3),
		MaxChatFiles:               intPtr(4),
		MomentDescriptionMaxLength: intPtr(5),
		CommentMaxLength:           intPtr(6),
		SearchRadiusMax:            intPtr(7),
		MessageRecallWindowSeconds: intPtr(8),
		StrangerMessageDailyQuota:  intPtr(9),
		ReactionEmojis:             []string{"👍", "❤️"},
		GroupInviteTTLHours:        intPtr(10),
		GroupApplicationTTLHours:   intPtr(11),
	}
	if apiErr := applyLimitsPatch(&target, patch); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if target.MaxGroupMembers != 1 || target.GroupApplicationTTLHours != 11 || len(target.ReactionEmojis) != 2 {
		t.Fatalf("limits not applied: %#v", target)
	}
	if apiErr := applyLimitsPatch(&target, PartialAppLimits{MaxChatFiles: intPtr(0)}); apiErr == nil {
		t.Fatal("expected validation error for non-positive limit")
	}
}

func TestApplyContactPatch(t *testing.T) {
	target := ContactInfo{}
	patch := PartialContactInfo{
		SupportEmail:      covStrPtr("  support@example.com "),
		PrivacyPolicyURL:  covStrPtr(" https://example.com/privacy "),
		TermsOfServiceURL: covStrPtr(" https://example.com/terms "),
		OfficialWebsite:   covStrPtr(" https://example.com "),
	}
	if apiErr := applyContactPatch(&target, patch); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if target.SupportEmail != "support@example.com" || target.OfficialWebsite != "https://example.com" {
		t.Fatalf("contact not applied: %#v", target)
	}
	if apiErr := applyContactPatch(&target, PartialContactInfo{SupportEmail: covStrPtr("not-an-email")}); apiErr == nil {
		t.Fatal("expected validation error for invalid support email")
	}
}

func TestValidateAuditFilter(t *testing.T) {
	valid, apiErr := validateAuditFilter(AuditLogListRequest{
		StartTime:  "2026-01-01T00:00:00Z",
		EndTime:    "2026-01-02T00:00:00Z",
		OperatorID: " admin ",
		TargetID:   " target ",
	})
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if valid.StartTime == nil || valid.EndTime == nil || valid.OperatorID != "admin" || valid.TargetID != "target" {
		t.Fatalf("unexpected filter: %#v", valid)
	}

	invalidCases := []struct {
		name string
		req  AuditLogListRequest
	}{
		{"bad_start", AuditLogListRequest{StartTime: "yesterday"}},
		{"bad_end", AuditLogListRequest{EndTime: "tomorrow"}},
		{"reversed_range", AuditLogListRequest{StartTime: "2026-01-02T00:00:00Z", EndTime: "2026-01-01T00:00:00Z"}},
		{"bad_cursor", AuditLogListRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, apiErr := validateAuditFilter(tc.req); apiErr == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

// covJoinMethodPtr 返回 groups 群加入方式指针（PartialGroupSettings 字段类型）。
func covJoinMethodPtr(v groupsmodule.GroupJoinMethod) *groupsmodule.GroupJoinMethod { return &v }

func TestServiceUpdateAppConfigSectionValidation(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := adminCtx("admin")

	validRequests := []UpdateAppConfigRequest{
		{Upload: &PartialUploadConfig{ImageMaxSize: intPtr(10), AllowedImageTypes: []string{"png"}}},
		{Defaults: &UpdateDefaultsPayload{DefaultUserAvatar: covStrPtr(" avatar ")}},
		{RateLimits: &PartialRateLimitConfig{GlobalPerHour: intPtr(10)}},
		{ResponseDefaults: &PartialResponseDefaults{SearchSuggestCount: intPtr(5)}},
		{Features: &PartialFeatureSwitches{EnableMoments: covBoolPtr(true)}},
		{Limits: &PartialAppLimits{MaxGroupMembers: intPtr(300)}},
		{Contact: &PartialContactInfo{SupportEmail: covStrPtr("support@example.com")}},
		{Auth: &PartialAuthConfig{LoginLockoutThreshold: intPtr(5)}},
	}
	for i, req := range validRequests {
		if apiErr, err := svc.UpdateAppConfig(ctx, req); apiErr != nil || err != nil {
			t.Fatalf("valid request #%d failed: apiErr=%v err=%v", i, apiErr, err)
		}
	}

	invalidRequests := []UpdateAppConfigRequest{
		{Upload: &PartialUploadConfig{ImageMaxSize: intPtr(0)}},
		{Defaults: &UpdateDefaultsPayload{DefaultLocation: &PartialLocation{LocationName: covStrPtr(" ")}}},
		{RateLimits: &PartialRateLimitConfig{GlobalPerHour: intPtr(0)}},
		{ResponseDefaults: &PartialResponseDefaults{SearchSuggestCount: intPtr(0)}},
		{Limits: &PartialAppLimits{MaxGroupMembers: intPtr(0)}},
		{Contact: &PartialContactInfo{SupportEmail: covStrPtr("nope")}},
		{Auth: &PartialAuthConfig{LoginLockoutMinutes: intPtr(0)}},
	}
	for i, req := range invalidRequests {
		apiErr, err := svc.UpdateAppConfig(ctx, req)
		if err != nil {
			t.Fatalf("invalid request #%d returned raw error: %v", i, err)
		}
		if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
			t.Fatalf("invalid request #%d: apiErr=%v", i, apiErr)
		}
	}
}

func TestServiceUpdateAppConfigRequiresPermission(t *testing.T) {
	svc, _, _, _ := newTestService()
	// user_bob 不是管理员；空 section 也会先被权限校验拦截。
	apiErr, err := svc.UpdateAppConfig(adminCtx("user_bob"), UpdateAppConfigRequest{})
	if err != nil {
		t.Fatalf("unexpected raw error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected forbidden, got %v", apiErr)
	}
}

// covProvider 覆盖 Provider 的两条错误出口。
type covProvider struct {
	getErr    error
	updateErr error
}

func (p covProvider) GetAppConfig(context.Context) (AdminAppConfig, error) {
	if p.getErr != nil {
		return AdminAppConfig{}, p.getErr
	}
	return AdminAppConfig{}, nil
}

func (p covProvider) UpdateAppConfig(context.Context, string, func(AdminAppConfig) (AdminAppConfig, *httpx.APIError)) (AdminAppConfig, *httpx.APIError, error) {
	if p.updateErr != nil {
		return AdminAppConfig{}, nil, p.updateErr
	}
	return AdminAppConfig{}, nil, nil
}

func TestServiceAppConfigProviderErrors(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewService(repo, covProvider{getErr: errCovAdmin}, NewInMemoryModerationStore(), nil)
	ctx := adminCtx("admin")

	if _, apiErr, err := svc.GetAppConfig(ctx); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("GetAppConfig: apiErr=%v err=%v", apiErr, err)
	}
	if apiErr, err := svc.UpdateAppConfig(ctx, UpdateAppConfigRequest{Limits: &PartialAppLimits{MaxGroupMembers: intPtr(10)}}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("UpdateAppConfig get error: apiErr=%v err=%v", apiErr, err)
	}

	svc = NewService(repo, covProvider{updateErr: errCovAdmin}, NewInMemoryModerationStore(), nil)
	if apiErr, err := svc.UpdateAppConfig(ctx, UpdateAppConfigRequest{Limits: &PartialAppLimits{MaxGroupMembers: intPtr(10)}}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("UpdateAppConfig update error: apiErr=%v err=%v", apiErr, err)
	}
}

// covErrMutableSource 的 Update 直接失败，用于覆盖 Provider 的写失败分支。
type covErrMutableSource struct {
	cfg    appconfig.AppConfig
	update error
}

func (s *covErrMutableSource) Get(context.Context) appconfig.AppConfig { return s.cfg }

func (s *covErrMutableSource) Update(_ context.Context, _ func(appconfig.AppConfig) appconfig.AppConfig) (appconfig.AppConfig, error) {
	return appconfig.AppConfig{}, s.update
}

func TestAppConfigProviderDefensiveBranches(t *testing.T) {
	// 首次校验通过、Update 回调内第二次校验失败：applyErr 防御性分支。
	source := appconfig.NewStaticSource(config.Load())
	provider := NewAppConfigProvider(source)
	calls := 0
	_, apiErr, err := provider.UpdateAppConfig(context.Background(), "admin", func(cfg AdminAppConfig) (AdminAppConfig, *httpx.APIError) {
		calls++
		if calls > 1 {
			return cfg, httpx.InvalidRequest("second pass failed")
		}
		return cfg, nil
	})
	if err != nil {
		t.Fatalf("unexpected raw error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected second-pass api error, got %v", apiErr)
	}

	failing := &covErrMutableSource{cfg: appconfig.AppConfig{}, update: errCovAdmin}
	if _, apiErr, err := NewAppConfigProvider(failing).UpdateAppConfig(context.Background(), "admin", func(cfg AdminAppConfig) (AdminAppConfig, *httpx.APIError) {
		return cfg, nil
	}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("expected source update error, got apiErr=%v err=%v", apiErr, err)
	}
}

func TestStrDeref(t *testing.T) {
	if got := strDeref(nil); got != "" {
		t.Fatalf("strDeref(nil) = %q, want empty", got)
	}
	if got := strDeref(covStrPtr("v")); got != "v" {
		t.Fatalf("strDeref = %q, want v", got)
	}
}

func TestDiffStrings(t *testing.T) {
	added := diffStrings([]string{"a", "b", "c"}, []string{"b"})
	if len(added) != 2 || added[0] != "a" || added[1] != "c" {
		t.Fatalf("diffStrings = %#v, want [a c]", added)
	}
	if got := diffStrings(nil, []string{"x"}); len(got) != 0 {
		t.Fatalf("diffStrings(nil, [x]) = %#v, want empty", got)
	}
}

func TestAppendGroupDismissAudit(t *testing.T) {
	_ = types.StrPtr // 保持 types 引用（占位）
	module, store := newTestModuleWithStore()
	ctx := context.Background()
	if err := module.AppendGroupDismissAudit(ctx, "admin", "group_hotspot", "HotSpot", "policy"); err != nil {
		t.Fatalf("AppendGroupDismissAudit: %v", err)
	}
	store.mu.RLock()
	found := false
	for _, entry := range store.auditLogs {
		if entry.Action == AuditActionGroupDismiss && entry.TargetID == "group_hotspot" {
			found = true
			if entry.Details["reason"] != "policy" {
				store.mu.RUnlock()
				t.Fatalf("reason lost: %#v", entry.Details)
			}
		}
	}
	store.mu.RUnlock()
	if !found {
		t.Fatal("group dismiss audit entry not written")
	}

	// 空 reason 不写入 reason 键。
	if err := module.AppendGroupDismissAudit(ctx, "admin", "group_hotspot", "HotSpot", ""); err != nil {
		t.Fatalf("AppendGroupDismissAudit without reason: %v", err)
	}

	// nil module / 未装配 handler 时安全 no-op。
	var nilModule *Module
	if err := nilModule.AppendGroupDismissAudit(ctx, "admin", "g", "n", "r"); err != nil {
		t.Fatalf("nil module should no-op, got %v", err)
	}
	if err := (&Module{}).AppendGroupDismissAudit(ctx, "admin", "g", "n", "r"); err != nil {
		t.Fatalf("module without handler should no-op, got %v", err)
	}
}

func TestSortAndUniqueStringsAndPermissionSet(t *testing.T) {
	if got := sortAndUniqueStrings(nil); got != nil {
		t.Fatalf("sortAndUniqueStrings(nil) = %#v, want nil", got)
	}
	if got := sortAndUniqueStrings([]string{"b", "", "a", "b"}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("sortAndUniqueStrings = %#v", got)
	}
	if !validPermissionSet(nil) {
		t.Fatal("empty permission set should be valid")
	}
	if validPermissionSet([]string{"not.a.real.permission"}) {
		t.Fatal("unknown permission key should be invalid")
	}
	if !validPermissionSet([]string{wildcardPermission}) {
		t.Fatal("wildcard permission should be valid")
	}
}

var _ = strings.TrimSpace
