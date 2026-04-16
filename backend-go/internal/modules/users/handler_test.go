package users

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/notifications"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type testRealtimeSink struct{ events []realtime.Event }

func (s *testRealtimeSink) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *testRealtimeSink) Close() error                { return nil }

type userErrorRepo struct {
	getProfileErr  error
	putProfileErr  error
	getSettingsErr error
	putSettingsErr error
	searchErr      error
	listBlockedErr error
	relErr         error
	followErr      error
	unfollowErr    error
	blockErr       error
	unblockErr     error
	remarkErr      error
	deleteErr      error
}

type invalidCursorRepo struct{}

func (invalidCursorRepo) GetProfile(context.Context, string) (MyProfile, bool, error) {
	return MyProfile{}, false, nil
}
func (invalidCursorRepo) PutProfile(context.Context, MyProfile) error { return nil }
func (invalidCursorRepo) DeleteProfile(context.Context, string) error { return nil }
func (invalidCursorRepo) GetSettings(context.Context, string) (UserSettings, bool, error) {
	return UserSettings{}, false, nil
}
func (invalidCursorRepo) PutSettings(context.Context, string, UserSettings) error { return nil }
func (invalidCursorRepo) SearchProfiles(context.Context, SearchUsersRequest) (PaginatedProfiles, error) {
	return PaginatedProfiles{InvalidCursor: true}, nil
}
func (invalidCursorRepo) Follow(context.Context, string, string, FollowSource, string) error {
	return nil
}
func (invalidCursorRepo) Unfollow(context.Context, string, string) error          { return nil }
func (invalidCursorRepo) Block(context.Context, string, string, string) error     { return nil }
func (invalidCursorRepo) Unblock(context.Context, string, string) error           { return nil }
func (invalidCursorRepo) SetRemark(context.Context, string, string, string) error { return nil }
func (invalidCursorRepo) ListBlocked(context.Context, string, int, string) (PaginatedBlockedUsers, error) {
	return PaginatedBlockedUsers{}, nil
}
func (invalidCursorRepo) ListLikedMoments(context.Context, string, int, string, string) (momenttypes.PaginatedMoments, error) {
	return momenttypes.PaginatedMoments{}, nil
}
func (invalidCursorRepo) Relationship(context.Context, string, string) (Relationship, error) {
	return Relationship{}, nil
}
func (invalidCursorRepo) ListFriends(context.Context, string, int, string) (PaginatedFriends, error) {
	return PaginatedFriends{}, nil
}
func (invalidCursorRepo) ListFollowing(context.Context, string, int, string) (PaginatedFollowing, error) {
	return PaginatedFollowing{}, nil
}
func (invalidCursorRepo) ListFollowers(context.Context, string, int, string) (PaginatedFollowers, error) {
	return PaginatedFollowers{}, nil
}
func (invalidCursorRepo) ListGroups(context.Context, string, int, string) (grouptypes.PaginatedGroups, error) {
	return grouptypes.PaginatedGroups{}, nil
}
func (invalidCursorRepo) ListMoments(context.Context, string, int, string, bool) (momenttypes.PaginatedMoments, error) {
	return momenttypes.PaginatedMoments{}, nil
}
func (invalidCursorRepo) CommonGroupCount(context.Context, string, string) (int, error) {
	return 0, nil
}
func (invalidCursorRepo) UserExists(context.Context, string) (bool, error) { return false, nil }

func (r *userErrorRepo) GetProfile(context.Context, string) (MyProfile, bool, error) {
	return MyProfile{}, false, r.getProfileErr
}
func (r *userErrorRepo) PutProfile(context.Context, MyProfile) error { return r.putProfileErr }
func (r *userErrorRepo) DeleteProfile(context.Context, string) error { return r.deleteErr }
func (r *userErrorRepo) GetSettings(context.Context, string) (UserSettings, bool, error) {
	return UserSettings{}, false, r.getSettingsErr
}
func (r *userErrorRepo) PutSettings(context.Context, string, UserSettings) error {
	return r.putSettingsErr
}
func (r *userErrorRepo) SearchProfiles(context.Context, SearchUsersRequest) (PaginatedProfiles, error) {
	return PaginatedProfiles{}, r.searchErr
}
func (r *userErrorRepo) Follow(context.Context, string, string, FollowSource, string) error {
	return r.followErr
}
func (r *userErrorRepo) Unfollow(context.Context, string, string) error      { return r.unfollowErr }
func (r *userErrorRepo) Block(context.Context, string, string, string) error { return r.blockErr }
func (r *userErrorRepo) Unblock(context.Context, string, string) error       { return r.unblockErr }
func (r *userErrorRepo) SetRemark(context.Context, string, string, string) error {
	return r.remarkErr
}
func (r *userErrorRepo) ListBlocked(context.Context, string, int, string) (PaginatedBlockedUsers, error) {
	return PaginatedBlockedUsers{}, r.listBlockedErr
}
func (r *userErrorRepo) ListLikedMoments(context.Context, string, int, string, string) (momenttypes.PaginatedMoments, error) {
	return momenttypes.PaginatedMoments{}, r.listBlockedErr
}
func (r *userErrorRepo) Relationship(context.Context, string, string) (Relationship, error) {
	return Relationship{}, r.relErr
}
func (r *userErrorRepo) ListFriends(context.Context, string, int, string) (PaginatedFriends, error) {
	return PaginatedFriends{}, r.listBlockedErr
}
func (r *userErrorRepo) ListFollowing(context.Context, string, int, string) (PaginatedFollowing, error) {
	return PaginatedFollowing{}, r.listBlockedErr
}
func (r *userErrorRepo) ListFollowers(context.Context, string, int, string) (PaginatedFollowers, error) {
	return PaginatedFollowers{}, r.listBlockedErr
}
func (r *userErrorRepo) ListGroups(context.Context, string, int, string) (grouptypes.PaginatedGroups, error) {
	return grouptypes.PaginatedGroups{}, r.listBlockedErr
}
func (r *userErrorRepo) ListMoments(context.Context, string, int, string, bool) (momenttypes.PaginatedMoments, error) {
	return momenttypes.PaginatedMoments{}, r.listBlockedErr
}
func (r *userErrorRepo) CommonGroupCount(context.Context, string, string) (int, error) {
	return 0, r.listBlockedErr
}
func (r *userErrorRepo) UserExists(context.Context, string) (bool, error) {
	return false, r.listBlockedErr
}

func TestRequestEmailChangeRequiresIdempotencyKey(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/me/email/request_change", strings.NewReader(`{"new_email":"alice@example.com","password":"secret"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()

	handler.RequestEmailChange(rr, req)
	assertUserCode(t, rr, 22000)
}

func TestContactChangeEndpointsReturnUnavailableWhenProviderMissing(t *testing.T) {
	handler := NewHandler(NewService())
	tests := []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
		body string
	}{
		{name: "request email", call: handler.RequestEmailChange, body: `{"new_email":"alice@example.com","password":"secret"}`},
		{name: "confirm email", call: handler.ConfirmEmailChange, body: `{"new_email":"alice@example.com","verification_code":"123456"}`},
		{name: "request phone", call: handler.RequestPhoneChange, body: `{"new_phone":"13800000000","password":"secret"}`},
		{name: "confirm phone", call: handler.ConfirmPhoneChange, body: `{"new_phone":"13800000000","verification_code":"123456"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body)).WithContext(ctx)
			req.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
			rr := httptest.NewRecorder()
			tc.call(rr, req)
			assertUserCode(t, rr, 10400)
		})
	}
}

func TestContactChangeEndpointsValidatePayload(t *testing.T) {
	handler := NewHandler(NewService())
	tests := []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
		body string
	}{
		{name: "request email missing password", call: handler.RequestEmailChange, body: `{"new_email":"alice@example.com"}`},
		{name: "confirm email missing code", call: handler.ConfirmEmailChange, body: `{"new_email":"alice@example.com"}`},
		{name: "request phone missing password", call: handler.RequestPhoneChange, body: `{"new_phone":"13800000000"}`},
		{name: "confirm phone missing code", call: handler.ConfirmPhoneChange, body: `{"new_phone":"13800000000"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body)).WithContext(ctx)
			req.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
			rr := httptest.NewRecorder()
			tc.call(rr, req)
			assertUserCode(t, rr, 10000)
		})
	}
}

func TestGetMyProfileReturnsCurrentUser(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/me/profile/get", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetMyProfile(rr, req)

	var resp struct {
		Code int       `json:"code"`
		Data MyProfile `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.UserID != "user_alice" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestUpdateMyProfilePersistsChanges(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/me/profile/update", strings.NewReader(`{"nickname":"Alice","bio":"hello"}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UpdateMyProfile(rr, req)

	getReq := httptest.NewRequest(http.MethodPost, "/me/profile/get", strings.NewReader(`{}`))
	getReq = getReq.WithContext(ctx)
	getRR := httptest.NewRecorder()
	handler.GetMyProfile(getRR, getReq)

	var resp struct {
		Code int       `json:"code"`
		Data MyProfile `json:"data"`
	}
	if err := json.NewDecoder(getRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Nickname != "Alice" || resp.Data.Bio != "hello" {
		t.Fatalf("unexpected profile: %#v", resp.Data)
	}
}

func TestUpdateMyProfileRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	handler := NewHandler(NewService())
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	t.Run("unknown field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/me/profile/update", strings.NewReader(`{"nickname":"Alice","bogus":true}`)).WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.UpdateMyProfile(rr, req)
		assertUserCode(t, rr, 10000)
	})

	t.Run("trailing JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/me/profile/update", strings.NewReader(`{"nickname":"Alice"}{"bio":"x"}`)).WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.UpdateMyProfile(rr, req)
		assertUserCode(t, rr, 10000)
	})
}

func TestUpdateMySettingsRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	handler := NewHandler(NewService())
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	t.Run("unknown field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/me/settings/update", strings.NewReader(`{"settings":{"allow_follow":true},"bogus":1}`)).WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.UpdateMySettings(rr, req)
		assertUserCode(t, rr, 10000)
	})

	t.Run("trailing JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/me/settings/update", strings.NewReader(`{"settings":{"allow_follow":true}}{}`)).WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.UpdateMySettings(rr, req)
		assertUserCode(t, rr, 10000)
	})
}

func TestGetMySettingsReturnsDefaults(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/me/settings/get", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetMySettings(rr, req)

	var resp struct {
		Code int          `json:"code"`
		Data UserSettings `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || !resp.Data.AllowFollow || resp.Data.AllowPrivateChat == "" {
		t.Fatalf("unexpected settings: %#v", resp)
	}
}

func TestFollowBlockRemarkAndOpenTheDoor(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	alice := defaultProfile("user_alice")
	alice.Tags = []string{"coffee", "music"}
	if err := service.repo.PutProfile(ctx, alice); err != nil {
		t.Fatalf("put alice profile: %v", err)
	}
	_ = service.repo.PutProfile(ctx, defaultProfile("user_bob"))
	bob := defaultProfile("user_bob")
	bob.Tags = []string{"music", "travel"}
	if err := service.repo.PutProfile(ctx, bob); err != nil {
		t.Fatalf("put bob profile: %v", err)
	}
	charlie := defaultProfile("user_charlie")
	charlie.Nickname = "Charlie"
	if err := service.repo.PutProfile(ctx, charlie); err != nil {
		t.Fatalf("put charlie profile: %v", err)
	}
	repo := service.repo.(*InMemoryRepository)
	repo.groupsByUser["user_alice"] = []grouptypes.GroupItem{{Group: grouptypes.GroupInfo{GroupID: "group_common", Name: "Common Group"}}}
	repo.groupsByUser["user_bob"] = []grouptypes.GroupItem{{Group: grouptypes.GroupInfo{GroupID: "group_common", Name: "Common Group"}}}
	_ = service.repo.PutSettings(ctx, "user_bob", defaultSettings())
	_ = service.repo.PutSettings(ctx, "user_charlie", defaultSettings())

	followReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_bob"}`))
	followReq = followReq.WithContext(ctx)
	followRR := httptest.NewRecorder()
	handler.Follow(followRR, followReq)

	followCharlieReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_charlie"}`))
	followCharlieReq = followCharlieReq.WithContext(ctx)
	followCharlieRR := httptest.NewRecorder()
	handler.Follow(followCharlieRR, followCharlieReq)

	charlieCtx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_charlie", SessionID: "sess_2"})
	charlieFollowAliceReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_alice"}`)).WithContext(charlieCtx)
	charlieFollowAliceRR := httptest.NewRecorder()
	handler.Follow(charlieFollowAliceRR, charlieFollowAliceReq)
	charlieFollowBobReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_bob"}`)).WithContext(charlieCtx)
	charlieFollowBobRR := httptest.NewRecorder()
	handler.Follow(charlieFollowBobRR, charlieFollowBobReq)

	bobCtx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_3"})
	bobFollowCharlieReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_charlie"}`)).WithContext(bobCtx)
	bobFollowCharlieRR := httptest.NewRecorder()
	handler.Follow(bobFollowCharlieRR, bobFollowCharlieReq)

	remarkReq := httptest.NewRequest(http.MethodPost, "/me/set_remark", strings.NewReader(`{"user_id":"user_bob","remark":"buddy"}`))
	remarkReq = remarkReq.WithContext(ctx)
	remarkRR := httptest.NewRecorder()
	handler.SetRemark(remarkRR, remarkReq)

	blockReq := httptest.NewRequest(http.MethodPost, "/me/block", strings.NewReader(`{"user_id":"user_bob"}`))
	blockReq = blockReq.WithContext(ctx)
	blockRR := httptest.NewRecorder()
	handler.Block(blockRR, blockReq)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"user_id":"user_bob"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code int      `json:"code"`
		Data UserItem `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 || openResp.Data.User.UserID != "user_bob" || openResp.Data.Relation == nil || !openResp.Data.Relation.IsBlocked {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
	if openResp.Data.SocialContext == nil || openResp.Data.SocialContext.MutualFriendsCount != 1 {
		t.Fatalf("unexpected social context: %#v", openResp.Data.SocialContext)
	}
	if len(openResp.Data.SocialContext.MutualFriendsPreview) != 1 || openResp.Data.SocialContext.MutualFriendsPreview[0].UserID != "user_charlie" {
		t.Fatalf("unexpected mutual friends preview: %#v", openResp.Data.SocialContext)
	}
	if openResp.Data.SocialContext.CommonGroupsCount == nil || *openResp.Data.SocialContext.CommonGroupsCount != 1 {
		t.Fatalf("unexpected common groups count: %#v", openResp.Data.SocialContext)
	}
	if len(openResp.Data.SocialContext.CommonTags) != 1 || openResp.Data.SocialContext.CommonTags[0] != "music" {
		t.Fatalf("unexpected common tags: %#v", openResp.Data.SocialContext)
	}

	notificationRepo := notifications.NewInMemoryRepository()
	serviceWithNotifications := NewServiceWithDeps(service.repo, notifications.NewServiceWithRepo(notificationRepo))
	if err := serviceWithNotifications.Follow(ctx, "user_bob", FollowSourceProfile, ""); err != nil {
		t.Fatalf("follow with notifications: %v", err)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_bob", notifications.NotificationListRequest{})
	if err != nil || len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "new_follower" {
		t.Fatalf("unexpected notifications page: %#v err=%v", notificationsPage, err)
	}

	blockedReq := httptest.NewRequest(http.MethodPost, "/me/list_blocked", strings.NewReader(`{}`))
	blockedReq = blockedReq.WithContext(ctx)
	blockedRR := httptest.NewRecorder()
	handler.ListBlocked(blockedRR, blockedReq)
	var blockedResp struct {
		Code int                   `json:"code"`
		Data PaginatedBlockedUsers `json:"data"`
	}
	if err := json.NewDecoder(blockedRR.Body).Decode(&blockedResp); err != nil {
		t.Fatalf("decode blocked response: %v", err)
	}
	if blockedResp.Code != 0 || len(blockedResp.Data.Items) != 1 || blockedResp.Data.Items[0].User.UserID != "user_bob" {
		t.Fatalf("unexpected blocked response: %#v", blockedResp)
	}
}

func TestFollowMutualFollowAddsNotificationsAndRealtime(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	if err := repo.PutProfile(context.Background(), defaultProfile("user_alice")); err != nil {
		t.Fatalf("put alice profile: %v", err)
	}
	if err := repo.PutProfile(context.Background(), defaultProfile("user_bob")); err != nil {
		t.Fatalf("put bob profile: %v", err)
	}
	if err := repo.PutSettings(context.Background(), "user_alice", defaultSettings()); err != nil {
		t.Fatalf("put alice settings: %v", err)
	}
	if err := repo.PutSettings(context.Background(), "user_bob", defaultSettings()); err != nil {
		t.Fatalf("put bob settings: %v", err)
	}
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	ctxBob := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_bob"})
	if err := service.Follow(ctxBob, "user_alice", FollowSourceProfile, ""); err != nil {
		t.Fatalf("seed reverse follow: %v", err)
	}

	notificationRepo := notifications.NewInMemoryRepository()
	serviceWithNotifications := NewServiceWithDeps(repo, notifications.NewServiceWithRepo(notificationRepo))
	aliceSink := &testRealtimeSink{}
	bobSink := &testRealtimeSink{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	if err := serviceWithNotifications.Follow(ctxAlice, "user_bob", FollowSourceProfile, ""); err != nil {
		t.Fatalf("follow with mutual follow: %v", err)
	}
	aliceNotifications, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list alice notifications: %v", err)
	}
	bobNotifications, err := notificationRepo.List(context.Background(), "user_bob", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list bob notifications: %v", err)
	}
	if len(aliceNotifications.Items) != 1 || aliceNotifications.Items[0].Type != "mutual_follow" {
		t.Fatalf("unexpected alice notifications: %#v", aliceNotifications)
	}
	if len(bobNotifications.Items) != 2 || bobNotifications.Items[0].Type != "mutual_follow" || bobNotifications.Items[1].Type != "new_follower" {
		t.Fatalf("unexpected bob notifications: %#v", bobNotifications)
	}
	if len(aliceSink.events) != 1 || aliceSink.events[0].Type != "MutualFollow" {
		t.Fatalf("unexpected alice events: %#v", aliceSink.events)
	}
	if len(bobSink.events) != 2 || bobSink.events[0].Type != "NewFollower" || bobSink.events[1].Type != "MutualFollow" {
		t.Fatalf("unexpected bob events: %#v", bobSink.events)
	}
}

func TestFollowCarriesSourceIntoNotificationAndRealtime(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	if err := repo.PutProfile(context.Background(), defaultProfile("user_alice")); err != nil {
		t.Fatalf("put alice profile: %v", err)
	}
	if err := repo.PutProfile(context.Background(), defaultProfile("user_bob")); err != nil {
		t.Fatalf("put bob profile: %v", err)
	}
	if err := repo.PutSettings(context.Background(), "user_alice", defaultSettings()); err != nil {
		t.Fatalf("put alice settings: %v", err)
	}
	if err := repo.PutSettings(context.Background(), "user_bob", defaultSettings()); err != nil {
		t.Fatalf("put bob settings: %v", err)
	}
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	notificationRepo := notifications.NewInMemoryRepository()
	serviceWithNotifications := NewServiceWithDeps(repo, notifications.NewServiceWithRepo(notificationRepo))
	bobSink := &testRealtimeSink{}
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	if err := serviceWithNotifications.Follow(ctxAlice, "user_bob", FollowSourceMoment, "moment_123"); err != nil {
		t.Fatalf("follow with source: %v", err)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_bob", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "new_follower" {
		t.Fatalf("unexpected notifications page: %#v", notificationsPage)
	}
	notifPayload := notificationsPage.Items[0].Payload
	if notifPayload["source"] != string(FollowSourceMoment) || notifPayload["source_moment_id"] != "moment_123" {
		t.Fatalf("unexpected notification payload: %#v", notifPayload)
	}
	if len(bobSink.events) != 1 || bobSink.events[0].Type != "NewFollower" {
		t.Fatalf("unexpected realtime events: %#v", bobSink.events)
	}
	eventPayload, ok := bobSink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected realtime payload type: %#v", bobSink.events[0].Payload)
	}
	if eventPayload["source"] != string(FollowSourceMoment) || eventPayload["source_moment_id"] != "moment_123" {
		t.Fatalf("unexpected realtime payload: %#v", eventPayload)
	}
}

func TestListLikedMomentsRespectsVisibility(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "viewer", SessionID: "sess_1"})
	if err := repo.PutProfile(context.Background(), defaultProfile("user_bob")); err != nil {
		t.Fatalf("put profile: %v", err)
	}
	if err := repo.PutSettings(context.Background(), "user_bob", UserSettings{LikedMomentsVisible: false}); err != nil {
		t.Fatalf("put settings: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/list_liked_moments", strings.NewReader(`{"user_id":"user_bob"}`)).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ListLikedMoments(rr, req)
	var resp struct {
		Code int                          `json:"code"`
		Data momenttypes.PaginatedMoments `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10200 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSearchCursorIsOpaqueAndTamperProof(t *testing.T) {
	repo := NewInMemoryRepository()
	alice := defaultProfile("user_alice")
	alice.Nickname = "Alice"
	alice.LastOnline = "2026-03-29T11:00:00Z"
	alice.CreatedAt = "2026-03-29T10:00:00Z"
	alice.Location = Location{Point: GeoPoint{Latitude: 35, Longitude: 112}, LocationName: "Test"}
	if err := repo.PutProfile(context.Background(), alice); err != nil {
		t.Fatalf("put alice profile: %v", err)
	}
	if err := repo.PutSettings(context.Background(), alice.UserID, defaultSettings()); err != nil {
		t.Fatalf("put alice settings: %v", err)
	}
	bob := defaultProfile("user_bob")
	bob.Nickname = "Alice Bob"
	bob.LastOnline = "2026-03-29T10:30:00Z"
	bob.CreatedAt = "2026-03-29T09:30:00Z"
	bob.Location = Location{Point: GeoPoint{Latitude: 35.0001, Longitude: 112.0001}, LocationName: "Test"}
	if err := repo.PutProfile(context.Background(), bob); err != nil {
		t.Fatalf("put bob profile: %v", err)
	}
	if err := repo.PutSettings(context.Background(), bob.UserID, defaultSettings()); err != nil {
		t.Fatalf("put bob settings: %v", err)
	}

	result, err := repo.SearchProfiles(context.Background(), SearchUsersRequest{
		Keyword:        "Alice",
		ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112},
		Pagination:     &PaginationParams{Limit: 1},
	})
	if err != nil {
		t.Fatalf("search profiles: %v", err)
	}
	if result.NextCursor == "" {
		t.Fatal("expected next cursor")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(result.NextCursor)
	if err != nil {
		t.Fatalf("decode cursor token: %v", err)
	}
	var plain map[string]any
	if err := json.Unmarshal(decoded, &plain); err == nil {
		t.Fatalf("expected opaque cursor, got plain json payload %#v", plain)
	}

	tampered := result.NextCursor[:len(result.NextCursor)-1] + "A"
	invalid, err := repo.SearchProfiles(context.Background(), SearchUsersRequest{
		Keyword:        "Alice",
		ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112},
		Pagination:     &PaginationParams{Limit: 1, Cursor: tampered},
	})
	if err != nil {
		t.Fatalf("search profiles with tampered cursor: %v", err)
	}
	if !invalid.InvalidCursor {
		t.Fatalf("expected invalid cursor, got %#v", invalid)
	}
}

func TestInMemoryListFollowingCursorAdvancesPage(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	for _, id := range []string{"user_a", "user_b", "user_c"} {
		profile := defaultProfile(id)
		if err := repo.PutProfile(ctx, profile); err != nil {
			t.Fatalf("put profile %s: %v", id, err)
		}
	}
	if err := repo.Follow(ctx, "user_a", "user_b", FollowSourceProfile, ""); err != nil {
		t.Fatalf("follow b: %v", err)
	}
	if err := repo.Follow(ctx, "user_a", "user_c", FollowSourceProfile, ""); err != nil {
		t.Fatalf("follow c: %v", err)
	}
	first, err := repo.ListFollowing(ctx, "user_a", 1, "")
	if err != nil {
		t.Fatalf("list following first page: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v", first)
	}
	second, err := repo.ListFollowing(ctx, "user_a", 1, first.NextCursor)
	if err != nil {
		t.Fatalf("list following second page: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("unexpected second page: %#v", second)
	}
	if first.Items[0].User.UserID == second.Items[0].User.UserID {
		t.Fatalf("cursor did not advance page: first=%#v second=%#v", first, second)
	}
}

func TestOpenTheDoorCountsNonSearchableCommonGroups(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	alice := defaultProfile("user_alice")
	bob := defaultProfile("user_bob")
	if err := service.repo.PutProfile(ctx, alice); err != nil {
		t.Fatalf("put alice profile: %v", err)
	}
	if err := service.repo.PutProfile(ctx, bob); err != nil {
		t.Fatalf("put bob profile: %v", err)
	}
	repo := service.repo.(*InMemoryRepository)
	repo.groupsByUser["user_alice"] = []grouptypes.GroupItem{{Group: grouptypes.GroupInfo{GroupID: "group_private", Name: "Private Group"}, IsSearchable: false}}
	repo.groupsByUser["user_bob"] = []grouptypes.GroupItem{{Group: grouptypes.GroupInfo{GroupID: "group_private", Name: "Private Group"}, IsSearchable: false}}
	_ = service.repo.PutSettings(ctx, "user_bob", defaultSettings())

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"user_id":"user_bob"}`)).WithContext(ctx)
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code int      `json:"code"`
		Data UserItem `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 || openResp.Data.SocialContext == nil || openResp.Data.SocialContext.CommonGroupsCount == nil || *openResp.Data.SocialContext.CommonGroupsCount != 1 {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
}

func TestInMemoryLikedMomentsCursorUsesLikedAt(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.likedMoments["user_alice"] = []likedMomentRecord{
		{LikedAt: "2026-03-29T12:00:00Z", Moment: momenttypes.MomentItem{MomentID: "moment_newer", CreatedAt: "2026-03-20T00:00:00Z", User: momenttypes.UserInfo{UserID: "user_bob"}}},
		{LikedAt: "2026-03-29T11:00:00Z", Moment: momenttypes.MomentItem{MomentID: "moment_older", CreatedAt: "2026-03-21T00:00:00Z", User: momenttypes.UserInfo{UserID: "user_bob"}}},
	}
	first, err := repo.ListLikedMoments(context.Background(), "user_alice", 1, "", "viewer")
	if err != nil {
		t.Fatalf("list liked first page: %v", err)
	}
	if len(first.Items) != 1 || first.Items[0].MomentID != "moment_newer" || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v", first)
	}
	second, err := repo.ListLikedMoments(context.Background(), "user_alice", 1, first.NextCursor, "viewer")
	if err != nil {
		t.Fatalf("list liked second page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].MomentID != "moment_older" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestUsersHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&userErrorRepo{
		getProfileErr:  errors.New("boom"),
		putProfileErr:  errors.New("boom"),
		getSettingsErr: errors.New("boom"),
		putSettingsErr: errors.New("boom"),
		searchErr:      errors.New("boom"),
		listBlockedErr: errors.New("boom"),
		relErr:         errors.New("boom"),
		followErr:      errors.New("boom"),
		blockErr:       errors.New("boom"),
		remarkErr:      errors.New("boom"),
		deleteErr:      errors.New("boom"),
	}))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	getProfileReq := httptest.NewRequest(http.MethodPost, "/me/profile/get", strings.NewReader(`{}`)).WithContext(ctx)
	getProfileRR := httptest.NewRecorder()
	handler.GetMyProfile(getProfileRR, getProfileReq)
	assertUserCode(t, getProfileRR, 10600)

	getSettingsReq := httptest.NewRequest(http.MethodPost, "/me/settings/get", strings.NewReader(`{}`)).WithContext(ctx)
	getSettingsRR := httptest.NewRecorder()
	handler.GetMySettings(getSettingsRR, getSettingsReq)
	assertUserCode(t, getSettingsRR, 10600)

	searchReq := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Alice","reference_point":{"latitude":35,"longitude":112}}`)).WithContext(ctx)
	searchRR := httptest.NewRecorder()
	handler.Search(searchRR, searchReq)
	assertUserCode(t, searchRR, 10600)

	invalidCursorHandler := NewHandler(NewServiceWithRepo(invalidCursorRepo{}))
	invalidCursorReq := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Alice","reference_point":{"latitude":35,"longitude":112},"pagination":{"cursor":"bad"}}`)).WithContext(ctx)
	invalidCursorRR := httptest.NewRecorder()
	invalidCursorHandler.Search(invalidCursorRR, invalidCursorReq)
	assertUserCode(t, invalidCursorRR, 10000)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"user_id":"user_bob"}`)).WithContext(ctx)
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)
	assertUserCode(t, openRR, 10600)

	followReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_bob"}`)).WithContext(ctx)
	followRR := httptest.NewRecorder()
	handler.Follow(followRR, followReq)
	assertUserCode(t, followRR, 10600)

	blockedReq := httptest.NewRequest(http.MethodPost, "/me/list_blocked", strings.NewReader(`{}`)).WithContext(ctx)
	blockedRR := httptest.NewRecorder()
	handler.ListBlocked(blockedRR, blockedReq)
	assertUserCode(t, blockedRR, 10600)

	likedReq := httptest.NewRequest(http.MethodPost, "/list_liked_moments", strings.NewReader(`{"user_id":"user_bob"}`)).WithContext(ctx)
	likedRR := httptest.NewRecorder()
	handler.ListLikedMoments(likedRR, likedReq)
	assertUserCode(t, likedRR, 10600)
}

func assertUserCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}

// --- Upgrade tests ---

func TestHandlerUpgradeRejectsInvalidLoginID(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	tests := []struct {
		name    string
		loginID string
	}{
		{name: "random string", loginID: "abc!@#"},
		{name: "too short phone", loginID: "123"},
		{name: "email without domain dot", loginID: "user@com"},
		{name: "email without local part", loginID: "@example.com"},
		{name: "mixed alphanumeric", loginID: "abc123def"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login_id":"`+tc.loginID+`","password":"pass"}`)).WithContext(ctx)
			handler.Upgrade(rr, req)
			assertUserCode(t, rr, 10000)
		})
	}
}

func TestHandlerUpgradeAcceptsValidEmail(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	alice := defaultProfile("user_alice")
	alice.UserType = UserTypeTemp
	if err := service.repo.PutProfile(context.Background(), alice); err != nil {
		t.Fatalf("put profile: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login_id":"alice@example.com","password":"secret"}`)).WithContext(ctx)
	handler.Upgrade(rr, req)
	assertUserCode(t, rr, 0)
}

func TestHandlerUpgradeAcceptsValidPhone(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	alice := defaultProfile("user_alice")
	alice.UserType = UserTypeTemp
	if err := service.repo.PutProfile(context.Background(), alice); err != nil {
		t.Fatalf("put profile: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"login_id":"13800138000","password":"secret"}`)).WithContext(ctx)
	handler.Upgrade(rr, req)
	assertUserCode(t, rr, 0)
}
