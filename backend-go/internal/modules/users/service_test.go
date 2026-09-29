package users

import (
	"context"
	"errors"
	"fmt"
	"testing"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/notify"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type stubUserRepo struct {
	profile      MyProfile
	profileFound bool
	// putSettingsCalls 统计 PutSettings 调用次数（F-10 纯读断言用）。
	putSettingsCalls int
	profileErr       error
	putProfileErr    error
	deleteErr        error

	profiles    map[string]MyProfile
	profilesErr error

	mutualTotal   int
	mutualPreview []UserInfo
	mutualErr     error

	settings       UserSettings
	settingsFound  bool
	settingsErr    error
	putSettingsErr error

	searchResult PaginatedProfiles
	searchErr    error

	ownedCount int

	followErr    error
	unfollowErr  error
	blockErr     error
	unblockErr   error
	setRemarkErr error

	blockedResult PaginatedBlockedUsers
	blockedErr    error

	reactedMoments momenttypes.PaginatedMoments
	reactedErr     error

	relation    Relationship
	relationErr error

	// followNoop 让 stub 的 Follow 返回 changed=false，用于验证「已关注」时
	// 服务层跳过重复通知（审计 F-12）。
	followNoop bool

	friends    PaginatedFriends
	friendsErr error

	following    PaginatedFollowing
	followingErr error

	followers    PaginatedFollowers
	followersErr error

	groupsResult grouptypes.PaginatedGroups
	groupsErr    error

	momentsResult momenttypes.PaginatedMoments
	momentsErr    error

	commonGroups int
	commonErr    error

	exists    bool
	existsErr error
}

func (r *stubUserRepo) GetProfile(_ context.Context, _ string) (MyProfile, bool, error) {
	return r.profile, r.profileFound, r.profileErr
}

func (r *stubUserRepo) GetProfiles(_ context.Context, userIDs []string) (map[string]MyProfile, error) {
	if r.profilesErr != nil {
		return nil, r.profilesErr
	}
	if r.profiles != nil {
		return r.profiles, nil
	}
	result := make(map[string]MyProfile, len(userIDs))
	if r.profileFound {
		for _, id := range userIDs {
			if id == r.profile.UserID {
				result[id] = r.profile
			}
		}
	}
	return result, nil
}

func (r *stubUserRepo) MutualFriends(_ context.Context, _, _ string, _ int) (int, []UserInfo, error) {
	return r.mutualTotal, r.mutualPreview, r.mutualErr
}
func (r *stubUserRepo) PutProfile(_ context.Context, p MyProfile) error {
	if r.putProfileErr != nil {
		return r.putProfileErr
	}
	r.profile = p
	r.profileFound = true
	return nil
}
func (r *stubUserRepo) DeleteProfile(_ context.Context, _ string) error { return r.deleteErr }
func (r *stubUserRepo) GetSettings(_ context.Context, _ string) (UserSettings, bool, error) {
	return r.settings, r.settingsFound, r.settingsErr
}
func (r *stubUserRepo) GetSettingsBatch(_ context.Context, userIDs []string) (map[string]UserSettings, error) {
	result := make(map[string]UserSettings, len(userIDs))
	if r.settingsErr != nil {
		return result, r.settingsErr
	}
	if r.settingsFound {
		for _, id := range userIDs {
			result[id] = r.settings
		}
	}
	return result, nil
}
func (r *stubUserRepo) PutSettings(_ context.Context, _ string, _ UserSettings) error {
	// 计数供 F-10 用例断言「纯读路径不写库」。
	r.putSettingsCalls++
	return r.putSettingsErr
}
func (r *stubUserRepo) SearchProfiles(_ context.Context, _ SearchUsersRequest) (PaginatedProfiles, error) {
	return r.searchResult, r.searchErr
}
func (r *stubUserRepo) Follow(_ context.Context, _ string, _ string, _ FollowSource, _ string) (bool, error) {
	// followNoop 模拟「已关注」的 no-op 路径（审计 F-12：changed=false 时
	// 服务层必须跳过通知与广播）。
	if r.followNoop {
		return false, r.followErr
	}
	return true, r.followErr
}
func (r *stubUserRepo) Unfollow(_ context.Context, _, _ string) error        { return r.unfollowErr }
func (r *stubUserRepo) Block(_ context.Context, _, _, _ string) error        { return r.blockErr }
func (r *stubUserRepo) BlockCascade(_ context.Context, _, _, _ string) error { return r.blockErr }
func (r *stubUserRepo) Unblock(_ context.Context, _, _ string) error         { return r.unblockErr }
func (r *stubUserRepo) SetRemark(_ context.Context, _, _, _ string) error {
	return r.setRemarkErr
}
func (r *stubUserRepo) CheckBlockedTags(_ context.Context, _ []string) ([]string, error) {
	return nil, nil
}
func (r *stubUserRepo) ListBlocked(_ context.Context, _ string, _ int, _ string) (PaginatedBlockedUsers, error) {
	return r.blockedResult, r.blockedErr
}
func (r *stubUserRepo) Relationship(_ context.Context, _, _ string) (Relationship, error) {
	return r.relation, r.relationErr
}
func (r *stubUserRepo) RelationshipBatch(_ context.Context, _ string, targetUserIDs []string) (map[string]Relationship, error) {
	result := make(map[string]Relationship, len(targetUserIDs))
	for _, id := range targetUserIDs {
		result[id] = r.relation
	}
	return result, r.relationErr
}
func (r *stubUserRepo) BlockedEitherWay(_ context.Context, _, _ string) (callerBlocked, targetBlocked bool, err error) {
	return r.relation.IsBlocked, false, r.relationErr
}
func (r *stubUserRepo) ListFriends(_ context.Context, _ string, _ int, _ string) (PaginatedFriends, error) {
	return r.friends, r.friendsErr
}
func (r *stubUserRepo) ListFollowing(_ context.Context, _ string, _ int, _ string) (PaginatedFollowing, error) {
	return r.following, r.followingErr
}
func (r *stubUserRepo) ListFollowers(_ context.Context, _ string, _ int, _ string) (PaginatedFollowers, error) {
	return r.followers, r.followersErr
}
func (r *stubUserRepo) CommonGroupCount(_ context.Context, _, _ string) (int, error) {
	return r.commonGroups, r.commonErr
}
func (r *stubUserRepo) UserExists(_ context.Context, _ string) (bool, error) {
	return r.exists, r.existsErr
}

func userCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func testProfile(userID string) MyProfile {
	return MyProfile{UserID: userID, Nickname: userID, UserType: UserTypeNormal}
}

// --- Profile ---

func TestServiceGetMyProfileCreatesDefault(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	profile, err := svc.GetMyProfile(userCtx("u1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.UserID != "u1" {
		t.Fatalf("expected u1, got %s", profile.UserID)
	}
}

func TestServiceGetMyProfileCreatesDefaultWhenMissing(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithDeps(repo, nil, nil)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "new_user"})
	profile, err := svc.GetMyProfile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.UserID != "new_user" {
		t.Fatalf("expected user_id=new_user, got %s", profile.UserID)
	}
	_, ok, _ := repo.GetProfile(ctx, "new_user")
	if !ok {
		t.Fatal("expected profile to be persisted")
	}
}

func TestServiceGetMyProfileReturnsExisting(t *testing.T) {
	expected := testProfile("u1")
	svc := NewServiceWithRepo(&stubUserRepo{profile: expected, profileFound: true})
	svc.SetContentProviders(&stubUserRepo{profile: expected, profileFound: true}, &stubUserRepo{profile: expected, profileFound: true})
	profile, err := svc.GetMyProfile(userCtx("u1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Nickname != "u1" {
		t.Fatalf("expected u1, got %s", profile.Nickname)
	}
}

func TestServiceGetMyProfileRequiresAuth(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	_, err := svc.GetMyProfile(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceUpdateMyProfileAppliesPatch(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	nick := "new_nick"
	err := svc.UpdateMyProfile(userCtx("u1"), UpdateAccountRequest{Nickname: &nick})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdateMyProfileRequiresAuth(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	err := svc.UpdateMyProfile(context.Background(), UpdateAccountRequest{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

// --- Settings ---

func TestServiceGetMySettingsCreatesDefault(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	settings, err := svc.GetMySettings(userCtx("u1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.PrivacyLevel != "public" {
		t.Fatalf("expected public privacy, got %s", settings.PrivacyLevel)
	}
}

func TestServiceUpdateMySettingsRejectsInvalidChatPerm(t *testing.T) {
	repo := &stubUserRepo{settingsFound: true}

	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)

	bad := ChatPerm("followers")
	err := svc.UpdateMySettings(userCtx("u1"), UpdateSettingsRequest{
		Settings: PartialUserSettings{AllowPrivateChat: &bad},
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}

	good := ChatPermFriends
	err = svc.UpdateMySettings(userCtx("u1"), UpdateSettingsRequest{
		Settings: PartialUserSettings{AllowPrivateChat: &good},
	})
	if err != nil {
		t.Fatalf("unexpected error for valid perm: %v", err)
	}
}

func TestServiceUpdateMySettingsAppliesPatch(t *testing.T) {
	repo := &stubUserRepo{settingsFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	locVisible := false
	err := svc.UpdateMySettings(userCtx("u1"), UpdateSettingsRequest{
		Settings: PartialUserSettings{LocationVisible: &locVisible},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Follow/Unfollow ---

func TestServiceFollowRejectsSelf(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true})
	svc.SetContentProviders(&stubUserRepo{exists: true}, &stubUserRepo{exists: true})
	err := svc.Follow(userCtx("u1"), "u1", FollowSourceProfile, "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

// 注销用户按不存在处理（038 匿名化留行 + is_deleted）：ensureUserExists
// 必须看删除标记，否则 follow/block 等会静默成功并重建已清理的关系边。
func TestServiceFollowRejectsDeletedTarget(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	if err := repo.PutProfile(ctx, MyProfile{UserID: "u1", Nickname: "u1", UserType: UserTypeNormal}); err != nil {
		t.Fatalf("PutProfile u1: %v", err)
	}
	if err := repo.PutProfile(ctx, MyProfile{UserID: "u2", Nickname: "u2", UserType: UserTypeNormal}); err != nil {
		t.Fatalf("PutProfile u2: %v", err)
	}
	if err := repo.DeleteProfile(ctx, "u2"); err != nil {
		t.Fatalf("DeleteProfile u2: %v", err)
	}
	svc := NewServiceWithDeps(repo, nil, nil)
	err := svc.Follow(userCtx("u1"), "u2", FollowSourceProfile, "")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected not found for deleted target, got %v", err)
	}
}

func TestServiceFollowRejectsUnknownTarget(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: false})
	svc.SetContentProviders(&stubUserRepo{exists: false}, &stubUserRepo{exists: false})
	err := svc.Follow(userCtx("u1"), "u2", FollowSourceProfile, "")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceFollowRejectsWhenFollowDisabled(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true, settings: UserSettings{AllowFollow: false}, settingsFound: true})
	svc.SetContentProviders(&stubUserRepo{exists: true, settings: UserSettings{AllowFollow: false}, settingsFound: true}, &stubUserRepo{exists: true, settings: UserSettings{AllowFollow: false}, settingsFound: true})
	err := svc.Follow(userCtx("u1"), "u2", FollowSourceProfile, "")
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceFollowHappyPath(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings()}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	err := svc.Follow(userCtx("u1"), "u2", FollowSourceProfile, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUnfollowRejectsSelf(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true})
	svc.SetContentProviders(&stubUserRepo{exists: true}, &stubUserRepo{exists: true})
	err := svc.Unfollow(userCtx("u1"), "u1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

// --- Block/Unblock ---

func TestServiceBlockRejectsSelf(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true})
	svc.SetContentProviders(&stubUserRepo{exists: true}, &stubUserRepo{exists: true})
	err := svc.Block(userCtx("u1"), "u1", "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServiceBlockHappyPath(t *testing.T) {
	repo := &stubUserRepo{exists: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	err := svc.Block(userCtx("u1"), "u2", "spam")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUnblockRejectsSelf(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true})
	svc.SetContentProviders(&stubUserRepo{exists: true}, &stubUserRepo{exists: true})
	err := svc.Unblock(userCtx("u1"), "u1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

// --- Location ---

func TestServiceUpdateLocationHappyPath(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	err := svc.UpdateLocation(userCtx("u1"), UpdateLocationRequest{Location: Location{Point: GeoPoint{Latitude: 1, Longitude: 2}, LocationName: "test"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Delete Account ---

func TestServiceDeleteAccountHappyPath(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDeleteAccountRequiresAuth(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	err := svc.DeleteAccount(context.Background(), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

// TestServiceDeleteAccountAnonymizesContentAuthor：spec api_spec_user.md
// 「数据保留策略」——账户删除后 users 行保留（内容因不触发级联而保留），
// 昵称/头像等个人信息替换为"已注销用户"占位，is_deleted 标记使已注销用户
// 不再可被搜索/访问。
func TestServiceDeleteAccountAnonymizesContentAuthor(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	_ = repo.PutProfile(context.Background(), MyProfile{UserID: "u1", Nickname: "Alice", AvatarURL: "https://cdn.example/a.png", UserType: UserTypeNormal, Location: Location{LocationName: "南天门"}})

	err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profile, found, err := repo.GetProfile(context.Background(), "u1")
	if err != nil || !found {
		t.Fatalf("profile must be kept (row preserved for content JOIN), found=%v err=%v", found, err)
	}
	if profile.Nickname != "已注销用户" {
		t.Fatalf("expected anonymized nickname, got %q", profile.Nickname)
	}
	if profile.AvatarURL != "" {
		t.Fatalf("expected cleared avatar, got %q", profile.AvatarURL)
	}
	if !profile.IsDeleted {
		t.Fatalf("expected is_deleted flag set")
	}

	// 已注销用户详情按不存在处理（open_the_door httpx.CodeNotFound）
	if _, ok, err := svc.OpenTheDoor(userCtx("u2"), "u1"); err != nil || ok {
		t.Fatalf("expected deleted user to be treated as not-found, ok=%v err=%v", ok, err)
	}
}

func TestServiceDeleteAccountRejectsWrongConfirmation(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "WRONG"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

// 孤儿群守卫：仍担任群主时拒绝注销（对齐退群时「群主需先转让」的既有规则；
// 注销会删掉全部 membership 含群主行，群行残留即永久无主）。
func TestServiceDeleteAccountRejectsWhenOwningGroups(t *testing.T) {
	repo := &stubUserRepo{ownedCount: 2}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"})
	if !errors.Is(err, ErrOwnsGroups) {
		t.Fatalf("expected owns-groups rejection, got %v", err)
	}
}

func TestServiceDeleteAccountProceedsWhenOwningNone(t *testing.T) {
	repo := &stubUserRepo{ownedCount: 0}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	if err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDeleteAccountProceedsWithoutProvider(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	if err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Upgrade ---

// stubCredentialWriter 记录 AttachCredentials 调用（upgrade 凭证写入测试桩）。
type stubCredentialWriter struct {
	calls   []UpgradeAccountRequest
	userIDs []string
	err     error
}

func (w *stubCredentialWriter) AttachCredentials(_ context.Context, userID, loginID, password string) error {
	if w.err != nil {
		return w.err
	}
	w.userIDs = append(w.userIDs, userID)
	w.calls = append(w.calls, UpgradeAccountRequest{LoginID: loginID, Password: password})
	return nil
}

// noopTxRunner 直通事务执行器（内存测试无需真实事务）。
func noopTxRunner(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func TestServiceUpgradeRejectsAlreadyNormal(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	_, err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "test@test.com", Password: "pass"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict for already normal user, got %v", err)
	}
}

// spec api_spec_user.md → upgrade 契约级语义：升级须把 login_id+password 落
// 登录凭证（auth_accounts），且与 user_type 同事务写入。
func TestServiceUpgradeWritesCredentials(t *testing.T) {
	tempProfile := testProfile("u1")
	tempProfile.UserType = UserTypeTemp
	repo := &stubUserRepo{profile: tempProfile, profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	writer := &stubCredentialWriter{}
	svc.SetCredentialWriter(writer, noopTxRunner)
	data, err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "username01", Password: "pass"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("expected exactly one AttachCredentials call, got %d", len(writer.calls))
	}
	if writer.calls[0].LoginID != "username01" || writer.calls[0].Password != "pass" {
		t.Fatalf("unexpected credential payload: %+v", writer.calls[0])
	}
	if writer.userIDs[0] != "u1" {
		t.Fatalf("expected credentials attached to u1, got %q", writer.userIDs[0])
	}
	if repo.profile.UserType != UserTypeNormal {
		t.Fatalf("expected user_type normal, got %q", repo.profile.UserType)
	}
	if data.UserType != UserTypeNormal {
		t.Fatalf("expected response user_type normal, got %q", data.UserType)
	}
}

// 凭证写入依赖未接线时升级必须拒绝（只改 user_type 不落凭证会导致升级用户
// 无法重新登录）。
func TestServiceUpgradeWithoutCredentialWriterFails(t *testing.T) {
	tempProfile := testProfile("u1")
	tempProfile.UserType = UserTypeTemp
	repo := &stubUserRepo{profile: tempProfile, profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	_, err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "username01", Password: "pass"})
	if !errors.Is(err, ErrUpgradeUnavailable) {
		t.Fatalf("expected ErrUpgradeUnavailable, got %v", err)
	}
}

// login_id 已被占用（凭证写入返回 ErrLoginIDTaken）→ 10400，user_type 不变。
func TestServiceUpgradeLoginIDTaken(t *testing.T) {
	tempProfile := testProfile("u1")
	tempProfile.UserType = UserTypeTemp
	repo := &stubUserRepo{profile: tempProfile, profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	writer := &stubCredentialWriter{err: fmt.Errorf("wrap: %w", ErrLoginIDTaken)}
	svc.SetCredentialWriter(writer, noopTxRunner)
	_, err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "taken01", Password: "pass"})
	if !errors.Is(err, ErrLoginIDTaken) {
		t.Fatalf("expected ErrLoginIDTaken, got %v", err)
	}
	if repo.profile.UserType == UserTypeNormal {
		t.Fatalf("expected user_type to stay temp on conflict")
	}
}

// --- update_account 头像归属校验 ---

// stubAvatarResolver 校验行为桩：err != nil 时一律拒绝。
type stubAvatarResolver struct{ err error }

func (r stubAvatarResolver) ResolveAvatarImage(_ context.Context, _, _ string) error {
	return r.err
}

func (r stubAvatarResolver) AvatarPublicPath(fileID string) string {
	return "/public/v1/media/avatars/" + fileID
}

// spec api_spec_user.md update_account 10300：avatar_file_id 不存在/不归属
// 当前用户/非 avatar 类/未完成上传时拒绝更新。
func TestUpdateMyProfileRejectsForeignAvatar(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	svc.SetAvatarMediaResolver(stubAvatarResolver{err: errors.New("not owned")})
	avatar := "file_foreign"
	err := svc.UpdateMyProfile(userCtx("u1"), UpdateAccountRequest{AvatarFileID: &avatar})
	if !errors.Is(err, ErrAvatarNotOwned) {
		t.Fatalf("expected ErrAvatarNotOwned, got %v", err)
	}
	if repo.profile.AvatarURL == "file_foreign" {
		t.Fatal("expected profile avatar to stay unchanged")
	}
}

func TestUpdateMyProfileAcceptsOwnedAvatar(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	svc.SetAvatarMediaResolver(stubAvatarResolver{})
	avatar := "file_owned"
	if err := svc.UpdateMyProfile(userCtx("u1"), UpdateAccountRequest{AvatarFileID: &avatar}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// avatar_url 存稳定代理路径（非裸 file_id）：客户端直接可用，见 media 包
	// service_avatar.go 与 UpdateMyProfile 转换注释。
	if repo.profile.AvatarURL != "/public/v1/media/avatars/file_owned" {
		t.Fatalf("expected proxied avatar URL, got %q", repo.profile.AvatarURL)
	}
}

// 空字符串表示清除头像（spec update_account），不经过归属校验。
func TestUpdateMyProfileClearAvatarSkipsOwnershipCheck(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	svc.SetAvatarMediaResolver(stubAvatarResolver{err: errors.New("must not be called")})
	avatar := ""
	if err := svc.UpdateMyProfile(userCtx("u1"), UpdateAccountRequest{AvatarFileID: &avatar}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.profile.AvatarURL != "" {
		t.Fatalf("expected avatar cleared, got %q", repo.profile.AvatarURL)
	}
}

// --- SetRemark ---

func TestServiceSetRemarkRequiresFollowing(t *testing.T) {
	repo := &stubUserRepo{exists: true, relation: RelationshipNone}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	err := svc.SetRemark(userCtx("u1"), "u2", "friend")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServiceSetRemarkHappyPath(t *testing.T) {
	repo := &stubUserRepo{exists: true, relation: Relationship{IsFollowing: true}}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	err := svc.SetRemark(userCtx("u1"), "u2", "friend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ListFollowing remark visibility (spec api_spec_user.md RelatedUserItem.remark) ---

func TestServiceListFollowingRemarkSelfKeeps(t *testing.T) {
	remark := "老A"
	repo := &stubUserRepo{
		exists: true,
		following: PaginatedFollowing{Items: []RelatedUserItem{
			{User: UserItem{User: UserInfo{UserID: "u2"}}, Remark: &remark},
		}},
	}
	svc := NewServiceWithRepo(repo)
	res, err := svc.ListFollowing(userCtx("u1"), ListFollowingRequest{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].Remark == nil || *res.Items[0].Remark != "老A" {
		t.Fatalf("expected self following list to keep remark, got %+v", res.Items)
	}
}

func TestServiceListFollowingRemarkOthersStripped(t *testing.T) {
	remark := "老A"
	repo := &stubUserRepo{
		exists:        true,
		settingsFound: true,
		settings:      UserSettings{SocialListsVisible: true},
		following: PaginatedFollowing{Items: []RelatedUserItem{
			{User: UserItem{User: UserInfo{UserID: "u2"}}, Remark: &remark},
		}},
	}
	svc := NewServiceWithRepo(repo)
	res, err := svc.ListFollowing(userCtx("u9"), ListFollowingRequest{UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].Remark != nil {
		t.Fatalf("expected others following list to strip remark, got %+v", res.Items)
	}
}

// --- ListBlocked ---

func TestServiceListBlockedHappyPath(t *testing.T) {
	repo := &stubUserRepo{blockedResult: PaginatedBlockedUsers{Items: []BlockedUserItem{{User: UserInfo{UserID: "u2"}}}}}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	result, err := svc.ListBlocked(userCtx("u1"), MyBlockedRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

// --- Search ---

func TestServiceSearchDelegates(t *testing.T) {
	repo := &stubUserRepo{
		searchResult:  PaginatedProfiles{Items: []MyProfile{testProfile("u2")}},
		settingsFound: true,
		settings:      defaultSettings(),
	}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	result, err := svc.Search(userCtx("u1"), SearchUsersRequest{ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceSearchRejectsInvalidCursor(t *testing.T) {
	repo := &stubUserRepo{searchResult: PaginatedProfiles{InvalidCursor: true}}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	_, err := svc.Search(userCtx("u1"), SearchUsersRequest{ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor, got %v", err)
	}
}

// --- ListFriends ---

func TestServiceListFriendsRequiresUserID(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetContentProviders(&stubUserRepo{}, &stubUserRepo{})
	_, err := svc.ListFriends(context.Background(), ListFriendsRequest{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request (user_id required), got %v", err)
	}
}

func TestServiceListFriendsRejectsHidden(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: UserSettings{SocialListsVisible: false}}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	_, err := svc.ListFriends(userCtx("u1"), ListFriendsRequest{UserID: "u2"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

// --- ListReactedMoments ---

func TestServiceListReactedMomentsRejectsHidden(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: UserSettings{ReactedMomentsVisible: false}}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	_, err := svc.ListReactedMoments(userCtx("u1"), UserReactedMomentsRequest{UserID: "u2"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

// --- ListGroups ---

func TestServiceListGroupsRejectsHidden(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: UserSettings{GroupsVisible: false}}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	_, err := svc.ListGroups(userCtx("u1"), UserGroupsRequest{UserID: "u2"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

// stubContentProviders 承载原 stubUserRepo 的跨模块聚合数据（D1 迁移后
// 数据获取走 GroupsByUserProvider / MomentsByUserProvider）。
type stubContentProviders struct {
	repo *stubUserRepo
}

func (p *stubContentProviders) ListGroupsByUser(_ context.Context, _ string, _ int, _ string) (grouptypes.PaginatedGroups, error) {
	return p.repo.groupsResult, p.repo.groupsErr
}
func (p *stubContentProviders) ListMomentsByAuthor(_ context.Context, _ string, _ int, _ string, _ bool) (momenttypes.PaginatedMoments, error) {
	return p.repo.momentsResult, p.repo.momentsErr
}
func (p *stubContentProviders) ListReactedByUser(_ context.Context, _ string, _ string, _ int, _ string) (momenttypes.PaginatedMoments, error) {
	return p.repo.reactedMoments, p.repo.reactedErr
}
func (p *stubContentProviders) CountCommonGroups(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (p *stubContentProviders) OwnedGroupCount(_ context.Context, _ string) (int, error) {
	return p.repo.ownedCount, nil
}

// ListGroupsByUser / ListMomentsByAuthor / ListReactedByUser / CountCommonGroups：
// stubUserRepo 同时实现 GroupsByUserProvider / MomentsByUserProvider（数据复用
// 原聚合字段），D1 迁移后由测试经 SetContentProviders 注入。
func (r *stubUserRepo) ListGroupsByUser(_ context.Context, _ string, _ int, _ string) (grouptypes.PaginatedGroups, error) {
	return r.groupsResult, r.groupsErr
}
func (r *stubUserRepo) ListMomentsByAuthor(_ context.Context, _ string, _ int, _ string, _ bool) (momenttypes.PaginatedMoments, error) {
	return r.momentsResult, r.momentsErr
}
func (r *stubUserRepo) ListReactedByUser(_ context.Context, _ string, _ string, _ int, _ string) (momenttypes.PaginatedMoments, error) {
	return r.reactedMoments, r.reactedErr
}
func (r *stubUserRepo) CountCommonGroups(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (r *stubUserRepo) OwnedGroupCount(_ context.Context, _ string) (int, error) {
	return r.ownedCount, nil
}

func TestServiceBatchGetSkipsMissingAndDeleted(t *testing.T) {
	// BatchGet 一次取多画像：缺席与已注销直接缺席，不携带 relation/stats。
	repo := &stubUserRepo{profiles: map[string]MyProfile{
		"u1": {UserID: "u1", Nickname: "One"},
		"u2": {UserID: "u2", Nickname: "Two", IsDeleted: true},
	}}
	svc := NewServiceWithRepo(repo)
	got, err := svc.BatchGet(userCtx("viewer"), []string{"u1", "u2", "missing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected only u1, got %v", got)
	}
	if got["u1"].Nickname != "One" {
		t.Fatalf("unexpected user: %+v", got["u1"])
	}
}

func TestServiceBatchGetPropagatesBatchError(t *testing.T) {
	repo := &stubUserRepo{profilesErr: errors.New("boom")}
	svc := NewServiceWithRepo(repo)
	if _, err := svc.BatchGet(userCtx("viewer"), []string{"u1"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// 审计 F-12：仓储报告 changed=false（此前已关注）时，Follow 必须是彻底 no-op
// ——不落 new_follower 通知、不广播实时事件。此前 upsert 恒成功，反复
// follow 会每轮重复通知。
func TestServiceFollowSkipsNotificationWhenAlreadyFollowing(t *testing.T) {
	notificationRepo := notify.NewInMemoryRepository()
	newSvc := func(repo *stubUserRepo) *Service {
		svc := NewServiceWithDeps(repo, notify.NewServiceWithRepo(notificationRepo), nil)
		svc.SetContentProviders(repo, repo)
		return svc
	}

	// 对照：首次关注会落一条通知。
	if err := newSvc(&stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings()}).
		Follow(userCtx("u1"), "u2", FollowSourceProfile, ""); err != nil {
		t.Fatalf("first follow: %v", err)
	}
	first, err := notificationRepo.List(context.Background(), "u2", notify.NotifyListRequest{Pagination: &notify.PaginationParams{Limit: 50}})
	if err != nil {
		t.Fatalf("list after first follow: %v", err)
	}
	if len(first.Items) == 0 {
		t.Fatal("expected a new_follower notification after the first follow")
	}

	// 已关注 → 仓储 changed=false：不得再落通知。
	noop := &stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings(), followNoop: true}
	if err := newSvc(noop).Follow(userCtx("u1"), "u2", FollowSourceProfile, ""); err != nil {
		t.Fatalf("repeat follow: %v", err)
	}
	second, err := notificationRepo.List(context.Background(), "u2", notify.NotifyListRequest{Pagination: &notify.PaginationParams{Limit: 50}})
	if err != nil {
		t.Fatalf("list after repeat follow: %v", err)
	}
	if len(second.Items) != len(first.Items) {
		t.Fatalf("repeat follow must not add notifications: before=%d after=%d", len(first.Items), len(second.Items))
	}
}

// 审计 F-10：SettingsFor 是纯读，不得像 GetMySettings 那样在设置行缺失时
// 写入默认行（读路径带写副作用会为被探测用户凭空插入 settings 行）。
func TestServiceSettingsForIsReadOnly(t *testing.T) {
	// 未找到设置：返回 found=false，且不产生任何写入。
	repo := &stubUserRepo{}
	svc := NewServiceWithRepo(repo)
	_, found, err := svc.SettingsFor(userCtx("u1"), "u2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("found must be false when the user never saved settings")
	}
	if repo.putSettingsCalls != 0 {
		t.Fatalf("SettingsFor must not write: putSettingsCalls=%d", repo.putSettingsCalls)
	}

	// 对照组：GetMySettings 在缺行时会补写默认行（既有行为，仅自读端点使用）。
	repo2 := &stubUserRepo{}
	svc2 := NewServiceWithRepo(repo2)
	if _, err := svc2.GetMySettings(userCtx("u1")); err != nil {
		t.Fatalf("get_my_settings: %v", err)
	}
	if repo2.putSettingsCalls == 0 {
		t.Fatal("GetMySettings is expected to backfill defaults (contrast case)")
	}

	// 已有设置：原样返回。
	repo3 := &stubUserRepo{settingsFound: true, settings: UserSettings{AllowPrivateChat: ChatPermFriends}}
	svc3 := NewServiceWithRepo(repo3)
	got, found, err := svc3.SettingsFor(userCtx("u1"), "u2")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if got.AllowPrivateChat != ChatPermFriends {
		t.Fatalf("AllowPrivateChat = %q, want %q", got.AllowPrivateChat, ChatPermFriends)
	}
}

func (r *stubUserRepo) NearbyMomentNotifyTargets(context.Context, float64, float64, int, string) ([]NearbyNotifyTarget, error) {
	return nil, nil
}
