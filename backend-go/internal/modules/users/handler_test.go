package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

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

func (r *userErrorRepo) GetProfile(context.Context, string) (MyProfile, bool, error) {
	return MyProfile{}, false, r.getProfileErr
}
func (r *userErrorRepo) PutProfile(context.Context, MyProfile) error { return r.putProfileErr }
func (r *userErrorRepo) DeleteProfile(context.Context, string) error { return r.deleteErr }
func (r *userErrorRepo) GetSettings(context.Context, string) (UserSettings, bool, error) {
	return UserSettings{}, false, r.getSettingsErr
}
func (r *userErrorRepo) PutSettings(context.Context, string, UserSettings) error { return r.putSettingsErr }
func (r *userErrorRepo) SearchProfiles(context.Context, SearchUsersRequest) ([]MyProfile, error) {
	return nil, r.searchErr
}
func (r *userErrorRepo) Follow(context.Context, string, string) error   { return r.followErr }
func (r *userErrorRepo) Unfollow(context.Context, string, string) error { return r.unfollowErr }
func (r *userErrorRepo) Block(context.Context, string, string) error    { return r.blockErr }
func (r *userErrorRepo) Unblock(context.Context, string, string) error  { return r.unblockErr }
func (r *userErrorRepo) SetRemark(context.Context, string, string, string) error {
	return r.remarkErr
}
func (r *userErrorRepo) ListBlocked(context.Context, string) ([]BlockedUserItem, error) {
	return nil, r.listBlockedErr
}
func (r *userErrorRepo) Relationship(context.Context, string, string) (Relationship, error) {
	return RelationshipNone, r.relErr
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
	_ = service.repo.PutProfile(ctx, defaultProfile("user_bob"))

	followReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_bob"}`))
	followReq = followReq.WithContext(ctx)
	followRR := httptest.NewRecorder()
	handler.Follow(followRR, followReq)

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
	if openResp.Code != 0 || openResp.Data.User.UserID != "user_bob" || openResp.Data.Relation != RelationshipBlocked {
		t.Fatalf("unexpected open response: %#v", openResp)
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
	if blockedResp.Code != 0 || len(blockedResp.Data.Items) != 1 || blockedResp.Data.Items[0].Remark != "buddy" {
		t.Fatalf("unexpected blocked response: %#v", blockedResp)
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

	searchReq := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Alice"}`)).WithContext(ctx)
	searchRR := httptest.NewRecorder()
	handler.Search(searchRR, searchReq)
	assertUserCode(t, searchRR, 10600)

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
}

func assertUserCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct { Code int `json:"code"` }
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
