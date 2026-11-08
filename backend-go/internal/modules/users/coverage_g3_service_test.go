package users

import (
	"context"
	"errors"
	"strings"
	"testing"

	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/hotcache"
)

func g3StrPtr(v string) *string { return &v }

func g3Repeat(s string, n int) string { return strings.Repeat(s, n) }

func g3GenderPtr(v string) *Gender {
	g := Gender(v)
	return &g
}

func g3BoolPtr(v bool) *bool { return &v }

func g3PresencePtr(v UserPresenceStatus) *UserPresenceStatus { return &v }

func momentPaginated(ids ...string) momenttypes.PaginatedMoments {
	items := make([]momenttypes.MomentItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, momenttypes.MomentItem{MomentID: id})
	}
	return momenttypes.PaginatedMoments{Items: items}
}

// g3AlternatingProfileRepo 让多次 GetProfile 依次返回不同画像（OpenTheDoor
// 先取目标画像、buildSocialContext 再取查看者画像）。
type g3AlternatingProfileRepo struct {
	*stubUserRepo
	first  MyProfile
	second MyProfile
	calls  int
}

func (r *g3AlternatingProfileRepo) GetProfile(ctx context.Context, userID string) (MyProfile, bool, error) {
	r.calls++
	if r.calls == 1 {
		return r.first, true, nil
	}
	return r.second, true, nil
}

// ── 可注入覆盖桩：复用 stubUserRepo 的默认实现，只替换需要报错的单方法 ──

type g3UpdateProfileErrRepo struct {
	*stubUserRepo
	err error
}

func (r *g3UpdateProfileErrRepo) UpdateProfile(context.Context, string, ProfilePatch, *[]string) (bool, error) {
	return false, r.err
}

type g3UpdateLocationErrRepo struct {
	*stubUserRepo
	err error
}

func (r *g3UpdateLocationErrRepo) UpdateProfileLocation(context.Context, string, Location) (bool, error) {
	return false, r.err
}

type g3UpdateSettingsErrRepo struct {
	*stubUserRepo
	err error
}

func (r *g3UpdateSettingsErrRepo) UpdateSettings(context.Context, string, PartialUserSettings) (bool, error) {
	return false, r.err
}

type g3BlockedTagsRepo struct {
	*stubUserRepo
	blocked []string
	err     error
}

func (r *g3BlockedTagsRepo) CheckBlockedTags(context.Context, []string) ([]string, error) {
	return r.blocked, r.err
}

// g3PlainRepo 仅实现 Repository 基础方法（不实现 recentLocationRepo 的轻量
// 位置读能力），用于覆盖 Service.RecentLocation 的 GetProfile 回退分支。
type g3PlainRepo struct {
	Repository
}

type g3ProfileSequenceRepo struct {
	*stubUserRepo
	failAfter int
	calls     int
}

func (r *g3ProfileSequenceRepo) GetProfile(ctx context.Context, userID string) (MyProfile, bool, error) {
	r.calls++
	if r.calls > r.failAfter {
		return MyProfile{}, false, errors.New("profile lookup failed")
	}
	return r.stubUserRepo.GetProfile(ctx, userID)
}

type g3SettingsSequenceRepo struct {
	*stubUserRepo
	failAfter int
	calls     int
}

func (r *g3SettingsSequenceRepo) GetSettings(ctx context.Context, userID string) (UserSettings, bool, error) {
	r.calls++
	if r.calls > r.failAfter {
		return UserSettings{}, false, errors.New("settings lookup failed")
	}
	return r.stubUserRepo.GetSettings(ctx, userID)
}

// g3NotifyWriter 是 userNotificationWriter 的可注入桩。
type g3NotifyWriter struct {
	newFollowerErr error
	mutualErr      error
	newFollowerIDs []string
	mutualIDs      []string
}

func (w *g3NotifyWriter) AppendNewFollower(_ context.Context, targetUserID string, _ UserInfo, _ string, _ string) (string, error) {
	if w.newFollowerErr != nil {
		return "", w.newFollowerErr
	}
	w.newFollowerIDs = append(w.newFollowerIDs, targetUserID)
	return "notify_new_follower", nil
}

func (w *g3NotifyWriter) AppendMutualFollow(_ context.Context, targetUserID string, _ UserInfo) (string, error) {
	if w.mutualErr != nil {
		return "", w.mutualErr
	}
	w.mutualIDs = append(w.mutualIDs, targetUserID)
	return "notify_mutual", nil
}

func TestG3ServiceGetMyProfileDefaultProfileError(t *testing.T) {
	boom := errors.New("boom")
	svc := NewServiceWithRepo(&stubUserRepo{putProfileErr: boom})
	if _, err := svc.GetMyProfile(userCtx("u1")); !errors.Is(err, boom) {
		t.Fatalf("expected default profile error, got %v", err)
	}
}

func TestG3ServiceUpdateMyProfileValidation(t *testing.T) {
	ctx := userCtx("u1")
	cases := []struct {
		name string
		req  UpdateAccountRequest
		want error
	}{
		{"empty nickname", UpdateAccountRequest{Nickname: g3StrPtr("")}, ErrInvalidProfileField},
		{"long nickname", UpdateAccountRequest{Nickname: g3StrPtr(g3Repeat("n", 31))}, ErrInvalidProfileField},
		{"long bio", UpdateAccountRequest{Bio: g3StrPtr(g3Repeat("b", 501))}, ErrInvalidProfileField},
		{"too many tags", UpdateAccountRequest{Tags: &[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}}, ErrInvalidProfileField},
		{"empty tag", UpdateAccountRequest{Tags: &[]string{""}}, ErrInvalidProfileField},
		{"long tag", UpdateAccountRequest{Tags: &[]string{g3Repeat("t", 31)}}, ErrInvalidProfileField},
		{"bad gender", UpdateAccountRequest{Gender: g3GenderPtr("alien")}, ErrInvalidProfileField},
	}
	for _, tc := range cases {
		svc := NewServiceWithRepo(&stubUserRepo{profile: testProfile("u1"), profileFound: true})
		if err := svc.UpdateMyProfile(ctx, tc.req); !errors.Is(err, tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestG3ServiceUpdateMyProfileRepoErrors(t *testing.T) {
	boom := errors.New("boom")
	nickname := "nick"
	ctx := userCtx("u1")

	updateErr := NewServiceWithRepo(&g3UpdateProfileErrRepo{stubUserRepo: &stubUserRepo{}, err: boom})
	if err := updateErr.UpdateMyProfile(ctx, UpdateAccountRequest{Nickname: &nickname}); !errors.Is(err, boom) {
		t.Fatalf("expected UpdateProfile error, got %v", err)
	}
	putErr := NewServiceWithRepo(&stubUserRepo{putProfileErr: boom})
	if err := putErr.UpdateMyProfile(ctx, UpdateAccountRequest{Nickname: &nickname}); !errors.Is(err, boom) {
		t.Fatalf("expected default profile error, got %v", err)
	}
	tagErrSvc := NewServiceWithRepo(&g3BlockedTagsRepo{stubUserRepo: &stubUserRepo{}, err: boom})
	if err := tagErrSvc.UpdateMyProfile(ctx, UpdateAccountRequest{Tags: &[]string{"x"}}); !errors.Is(err, boom) {
		t.Fatalf("expected blocked tags error, got %v", err)
	}
	blockedSvc := NewServiceWithRepo(&g3BlockedTagsRepo{stubUserRepo: &stubUserRepo{}, blocked: []string{"bad"}})
	if err := blockedSvc.UpdateMyProfile(ctx, UpdateAccountRequest{Tags: &[]string{"bad"}}); !errors.Is(err, ErrBlockedTag) {
		t.Fatalf("expected ErrBlockedTag, got %v", err)
	}
}

func TestG3ServiceUpdateMyProfileSuccessPaths(t *testing.T) {
	ctx := userCtx("u1")
	// avatar 值为裸 file_id 且未接线 media resolver 时直存。
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	avatar := "file_raw"
	if err := svc.UpdateMyProfile(ctx, UpdateAccountRequest{AvatarFileID: &avatar}); err != nil {
		t.Fatalf("avatar update: %v", err)
	}
	if repo.profile.AvatarURL != "file_raw" {
		t.Fatalf("avatar url = %q", repo.profile.AvatarURL)
	}
	// 空请求为 no-op。
	if err := svc.UpdateMyProfile(ctx, UpdateAccountRequest{}); err != nil {
		t.Fatalf("empty patch: %v", err)
	}
	// 合法 gender/tags/bio 正常落库。
	gender := GenderMale
	if err := svc.UpdateMyProfile(ctx, UpdateAccountRequest{Gender: &gender, Bio: g3StrPtr("hi"), Tags: &[]string{"tag1"}}); err != nil {
		t.Fatalf("full patch: %v", err)
	}
	if repo.profile.Gender != GenderMale || repo.profile.Bio != "hi" {
		t.Fatalf("profile after patch = %#v", repo.profile)
	}
}

func TestG3ServiceSearchBranches(t *testing.T) {
	boom := errors.New("boom")
	// settings 批量读取失败。
	settingsErrSvc := NewServiceWithRepo(&stubUserRepo{
		searchResult: PaginatedProfiles{Items: []MyProfile{{UserID: "u2"}}},
		settingsErr:  boom,
	})
	if _, err := settingsErrSvc.Search(userCtx("viewer"), SearchUsersRequest{Keyword: "x"}); !errors.Is(err, boom) {
		t.Fatalf("expected settings batch error, got %v", err)
	}

	// 正常分页：跳过本人、补齐关系与游标。
	repo := &stubUserRepo{
		searchResult: PaginatedProfiles{
			Items: []MyProfile{
				{UserID: "viewer", Nickname: "Viewer", Location: Location{Point: GeoPoint{Latitude: 34.2, Longitude: 117.1}}},
				{UserID: "u2", Nickname: "Two", Location: Location{Point: GeoPoint{Latitude: 34.21, Longitude: 117.11}}},
			},
			NextCursor: "next_cursor",
		},
		settingsFound: true,
		settings:      defaultSettings(),
		relation:      Relationship{IsFollowing: true},
	}
	svc := NewServiceWithRepo(repo)
	page, err := svc.Search(userCtx("viewer"), SearchUsersRequest{Keyword: "x", ReferencePoint: GeoPoint{Latitude: 34.2, Longitude: 117.1}})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == nil || *page.NextCursor != "next_cursor" {
		t.Fatalf("search page = %#v", page)
	}
	if page.Items[1].Relation == nil || !page.Items[1].Relation.IsFollowing {
		t.Fatalf("expected relation on non-self item, got %#v", page.Items[1].Relation)
	}
	if page.Items[1].Location == nil || page.Items[1].Location.Distance == nil {
		t.Fatalf("expected distance on location-bearing item")
	}

	// 无设置行时按默认设置渲染。
	noSettings := NewServiceWithRepo(&stubUserRepo{
		searchResult: PaginatedProfiles{Items: []MyProfile{{UserID: "u2", Nickname: "Two", PresenceStatus: UserPresenceOnline}}},
	})
	page, err = noSettings.Search(context.Background(), SearchUsersRequest{Keyword: "x"})
	if err != nil {
		t.Fatalf("search without settings: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].PresenceStatus == nil {
		t.Fatalf("default settings presence missing: %#v", page.Items)
	}

	// 关系批量查询失败。
	relErrSvc := NewServiceWithRepo(&stubUserRepo{
		searchResult:  PaginatedProfiles{Items: []MyProfile{{UserID: "u2"}}},
		settingsFound: true,
		settings:      defaultSettings(),
		relationErr:   boom,
	})
	if _, err := relErrSvc.Search(userCtx("viewer"), SearchUsersRequest{Keyword: "x"}); !errors.Is(err, boom) {
		t.Fatalf("expected relation batch error, got %v", err)
	}

	// 仓储快路径已返回 settings 时不再批量查询（GetSettingsBatch 会报错也必须成功）。
	fastPath := NewServiceWithRepo(&stubUserRepo{
		searchResult: PaginatedProfiles{
			Items:    []MyProfile{{UserID: "u2"}},
			Settings: map[string]UserSettings{"u2": defaultSettings()},
		},
		settingsErr: boom,
	})
	if _, err := fastPath.Search(context.Background(), SearchUsersRequest{Keyword: "x"}); err != nil {
		t.Fatalf("fast path must not query settings: %v", err)
	}
}

func TestG3ServiceOpenTheDoorBranches(t *testing.T) {
	boom := errors.New("boom")

	settingsErrSvc := NewServiceWithRepo(&stubUserRepo{profile: testProfile("u2"), profileFound: true, settingsErr: boom})
	if _, _, err := settingsErrSvc.OpenTheDoor(context.Background(), "u2"); !errors.Is(err, boom) {
		t.Fatalf("expected settings error, got %v", err)
	}

	relErrSvc := NewServiceWithRepo(&stubUserRepo{profile: testProfile("u2"), profileFound: true, settingsFound: true, relationErr: boom})
	if _, _, err := relErrSvc.OpenTheDoor(userCtx("viewer"), "u2"); !errors.Is(err, boom) {
		t.Fatalf("expected relationship error, got %v", err)
	}

	socialErrSvc := NewServiceWithRepo(&stubUserRepo{profile: testProfile("u2"), profileFound: true, settingsFound: true, mutualErr: boom})
	if _, _, err := socialErrSvc.OpenTheDoor(userCtx("viewer"), "u2"); !errors.Is(err, boom) {
		t.Fatalf("expected social context error, got %v", err)
	}

	// 本人视角：bio/presence/location/stats 全部返回，且无 social_context。
	selfProfile := MyProfile{
		UserID:         "u2",
		Nickname:       "Two",
		Bio:            "bio",
		Tags:           []string{"tag"},
		Location:       Location{Point: GeoPoint{Latitude: 34.2, Longitude: 117.1}, LocationName: "徐州"},
		PresenceStatus: UserPresenceOnline,
		Stats:          UserStats{MomentsCount: 1},
	}
	selfSvc := NewServiceWithRepo(&stubUserRepo{profile: selfProfile, profileFound: true, settingsFound: true, settings: defaultSettings()})
	item, ok, err := selfSvc.OpenTheDoor(userCtx("u2"), "u2")
	if err != nil || !ok {
		t.Fatalf("self open the door: ok=%v err=%v", ok, err)
	}
	if item.Bio == nil || item.PresenceStatus == nil || item.Location == nil || item.Stats == nil {
		t.Fatalf("self view must carry all fields: %#v", item)
	}
	if item.SocialContext != nil {
		t.Fatalf("self view must not include social context")
	}

	// 社交上下文：共同好友/共同群组/共同标签。
	viewerProfile := MyProfile{UserID: "viewer", Nickname: "Viewer", Tags: []string{"美食"}}
	targetProfile := MyProfile{UserID: "u2", Nickname: "Two", Tags: []string{"美食"}}
	socialRepo := &stubUserRepo{
		profile:       targetProfile,
		profileFound:  true,
		settingsFound: true,
		settings:      defaultSettings(),
		mutualTotal:   2,
		mutualPreview: []UserInfo{{UserID: "u3", Nickname: "Three"}},
		commonGroups:  3,
		profiles:      map[string]MyProfile{"viewer": viewerProfile},
	}
	// GetProfile 同时服务 viewer 与 target：这里用 sequenced 桩让第 1 次返回 target。
	socialRepoSvc := NewServiceWithRepo(&g3AlternatingProfileRepo{
		stubUserRepo: socialRepo,
		first:        targetProfile,
		second:       viewerProfile,
	})
	socialRepoSvc.SetContentProviders(&stubGroupsProvider{commonGroups: map[string]int{"viewer|u2": 3}}, &stubMomentsProvider{})
	item, ok, err = socialRepoSvc.OpenTheDoor(userCtx("viewer"), "u2")
	if err != nil || !ok {
		t.Fatalf("social open the door: ok=%v err=%v", ok, err)
	}
	if item.SocialContext == nil || item.SocialContext.MutualFriendsCount != 2 || item.SocialContext.CommonGroupsCount == nil || *item.SocialContext.CommonGroupsCount != 3 || len(item.SocialContext.CommonTags) != 1 {
		t.Fatalf("social context = %#v", item.SocialContext)
	}
}

func TestG3ServiceUpdateLocationBranches(t *testing.T) {
	boom := errors.New("boom")
	req := UpdateLocationRequest{Location: Location{Point: GeoPoint{Latitude: 34.2, Longitude: 117.1}, LocationName: "徐州"}}

	if err := NewServiceWithRepo(&stubUserRepo{}).UpdateLocation(context.Background(), req); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	repoErrSvc := NewServiceWithRepo(&g3UpdateLocationErrRepo{stubUserRepo: &stubUserRepo{}, err: boom})
	if err := repoErrSvc.UpdateLocation(userCtx("u1"), req); !errors.Is(err, boom) {
		t.Fatalf("expected repo error, got %v", err)
	}
	putErrSvc := NewServiceWithRepo(&stubUserRepo{putProfileErr: boom})
	if err := putErrSvc.UpdateLocation(userCtx("u1"), req); !errors.Is(err, boom) {
		t.Fatalf("expected default profile error, got %v", err)
	}
	// 行不存在时默认建档成功。
	created := &stubUserRepo{}
	if err := NewServiceWithRepo(created).UpdateLocation(userCtx("u1"), req); err != nil {
		t.Fatalf("default profile create: %v", err)
	}
	if !created.profileFound || created.profile.Location.LocationName != "徐州" {
		t.Fatalf("created profile = %#v", created.profile)
	}
}

func TestG3ServiceUpgradeBranches(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewServiceWithRepo(&stubUserRepo{profileErr: boom}).Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "username01", Password: "pass"}); !errors.Is(err, boom) {
		t.Fatalf("expected profile error, got %v", err)
	}
	if _, err := NewServiceWithRepo(&stubUserRepo{}).Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "username01", Password: "pass"}); !errors.Is(err, ErrUpgradeUnavailable) {
		t.Fatalf("expected upgrade unavailable, got %v", err)
	}
	temp := testProfile("u1")
	temp.UserType = UserTypeTemp
	svc := NewServiceWithRepo(&stubUserRepo{profile: temp, profileFound: true})
	svc.SetCredentialWriter(&stubCredentialWriter{err: boom}, noopTxRunner)
	if _, err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "username01", Password: "pass"}); err == nil {
		t.Fatal("expected wrapped credential error")
	}
}

func TestG3ServiceDeleteAccountBranches(t *testing.T) {
	boom := errors.New("boom")
	req := DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"}
	if err := NewServiceWithRepo(&stubUserRepo{}).DeleteAccount(context.Background(), req); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	ownedSvc := NewServiceWithRepo(&stubUserRepo{})
	ownedSvc.SetContentProviders(&stubGroupsProvider{err: boom}, nil)
	ownedSvc.SetAccountPurger(&stubAccountPurger{})
	if err := ownedSvc.DeleteAccount(userCtx("u1"), req); !errors.Is(err, boom) {
		t.Fatalf("expected owned groups error, got %v", err)
	}
	noPurger := NewServiceWithRepo(&stubUserRepo{})
	if err := noPurger.DeleteAccount(userCtx("u1"), req); err == nil {
		t.Fatal("expected missing purger error")
	}
	purgeOK := NewServiceWithRepo(&stubUserRepo{deleteErr: boom})
	purgeOK.SetAccountPurger(&stubAccountPurger{})
	if err := purgeOK.DeleteAccount(userCtx("u1"), req); !errors.Is(err, boom) {
		t.Fatalf("expected delete profile error, got %v", err)
	}
}

func TestG3ServiceRecentLocationBranches(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	if _, found := svc.RecentLocation(context.Background(), ""); found {
		t.Fatal("empty user must not have location")
	}
	errSvc := NewServiceWithRepo(&stubUserRepo{recentLocationErr: errors.New("boom")})
	if _, found := errSvc.RecentLocation(context.Background(), "u1"); found {
		t.Fatal("error must fall back to not found")
	}

	plainErr := g3PlainRepo{Repository: &stubUserRepo{profileErr: errors.New("boom")}}
	if _, found := NewServiceWithRepo(plainErr).RecentLocation(context.Background(), "u1"); found {
		t.Fatal("profile error must fall back to not found")
	}
	plainMissing := g3PlainRepo{Repository: &stubUserRepo{}}
	if _, found := NewServiceWithRepo(plainMissing).RecentLocation(context.Background(), "u1"); found {
		t.Fatal("missing profile must fall back to not found")
	}
	plainZero := g3PlainRepo{Repository: &stubUserRepo{profile: MyProfile{UserID: "u1"}, profileFound: true}}
	if _, found := NewServiceWithRepo(plainZero).RecentLocation(context.Background(), "u1"); found {
		t.Fatal("zero point must not be a location")
	}
	plainOK := g3PlainRepo{Repository: &stubUserRepo{profile: MyProfile{UserID: "u1", Location: Location{Point: GeoPoint{Latitude: 34.2, Longitude: 117.1}}}, profileFound: true}}
	pt, found := NewServiceWithRepo(plainOK).RecentLocation(context.Background(), "u1")
	if !found || pt.Latitude != 34.2 {
		t.Fatalf("expected profile point, got %#v found=%v", pt, found)
	}
}

func TestG3ServiceGetActorInfoBranches(t *testing.T) {
	if _, err := NewServiceWithRepo(&stubUserRepo{profileErr: errors.New("boom")}).GetActorInfo(context.Background(), "u1"); err == nil {
		t.Fatal("expected actor info error")
	}
	info, err := NewServiceWithRepo(&stubUserRepo{}).GetActorInfo(context.Background(), "u1")
	if err != nil || info.UserID != "u1" || info.Nickname != "u1" {
		t.Fatalf("missing actor fallback = %#v err=%v", info, err)
	}
}

func TestG3ServiceDefaultProfileUsesAppConfig(t *testing.T) {
	source := appconfig.NewStaticSource(config.Load())
	svc := NewServiceWithDeps(&stubUserRepo{}, nil, source)
	profile, err := svc.GetMyProfile(userCtx("u_app"))
	if err != nil {
		t.Fatalf("GetMyProfile: %v", err)
	}
	defaults := source.Get(context.Background()).Defaults
	if profile.Location.LocationName != defaults.DefaultLocation.LocationName {
		t.Fatalf("default location name = %q, want %q", profile.Location.LocationName, defaults.DefaultLocation.LocationName)
	}
	ref := svc.resolveReferencePoint(context.Background(), GeoPoint{})
	if ref.Latitude != defaults.DefaultLocation.Point.Latitude || ref.Longitude != defaults.DefaultLocation.Point.Longitude {
		t.Fatalf("reference point = %#v, want appconfig default %#v", ref, defaults.DefaultLocation.Point)
	}
}

func TestG3ServiceGetMySettingsBranches(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewServiceWithRepo(&stubUserRepo{}).GetMySettings(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if _, err := NewServiceWithRepo(&stubUserRepo{settingsErr: boom}).GetMySettings(userCtx("u1")); !errors.Is(err, boom) {
		t.Fatalf("expected settings error, got %v", err)
	}
	if _, err := NewServiceWithRepo(&stubUserRepo{putSettingsErr: boom}).GetMySettings(userCtx("u1")); !errors.Is(err, boom) {
		t.Fatalf("expected put settings error, got %v", err)
	}
	repo := &stubUserRepo{}
	if settings, err := NewServiceWithRepo(repo).GetMySettings(userCtx("u1")); err != nil || !settings.PresenceVisible || repo.putSettingsCalls != 1 {
		t.Fatalf("default settings = %#v err=%v calls=%d", settings, err, repo.putSettingsCalls)
	}
}

func TestG3ServiceChatSettingsForMissing(t *testing.T) {
	perm, found, err := NewServiceWithRepo(&stubUserRepo{}).ChatSettingsFor(context.Background(), "u1")
	if err != nil || found || perm != "" {
		t.Fatalf("missing settings = %q %v %v", perm, found, err)
	}
}

func TestG3ServiceUpdateMySettingsBranches(t *testing.T) {
	boom := errors.New("boom")
	if err := NewServiceWithRepo(&stubUserRepo{}).UpdateMySettings(context.Background(), UpdateSettingsRequest{}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	visibility := true
	req := UpdateSettingsRequest{Settings: PartialUserSettings{PresenceVisible: &visibility}}
	if err := NewServiceWithRepo(&g3UpdateSettingsErrRepo{stubUserRepo: &stubUserRepo{}, err: boom}).UpdateMySettings(userCtx("u1"), req); !errors.Is(err, boom) {
		t.Fatalf("expected update settings error, got %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{putSettingsErr: boom}).UpdateMySettings(userCtx("u1"), req); !errors.Is(err, boom) {
		t.Fatalf("expected put settings error, got %v", err)
	}
	cache := hotcache.New(nil)
	defer cache.Close()
	repo := &stubUserRepo{settingsFound: true, settings: defaultSettings()}
	svc := NewServiceWithRepo(repo)
	svc.SetSettingsCache(cache)
	if err := svc.UpdateMySettings(userCtx("u1"), req); err != nil {
		t.Fatalf("settings update with cache: %v", err)
	}
}

func TestG3ServiceUpdateMySettingsAppliesFullPatch(t *testing.T) {
	privacy := "private"
	allowPrivateChat := ChatPermNobody
	values := []*bool{g3BoolPtr(true), g3BoolPtr(false)}
	req := UpdateSettingsRequest{Settings: PartialUserSettings{
		PrivacyLevel:          &privacy,
		LocationVisible:       values[0],
		PresenceVisible:       values[1],
		SocialListsVisible:    values[0],
		GroupsVisible:         values[1],
		ReactedMomentsVisible: values[0],
		StatsVisible:          values[1],
		AllowFollow:           values[0],
		AllowPrivateChat:      &allowPrivateChat,
		Searchable:            values[0],
		PhoneSearchable:       values[0],
		NicknameSearchable:    values[1],
		LocationSearchable:    values[0],
		NearbyMomentNotify:    values[0],
	}}
	repo := &stubUserRepo{}
	svc := NewServiceWithRepo(repo)
	// 内存仓储的默认设置行缺失 → 走 applySettingsPatch 全字段分支后建默认行。
	if err := svc.UpdateMySettings(userCtx("u1"), req); err != nil {
		t.Fatalf("full settings patch: %v", err)
	}
	settings := defaultSettings()
	applySettingsPatch(&settings, req.Settings)
	if settings.PrivacyLevel != "private" || settings.AllowPrivateChat != ChatPermNobody || !settings.NearbyMomentNotify {
		t.Fatalf("applied settings = %#v", settings)
	}
}

func TestG3ServiceListReactedMomentsBranches(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewServiceWithRepo(&stubUserRepo{}).ListReactedMoments(context.Background(), UserReactedMomentsRequest{}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
	if _, err := NewServiceWithRepo(&stubUserRepo{existsErr: boom}).ListReactedMoments(context.Background(), UserReactedMomentsRequest{UserID: "u2"}); !errors.Is(err, boom) {
		t.Fatalf("expected user exists error, got %v", err)
	}
	if result, err := NewServiceWithRepo(&stubUserRepo{exists: true}).ListReactedMoments(context.Background(), UserReactedMomentsRequest{UserID: "u2"}); err != nil || len(result.Items) != 0 {
		t.Fatalf("unwired provider = %#v err=%v", result, err)
	}
	repo := &stubUserRepo{exists: true, reactedErr: boom}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(nil, repo)
	if _, err := svc.ListReactedMoments(context.Background(), UserReactedMomentsRequest{UserID: "u2"}); !errors.Is(err, boom) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestG3ServiceListGroupsBranches(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewServiceWithRepo(&stubUserRepo{exists: true, settingsErr: boom}).ListGroups(context.Background(), UserGroupsRequest{UserID: "u2"}); !errors.Is(err, boom) {
		t.Fatalf("expected settings error, got %v", err)
	}
	if result, err := NewServiceWithRepo(&stubUserRepo{exists: true}).ListGroups(context.Background(), UserGroupsRequest{UserID: "u2"}); err != nil || len(result.Items) != 0 {
		t.Fatalf("unwired groups provider = %#v err=%v", result, err)
	}
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings()}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(&stubGroupsProvider{err: boom}, repo)
	if _, err := svc.ListGroups(context.Background(), UserGroupsRequest{UserID: "u2"}); !errors.Is(err, boom) {
		t.Fatalf("expected groups provider error, got %v", err)
	}
}

func TestG3ServiceGetProfileBundleBranches(t *testing.T) {
	boom := errors.New("boom")
	if _, ok, err := NewServiceWithRepo(&stubUserRepo{}).GetProfileBundle(context.Background(), "missing", true); err != nil || ok {
		t.Fatalf("missing bundle ok=%v err=%v", ok, err)
	}
	repo := &stubUserRepo{
		profile:       testProfile("u2"),
		profileFound:  true,
		exists:        true,
		settingsFound: true,
		settings:      defaultSettings(),
		momentsResult: momentPaginated("moment_1"),
	}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	bundle, ok, err := svc.GetProfileBundle(context.Background(), "u2", true)
	if err != nil || !ok || bundle.RecentMoments == nil || len(bundle.RecentMoments.Items) != 1 {
		t.Fatalf("bundle = %#v ok=%v err=%v", bundle, ok, err)
	}
	withConfig := NewServiceWithDeps(repo, nil, appconfig.NewStaticSource(config.Load()))
	withConfig.SetContentProviders(repo, repo)
	if _, ok, err := withConfig.GetProfileBundle(context.Background(), "u2", true); err != nil || !ok {
		t.Fatalf("bundle with appconfig ok=%v err=%v", ok, err)
	}
	repo.momentsErr = boom
	if _, _, err := svc.GetProfileBundle(context.Background(), "u2", true); !errors.Is(err, boom) {
		t.Fatalf("expected moments error, got %v", err)
	}
}

func TestG3ServiceFollowBranches(t *testing.T) {
	boom := errors.New("boom")
	if err := NewServiceWithRepo(&stubUserRepo{}).Follow(context.Background(), "u2", FollowSourceProfile, ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings(), followErr: boom}).Follow(userCtx("u1"), "u2", FollowSourceProfile, ""); !errors.Is(err, boom) {
		t.Fatalf("expected follow error, got %v", err)
	}
	// 通知写入失败不阻断关注。
	writer := &g3NotifyWriter{newFollowerErr: boom}
	svc := NewServiceWithDeps(&stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings()}, writer, nil)
	if err := svc.Follow(userCtx("u1"), "u2", FollowSourceMoment, "moment_1"); err != nil {
		t.Fatalf("follow with notification error: %v", err)
	}
	// 互关关系检查失败。
	relErrSvc := NewServiceWithDeps(&stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings(), relationErr: boom}, nil, nil)
	if err := relErrSvc.Follow(userCtx("u1"), "u2", FollowSourceProfile, ""); !errors.Is(err, boom) {
		t.Fatalf("expected relationship error, got %v", err)
	}
	// 互关通知失败（两次 AppendMutualFollow 都报错）。
	mutualWriter := &g3NotifyWriter{mutualErr: boom}
	mutualSvc := NewServiceWithDeps(&stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings(), relation: Relationship{IsFollowing: true}}, mutualWriter, nil)
	if err := mutualSvc.Follow(userCtx("u1"), "u2", FollowSourceProfile, ""); err != nil {
		t.Fatalf("mutual follow with notification errors: %v", err)
	}
	// 关系缓存失效路径。
	cache := hotcache.New(nil)
	defer cache.Close()
	cachedSvc := NewServiceWithDeps(&stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings()}, nil, nil)
	cachedSvc.SetSettingsCache(cache)
	if err := cachedSvc.Follow(userCtx("u1"), "u2", FollowSourceProfile, ""); err != nil {
		t.Fatalf("cached follow: %v", err)
	}
}

func TestG3ServiceUnfollowBlockBranches(t *testing.T) {
	boom := errors.New("boom")
	if err := NewServiceWithRepo(&stubUserRepo{}).Unfollow(context.Background(), "u2"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unfollow unauthorized = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{exists: true, unfollowErr: boom}).Unfollow(userCtx("u1"), "u2"); !errors.Is(err, boom) {
		t.Fatalf("unfollow repo error = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{existsErr: boom}).Unfollow(userCtx("u1"), "u2"); !errors.Is(err, boom) {
		t.Fatalf("unfollow exists error = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{}).Block(context.Background(), "u2", ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("block unauthorized = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{existsErr: boom}).Block(userCtx("u1"), "u2", ""); !errors.Is(err, boom) {
		t.Fatalf("block exists error = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{exists: true, blockErr: boom}).Block(userCtx("u1"), "u2", ""); !errors.Is(err, boom) {
		t.Fatalf("block cascade error = %v", err)
	}
}

func TestG3ServiceChatTargetsContextBranches(t *testing.T) {
	boom := errors.New("boom")
	svc := NewServiceWithRepo(&stubUserRepo{})
	if result, err := svc.ChatTargetsContext(context.Background(), "viewer", nil); err != nil || len(result) != 0 {
		t.Fatalf("empty targets = %#v err=%v", result, err)
	}
	if _, err := NewServiceWithRepo(&stubUserRepo{profilesErr: boom}).ChatTargetsContext(context.Background(), "viewer", []string{"u2"}); !errors.Is(err, boom) {
		t.Fatalf("profiles error = %v", err)
	}
	profiles := map[string]MyProfile{
		"u2":      {UserID: "u2", Nickname: "Two", PresenceStatus: UserPresenceOnline},
		"deleted": {UserID: "deleted", IsDeleted: true},
	}
	if _, err := NewServiceWithRepo(&stubUserRepo{profiles: profiles, settingsErr: boom}).ChatTargetsContext(context.Background(), "viewer", []string{"u2"}); !errors.Is(err, boom) {
		t.Fatalf("settings batch error = %v", err)
	}
	if _, err := NewServiceWithRepo(&stubUserRepo{profiles: profiles, settingsFound: true, settings: defaultSettings(), relationErr: boom}).ChatTargetsContext(context.Background(), "viewer", []string{"u2"}); !errors.Is(err, boom) {
		t.Fatalf("relationship batch error = %v", err)
	}
	result, err := NewServiceWithRepo(&stubUserRepo{
		profiles:      profiles,
		settingsFound: true,
		settings:      defaultSettings(),
		relation:      Relationship{IsFollowing: true},
	}).ChatTargetsContext(context.Background(), "viewer", []string{"u2", "deleted", "missing"})
	if err != nil {
		t.Fatalf("chat targets: %v", err)
	}
	if _, ok := result["deleted"]; ok {
		t.Fatal("deleted user must be omitted")
	}
	item, ok := result["u2"]
	if !ok || item.PresenceStatus == nil || !item.SettingsFound || !item.Relation.IsFollowing {
		t.Fatalf("u2 context = %#v", item)
	}
	// presence_visible=false 时在线状态应隐藏。
	hidden := defaultSettings()
	hidden.PresenceVisible = false
	result, err = NewServiceWithRepo(&stubUserRepo{profiles: profiles, settingsFound: true, settings: hidden}).ChatTargetsContext(context.Background(), "viewer", []string{"u2"})
	if err != nil || result["u2"].PresenceStatus != nil {
		t.Fatalf("hidden presence = %#v err=%v", result["u2"], err)
	}
}

func TestG3ServiceSetRemarkBranches(t *testing.T) {
	boom := errors.New("boom")
	if err := NewServiceWithRepo(&stubUserRepo{}).SetRemark(context.Background(), "u2", "x"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("set remark unauthorized = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{existsErr: boom}).SetRemark(userCtx("u1"), "u2", "x"); !errors.Is(err, boom) {
		t.Fatalf("set remark exists error = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{exists: true, relationErr: boom}).SetRemark(userCtx("u1"), "u2", "x"); !errors.Is(err, boom) {
		t.Fatalf("set remark relationship error = %v", err)
	}
	if err := NewServiceWithRepo(&stubUserRepo{exists: true, relation: Relationship{IsFollowing: true}, setRemarkErr: boom}).SetRemark(userCtx("u1"), "u2", "x"); !errors.Is(err, boom) {
		t.Fatalf("set remark repo error = %v", err)
	}
}

func TestG3ServiceListBlockedUnauthorized(t *testing.T) {
	if _, err := NewServiceWithRepo(&stubUserRepo{}).ListBlocked(context.Background(), MyBlockedRequest{}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestG3ServiceSocialContextBranches(t *testing.T) {
	boom := errors.New("boom")
	target := testProfile("u2")
	viewer := testProfile("viewer")

	profileErrSvc := NewServiceWithRepo(&g3ProfileSequenceRepo{stubUserRepo: &stubUserRepo{profile: target, profileFound: true, settingsFound: true}, failAfter: 1})
	if _, _, err := profileErrSvc.OpenTheDoor(userCtx("viewer"), "u2"); err == nil {
		t.Fatal("expected viewer profile error")
	}

	mutualErrSvc := NewServiceWithRepo(&stubUserRepo{profile: target, profileFound: true, settingsFound: true, mutualErr: boom})
	if _, _, err := mutualErrSvc.OpenTheDoor(userCtx("viewer"), "u2"); !errors.Is(err, boom) {
		t.Fatalf("expected mutual friends error, got %v", err)
	}

	settingsErrSvc := NewServiceWithRepo(&g3SettingsSequenceRepo{stubUserRepo: &stubUserRepo{profile: target, profileFound: true, settingsFound: true}, failAfter: 1})
	if _, _, err := settingsErrSvc.OpenTheDoor(userCtx("viewer"), "u2"); err == nil {
		t.Fatal("expected target settings error")
	}

	// groups provider 未接线：社会上下文因无共同点返回 nil，仅记录 warn。
	noProviderSvc := NewServiceWithRepo(&stubUserRepo{profile: target, profileFound: true, settingsFound: true})
	item, ok, err := noProviderSvc.OpenTheDoor(userCtx("viewer"), "u2")
	if err != nil || !ok || item.SocialContext != nil {
		t.Fatalf("no-provider social context = %#v ok=%v err=%v", item.SocialContext, ok, err)
	}

	// CountCommonGroups 报错。
	groupsErrSvc := NewServiceWithRepo(&stubUserRepo{profile: target, profileFound: true, settingsFound: true, settings: defaultSettings()})
	groupsErrSvc.SetContentProviders(&stubGroupsProvider{err: boom}, nil)
	if _, _, err := groupsErrSvc.OpenTheDoor(userCtx("viewer"), "u2"); !errors.Is(err, boom) {
		t.Fatalf("expected common groups error, got %v", err)
	}

	// 共同标签去重（大小写不敏感、保持左操作数原始大小写）。
	common := intersectTags([]string{"Food", "food", "Music"}, []string{"FOOD", "music", "MUSIC"})
	if len(common) != 2 || common[0] != "Food" {
		t.Fatalf("intersectTags = %#v", common)
	}
	if intersectTags(nil, []string{"x"}) != nil || intersectTags([]string{"x"}, nil) != nil {
		t.Fatal("intersectTags with empty side must be nil")
	}
	_ = viewer
}

func TestG3ServicePresenceVisibilityHelpers(t *testing.T) {
	boom := errors.New("boom")
	// 空列表直接返回。
	applyPresenceVisibility(context.Background(), &stubUserRepo{}, nil)
	// settings 批量读取失败：保持可见。
	item := RelatedUserItem{User: UserItem{User: UserInfo{UserID: "u2"}}}
	applyPresenceVisibility(context.Background(), &stubUserRepo{settingsErr: boom}, []RelatedUserItem{item})
	if item.User.PresenceStatus != nil {
		t.Fatal("presence must stay as-is on settings error")
	}
	// presence_visible=false 时隐藏。
	settings := defaultSettings()
	settings.PresenceVisible = false
	hiddenItems := []RelatedUserItem{{User: UserItem{User: UserInfo{UserID: "u2"}, PresenceStatus: g3PresencePtr(UserPresenceOnline)}}}
	applyPresenceVisibility(context.Background(), &stubUserRepo{settingsFound: true, settings: settings}, hiddenItems)
	if hiddenItems[0].User.PresenceStatus != nil {
		t.Fatal("presence must be hidden when not visible")
	}
	if err := ensureSocialListsVisible(context.Background(), &stubUserRepo{settingsErr: boom}, "u2", false); !errors.Is(err, boom) {
		t.Fatalf("expected settings error, got %v", err)
	}
	if err := ensureSocialListsVisible(context.Background(), &stubUserRepo{}, "u2", true); err != nil {
		t.Fatalf("self must bypass privacy: %v", err)
	}
}

// g3SettingsMissingSequenceRepo 第一次 GetSettings 正常返回，之后返回 not found
// （覆盖 buildSocialContext 的默认设置回退）。
type g3SettingsMissingSequenceRepo struct {
	*stubUserRepo
	calls int
}

func (r *g3SettingsMissingSequenceRepo) GetSettings(ctx context.Context, userID string) (UserSettings, bool, error) {
	r.calls++
	if r.calls > 1 {
		return UserSettings{}, false, nil
	}
	return r.stubUserRepo.GetSettings(ctx, userID)
}

func TestG3ServiceMiscBranches(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewServiceWithRepo(&stubUserRepo{}).Upgrade(context.Background(), UpgradeAccountRequest{}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("upgrade unauthorized = %v", err)
	}
	if isNumericLoginID("") {
		t.Fatal("empty login id must not be numeric")
	}
	if !isNumericLoginID("13800000000") {
		t.Fatal("numeric login id not detected")
	}

	info, err := NewServiceWithRepo(&stubUserRepo{profile: MyProfile{UserID: "u1", Nickname: "Nick", AvatarURL: "avatar"}, profileFound: true}).GetActorInfo(context.Background(), "u1")
	if err != nil || info.Nickname != "Nick" || info.AvatarURL != "avatar" {
		t.Fatalf("actor info = %#v err=%v", info, err)
	}

	// GetMySettings 已有设置行：直接返回，不写库。
	repo := &stubUserRepo{settingsFound: true, settings: defaultSettings()}
	if settings, err := NewServiceWithRepo(repo).GetMySettings(userCtx("u1")); err != nil || !settings.Searchable || repo.putSettingsCalls != 0 {
		t.Fatalf("existing settings = %#v err=%v puts=%d", settings, err, repo.putSettingsCalls)
	}
	perm, found, err := NewServiceWithRepo(&stubUserRepo{settingsFound: true, settings: defaultSettings()}).ChatSettingsFor(context.Background(), "u1")
	if err != nil || !found || perm != string(ChatPermEveryone) {
		t.Fatalf("chat settings = %q %v %v", perm, found, err)
	}

	// Unfollow 成功 + 关系缓存失效。
	cache := hotcache.New(nil)
	defer cache.Close()
	unfollowSvc := NewServiceWithRepo(&stubUserRepo{exists: true})
	unfollowSvc.SetSettingsCache(cache)
	if err := unfollowSvc.Unfollow(userCtx("u1"), "u2"); err != nil {
		t.Fatalf("unfollow success: %v", err)
	}

	// buildSocialContext：第二次 GetSettings 无行 → 默认设置回退。
	target := MyProfile{UserID: "u2", Nickname: "Two"}
	missingSettingsSvc := NewServiceWithRepo(&g3SettingsMissingSequenceRepo{
		stubUserRepo: &stubUserRepo{profile: target, profileFound: true, settingsFound: true, settings: defaultSettings()},
	})
	item, ok, err := missingSettingsSvc.OpenTheDoor(userCtx("viewer"), "u2")
	if err != nil || !ok || item.SocialContext != nil {
		t.Fatalf("missing settings social context = %#v ok=%v err=%v", item.SocialContext, ok, err)
	}

	// buildSocialContext：groups provider 未接线时仅 warn 并返回 nil 上下文。
	warnSvc := NewServiceWithRepo(&stubUserRepo{profile: target, profileFound: true, settingsFound: true, settings: defaultSettings()})
	if _, ok, err := warnSvc.OpenTheDoor(userCtx("viewer"), "u2"); err != nil || !ok {
		t.Fatalf("no-provider warn path ok=%v err=%v", ok, err)
	}

	// toAccountData：gender 非空时下发指针。
	genderProfile := testProfile("u_gender")
	genderProfile.Gender = GenderFemale
	account, err := NewServiceWithRepo(&stubUserRepo{profile: genderProfile, profileFound: true}).GetMyProfile(userCtx("u_gender"))
	if err != nil || account.Gender == nil || *account.Gender != GenderFemale {
		t.Fatalf("account gender = %#v err=%v", account.Gender, err)
	}

	// profile writer：未接线 PG 后端时的 ErrNotImplemented 分支。
	offSvc := NewServiceWithRepo(NewPostgresRepository(nil))
	if err := offSvc.ProvisionUser(context.Background(), ProvisionUserInput{UserID: "u1"}); err == nil {
		t.Fatal("expected provision user unavailable error")
	}
	if err := offSvc.RecordLogin(context.Background(), "u1", "ip", "loc"); err == nil {
		t.Fatal("expected record login unavailable error")
	}
	_ = boom
}
