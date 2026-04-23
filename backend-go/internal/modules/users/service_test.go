package users

import (
	"context"
	"errors"
	"testing"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type stubUserRepo struct {
	profile       MyProfile
	profileFound  bool
	profileErr    error
	putProfileErr error
	deleteErr     error

	settings       UserSettings
	settingsFound  bool
	settingsErr    error
	putSettingsErr error

	searchResult PaginatedProfiles
	searchErr    error

	followErr    error
	unfollowErr  error
	blockErr     error
	unblockErr   error
	setRemarkErr error

	blockedResult PaginatedBlockedUsers
	blockedErr    error

	likedMoments momenttypes.PaginatedMoments
	likedErr     error

	relation    Relationship
	relationErr error

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
func (r *stubUserRepo) PutSettings(_ context.Context, _ string, _ UserSettings) error {
	return r.putSettingsErr
}
func (r *stubUserRepo) SearchProfiles(_ context.Context, _ SearchUsersRequest) (PaginatedProfiles, error) {
	return r.searchResult, r.searchErr
}
func (r *stubUserRepo) Follow(_ context.Context, _ string, _ string, _ FollowSource, _ string) error {
	return r.followErr
}
func (r *stubUserRepo) Unfollow(_ context.Context, _, _ string) error { return r.unfollowErr }
func (r *stubUserRepo) Block(_ context.Context, _, _, _ string) error { return r.blockErr }
func (r *stubUserRepo) Unblock(_ context.Context, _, _ string) error  { return r.unblockErr }
func (r *stubUserRepo) SetRemark(_ context.Context, _, _, _ string) error {
	return r.setRemarkErr
}
func (r *stubUserRepo) ListBlocked(_ context.Context, _ string, _ int, _ string) (PaginatedBlockedUsers, error) {
	return r.blockedResult, r.blockedErr
}
func (r *stubUserRepo) ListLikedMoments(_ context.Context, _ string, _ int, _ string, _ string) (momenttypes.PaginatedMoments, error) {
	return r.likedMoments, r.likedErr
}
func (r *stubUserRepo) Relationship(_ context.Context, _, _ string) (Relationship, error) {
	return r.relation, r.relationErr
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
func (r *stubUserRepo) ListGroups(_ context.Context, _ string, _ int, _ string) (grouptypes.PaginatedGroups, error) {
	return r.groupsResult, r.groupsErr
}
func (r *stubUserRepo) ListMoments(_ context.Context, _ string, _ int, _ string, _ bool) (momenttypes.PaginatedMoments, error) {
	return r.momentsResult, r.momentsErr
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
	svc := NewServiceWithDeps(repo, nil)
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
	_, err := svc.GetMyProfile(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceUpdateMyProfileAppliesPatch(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	nick := "new_nick"
	err := svc.UpdateMyProfile(userCtx("u1"), UpdateProfileRequest{Nickname: &nick})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdateMyProfileRequiresAuth(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	err := svc.UpdateMyProfile(context.Background(), UpdateProfileRequest{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

// --- Settings ---

func TestServiceGetMySettingsCreatesDefault(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	settings, err := svc.GetMySettings(userCtx("u1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.PrivacyLevel != "public" {
		t.Fatalf("expected public privacy, got %s", settings.PrivacyLevel)
	}
}

func TestServiceUpdateMySettingsAppliesPatch(t *testing.T) {
	repo := &stubUserRepo{settingsFound: true}
	svc := NewServiceWithRepo(repo)
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
	err := svc.Follow(userCtx("u1"), "u1", FollowSourceProfile, "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServiceFollowRejectsUnknownTarget(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: false})
	err := svc.Follow(userCtx("u1"), "u2", FollowSourceProfile, "")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceFollowRejectsWhenFollowDisabled(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true, settings: UserSettings{AllowFollow: false}, settingsFound: true})
	err := svc.Follow(userCtx("u1"), "u2", FollowSourceProfile, "")
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceFollowHappyPath(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings()}
	svc := NewServiceWithRepo(repo)
	err := svc.Follow(userCtx("u1"), "u2", FollowSourceProfile, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUnfollowRejectsSelf(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true})
	err := svc.Unfollow(userCtx("u1"), "u1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

// --- Block/Unblock ---

func TestServiceBlockRejectsSelf(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true})
	err := svc.Block(userCtx("u1"), "u1", "")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServiceBlockHappyPath(t *testing.T) {
	repo := &stubUserRepo{exists: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Block(userCtx("u1"), "u2", "spam")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUnblockRejectsSelf(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{exists: true})
	err := svc.Unblock(userCtx("u1"), "u1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

// --- Location ---

func TestServiceUpdateLocationHappyPath(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.UpdateLocation(userCtx("u1"), UpdateLocationRequest{Location: Location{Point: GeoPoint{Latitude: 1, Longitude: 2}, LocationName: "test"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Delete Account ---

func TestServiceDeleteAccountHappyPath(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDeleteAccountRequiresAuth(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	err := svc.DeleteAccount(context.Background(), DeleteAccountRequest{Confirmation: "DELETE_MY_ACCOUNT"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceDeleteAccountRejectsWrongConfirmation(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	err := svc.DeleteAccount(userCtx("u1"), DeleteAccountRequest{Confirmation: "WRONG"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

// --- Upgrade ---

func TestServiceUpgradeRejectsAlreadyNormal(t *testing.T) {
	repo := &stubUserRepo{profile: testProfile("u1"), profileFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "test@test.com", Password: "pass"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict for already normal user, got %v", err)
	}
}

func TestServiceUpgradeWithEmailWritesToEmail(t *testing.T) {
	tempProfile := testProfile("u1")
	tempProfile.UserType = UserTypeTemp
	repo := &stubUserRepo{profile: tempProfile, profileFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "test@test.com", Password: "pass"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.profile.Email != "test@test.com" {
		t.Fatalf("expected email to be set, got %q", repo.profile.Email)
	}
}

func TestServiceUpgradeWithPhoneWritesToPhone(t *testing.T) {
	tempProfile := testProfile("u1")
	tempProfile.UserType = UserTypeTemp
	repo := &stubUserRepo{profile: tempProfile, profileFound: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Upgrade(userCtx("u1"), UpgradeAccountRequest{LoginID: "13800138000", Password: "pass"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.profile.Phone != "13800138000" {
		t.Fatalf("expected phone to be set, got %q", repo.profile.Phone)
	}
}

// --- SetRemark ---

func TestServiceSetRemarkRequiresFollowing(t *testing.T) {
	repo := &stubUserRepo{exists: true, relation: RelationshipNone}
	svc := NewServiceWithRepo(repo)
	err := svc.SetRemark(userCtx("u1"), "u2", "friend")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServiceSetRemarkHappyPath(t *testing.T) {
	repo := &stubUserRepo{exists: true, relation: Relationship{IsFollowing: true}}
	svc := NewServiceWithRepo(repo)
	err := svc.SetRemark(userCtx("u1"), "u2", "friend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Contact change ---

func TestServiceRequestEmailChangeUnavailable(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	_, err := svc.RequestEmailChange(userCtx("u1"), RequestEmailChangeRequest{})
	if !errors.Is(err, ErrContactChangeUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestServiceRequestPhoneChangeUnavailable(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	_, err := svc.RequestPhoneChange(userCtx("u1"), RequestPhoneChangeRequest{})
	if !errors.Is(err, ErrContactChangeUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

// --- ListBlocked ---

func TestServiceListBlockedHappyPath(t *testing.T) {
	repo := &stubUserRepo{blockedResult: PaginatedBlockedUsers{Items: []BlockedUserItem{{User: UserInfo{UserID: "u2"}}}}}
	svc := NewServiceWithRepo(repo)
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
	_, err := svc.Search(userCtx("u1"), SearchUsersRequest{ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor, got %v", err)
	}
}

// --- ListFriends ---

func TestServiceListFriendsRequiresAuth(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	_, err := svc.ListFriends(context.Background(), ListFriendsRequest{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceListFriendsRejectsHidden(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: UserSettings{SocialListsVisible: false}}
	svc := NewServiceWithRepo(repo)
	_, err := svc.ListFriends(userCtx("u1"), ListFriendsRequest{UserID: "u2"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

// --- ListLikedMoments ---

func TestServiceListLikedMomentsRejectsHidden(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: UserSettings{LikedMomentsVisible: false}}
	svc := NewServiceWithRepo(repo)
	_, err := svc.ListLikedMoments(userCtx("u1"), UserLikedMomentsRequest{UserID: "u2"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

// --- ListGroups ---

func TestServiceListGroupsRejectsHidden(t *testing.T) {
	repo := &stubUserRepo{exists: true, settingsFound: true, settings: UserSettings{GroupsVisible: false}}
	svc := NewServiceWithRepo(repo)
	_, err := svc.ListGroups(userCtx("u1"), UserGroupsRequest{UserID: "u2"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}
