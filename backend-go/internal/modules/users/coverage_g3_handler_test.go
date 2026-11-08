package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

func g3CallHandler(t *testing.T, fn http.HandlerFunc, ctx context.Context, body string) (int, int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/g3", strings.NewReader(body))
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	fn(rec, req)
	var payload struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return rec.Code, payload.Code
}

func TestG3HandlerDecodeErrorBranches(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{}))
	ctx := userCtx("g3_user")
	if _, code := g3CallHandler(t, handler.GetMyProfile, ctx, "[1]"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("GetMyProfile non-object body code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.GetMySettings, ctx, "[1]"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("GetMySettings non-object body code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.UpdateMyProfile, ctx, `{"unknown_field":true}`); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("UpdateMyProfile unknown field code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.UpdateMySettings, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("UpdateMySettings malformed body code = %d", code)
	}
}

func TestG3HandlerServiceErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	ctx := userCtx("g3_user")

	profileErr := NewHandler(NewServiceWithRepo(&stubUserRepo{profileErr: boom}))
	if _, code := g3CallHandler(t, profileErr.GetMyProfile, ctx, "{}"); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("GetMyProfile repo error code = %d", code)
	}
	settingsErr := NewHandler(NewServiceWithRepo(&stubUserRepo{settingsErr: boom}))
	if _, code := g3CallHandler(t, settingsErr.GetMySettings, ctx, "{}"); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("GetMySettings repo error code = %d", code)
	}

	nickname := "g3nick"
	updateProfile := NewHandler(NewServiceWithRepo(&stubUserRepo{putProfileErr: boom}))
	if _, code := g3CallHandler(t, updateProfile.UpdateMyProfile, ctx, `{"nickname":"`+nickname+`"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("UpdateMyProfile repo error code = %d", code)
	}
	updateSettings := NewHandler(NewServiceWithRepo(&stubUserRepo{putSettingsErr: boom}))
	if _, code := g3CallHandler(t, updateSettings.UpdateMySettings, ctx, `{"settings":{"presence_visible":true}}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("UpdateMySettings repo error code = %d", code)
	}
}

func TestG3HandlerWriteServiceErrorWithData(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/g3", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	handler.writeServiceError(rec, req, "G3", ErrPermissionDenied)
	payload := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"] != float64(httpx.CodeForbidden) || payload["data"] == nil {
		t.Fatalf("forbidden response = %#v", payload)
	}
}

func TestG3HandlerSearchBranches(t *testing.T) {
	boom := errors.New("search failed")
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{searchErr: boom}))
	ctx := userCtx("g3_user")
	if _, code := g3CallHandler(t, handler.Search, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("Search malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.Search, ctx, `{"keyword":"alice"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("Search repo error code = %d", code)
	}
}

func TestG3HandlerOpenTheDoorBranches(t *testing.T) {
	boom := errors.New("boom")
	ctx := userCtx("g3_user")
	notFound := NewHandler(NewServiceWithRepo(&stubUserRepo{}))
	if _, code := g3CallHandler(t, notFound.OpenTheDoor, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("OpenTheDoor malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, notFound.OpenTheDoor, ctx, `{"user_id":"missing_user"}`); code != int(httpx.CodeNotFound) {
		t.Fatalf("OpenTheDoor missing user code = %d", code)
	}
	repoErr := NewHandler(NewServiceWithRepo(&stubUserRepo{profileErr: boom}))
	if _, code := g3CallHandler(t, repoErr.OpenTheDoor, ctx, `{"user_id":"u2"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("OpenTheDoor repo error code = %d", code)
	}
}

func TestG3HandlerBatchGetServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{profilesErr: errors.New("boom")}))
	if _, code := g3CallHandler(t, handler.BatchGet, userCtx("u1"), `{"user_ids":["u2"]}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("BatchGet repo error code = %d", code)
	}
}

func TestG3HandlerUpdateLocationBranches(t *testing.T) {
	boom := errors.New("boom")
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{putProfileErr: boom}))
	ctx := userCtx("g3_user")
	if _, code := g3CallHandler(t, handler.UpdateLocation, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("UpdateLocation malformed body code = %d", code)
	}
	outOfRange := `{"location":{"point":{"latitude":999,"longitude":0},"location_name":"x"}}`
	if _, code := g3CallHandler(t, handler.UpdateLocation, ctx, outOfRange); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("UpdateLocation out-of-range code = %d", code)
	}
	inRange := `{"location":{"point":{"latitude":34.2,"longitude":117.1},"location_name":"徐州"}}`
	if _, code := g3CallHandler(t, handler.UpdateLocation, ctx, inRange); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("UpdateLocation repo error code = %d", code)
	}
}

func TestG3HandlerUpgradeBranches(t *testing.T) {
	boom := errors.New("boom")
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{profileErr: boom}))
	ctx := userCtx("g3_user")
	if _, code := g3CallHandler(t, handler.Upgrade, ctx, `{"login_id":"","password":""}`); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("Upgrade empty body code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.Upgrade, ctx, `{"login_id":"username01","password":"pass"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("Upgrade repo error code = %d", code)
	}
}

func TestG3IsValidLoginIDBranches(t *testing.T) {
	cases := map[string]bool{
		"a b@example.com":       false,
		"a@@b.com":              false,
		"a@b":                   false,
		"a@.b.com":              false,
		"a@b.com.":              false,
		"a@b..com":              false,
		"valid@example.com":     true,
		"username01":            true,
		"13800000000":           false,
		"ab":                    false,
		strings.Repeat("a", 65): false,
	}
	for input, want := range cases {
		if got := isValidLoginID(input); got != want {
			t.Fatalf("isValidLoginID(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestG3HandlerDeleteAccountDecodeError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{}))
	ctx := userCtx("g3_user")
	if _, code := g3CallHandler(t, handler.DeleteAccount, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("DeleteAccount malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.DeleteAccount, ctx, `{"confirmation":"DELETE"}`); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("DeleteAccount wrong confirmation code = %d", code)
	}
}

func TestG3HandlerFollowBranches(t *testing.T) {
	ctx := userCtx("g3_user")
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{exists: true, settingsFound: true, settings: defaultSettings(), unfollowErr: errors.New("boom")}))
	if _, code := g3CallHandler(t, handler.Follow, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("Follow malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.Follow, ctx, `{"user_id":"u2","active":true,"source":"moment"}`); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("Follow moment without source moment id code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.Follow, ctx, `{"user_id":"u2","active":true,"source_moment_id":"m1"}`); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("Follow source moment id without moment source code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.Follow, ctx, `{"user_id":"u2","active":false,"source":"profile","source_moment_id":"m1"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("Follow unfollow repo error code = %d", code)
	}
}

func TestG3HandlerBlockBranches(t *testing.T) {
	ctx := userCtx("g3_user")
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{exists: true, blockErr: errors.New("boom")}))
	if _, code := g3CallHandler(t, handler.Block, ctx, `{"user_id":"u2","active":true}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("Block repo error code = %d", code)
	}
}

func TestG3HandlerListBlockedDecodeError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{}))
	if _, code := g3CallHandler(t, handler.ListBlocked, userCtx("g3_user"), "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("ListBlocked malformed body code = %d", code)
	}
}

func TestG3HandlerSetRemarkBranches(t *testing.T) {
	ctx := userCtx("g3_user")
	handler := NewHandler(NewServiceWithRepo(&stubUserRepo{exists: true, relation: Relationship{IsFollowing: true}, setRemarkErr: errors.New("boom")}))
	if _, code := g3CallHandler(t, handler.SetRemark, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("SetRemark malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, handler.SetRemark, ctx, `{"user_id":"u2","remark":"buddy"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("SetRemark repo error code = %d", code)
	}
}

func TestG3HandlerListReactedMomentsServiceError(t *testing.T) {
	repo := &stubUserRepo{exists: true, reactedErr: errors.New("boom")}
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(repo, repo)
	handler := NewHandler(svc)
	if _, code := g3CallHandler(t, handler.ListReactedMoments, context.Background(), `{"user_id":"u2"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("ListReactedMoments provider error code = %d", code)
	}
}

func TestG3PublicHandlerBranches(t *testing.T) {
	boom := errors.New("boom")
	ctx := context.Background()

	searchHandler := NewHandler(NewServiceWithRepo(&stubUserRepo{searchErr: boom}))
	if _, code := g3CallHandler(t, searchHandler.PublicSearch, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("PublicSearch malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, searchHandler.PublicSearch, ctx, `{"keyword":"alice"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("PublicSearch repo error code = %d", code)
	}

	notFound := NewHandler(NewServiceWithRepo(&stubUserRepo{}))
	if _, code := g3CallHandler(t, notFound.PublicOpenTheDoor, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("PublicOpenTheDoor malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, notFound.PublicOpenTheDoor, ctx, `{"user_id":"missing"}`); code != int(httpx.CodeNotFound) {
		t.Fatalf("PublicOpenTheDoor missing user code = %d", code)
	}
	openErr := NewHandler(NewServiceWithRepo(&stubUserRepo{profileErr: boom}))
	if _, code := g3CallHandler(t, openErr.PublicOpenTheDoor, ctx, `{"user_id":"u2"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("PublicOpenTheDoor repo error code = %d", code)
	}

	batchErr := NewHandler(NewServiceWithRepo(&stubUserRepo{profilesErr: boom}))
	if _, code := g3CallHandler(t, batchErr.PublicBatchGet, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("PublicBatchGet malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, batchErr.PublicBatchGet, ctx, `{"user_ids":["u2"]}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("PublicBatchGet repo error code = %d", code)
	}

	listErrRepo := &stubUserRepo{exists: true, settingsErr: boom}
	listErr := NewHandler(NewServiceWithRepo(listErrRepo))
	for name, fn := range map[string]http.HandlerFunc{
		"friends":   listErr.PublicListFriends,
		"following": listErr.PublicListFollowing,
		"followers": listErr.PublicListFollowers,
		"groups":    listErr.PublicListGroups,
	} {
		if _, code := g3CallHandler(t, fn, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
			t.Fatalf("%s malformed body code = %d", name, code)
		}
		if _, code := g3CallHandler(t, fn, ctx, `{"user_id":"u2"}`); code != int(httpx.CodeInternalUnavailable) {
			t.Fatalf("%s settings error code = %d", name, code)
		}
	}

	momentsRepo := &stubUserRepo{exists: true, momentsErr: boom, reactedErr: boom}
	momentsSvc := NewServiceWithRepo(momentsRepo)
	momentsSvc.SetContentProviders(momentsRepo, momentsRepo)
	momentsHandler := NewHandler(momentsSvc)
	if _, code := g3CallHandler(t, momentsHandler.PublicListMoments, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("PublicListMoments malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, momentsHandler.PublicListMoments, ctx, `{"user_id":"u2"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("PublicListMoments provider error code = %d", code)
	}
	if _, code := g3CallHandler(t, momentsHandler.PublicListReactedMoments, ctx, "not-json"); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("PublicListReactedMoments malformed body code = %d", code)
	}
	if _, code := g3CallHandler(t, momentsHandler.PublicListReactedMoments, ctx, `{"user_id":"u2"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("PublicListReactedMoments provider error code = %d", code)
	}
}

func TestG3ToPublicPaginatedUsersNonEmpty(t *testing.T) {
	page := toPublicPaginatedUsers(PaginatedUsers{Items: []UserItem{{User: UserInfo{UserID: "u1", Nickname: "One"}}}})
	if len(page.Items) != 1 || page.Items[0].User.UserID != "u1" {
		t.Fatalf("converted page = %#v", page)
	}
	if page.Items[0].Tags == nil {
		t.Fatal("nil tags must be converted to empty slice")
	}
}

func TestG3HandlerSuccessBranches(t *testing.T) {
	ctx := userCtx("g3_user")

	searchRepo := &stubUserRepo{
		searchResult:  PaginatedProfiles{Items: []MyProfile{{UserID: "u2", Nickname: "Two"}}},
		settingsFound: true,
		settings:      defaultSettings(),
	}
	if _, code := g3CallHandler(t, NewHandler(NewServiceWithRepo(searchRepo)).Search, ctx, `{"keyword":"two"}`); code != 0 {
		t.Fatalf("Search success code = %d", code)
	}

	if _, code := g3CallHandler(t, NewHandler(NewServiceWithRepo(&stubUserRepo{profileFound: true})).UpdateLocation, ctx, `{"location":{"point":{"latitude":34.2,"longitude":117.1},"location_name":"徐州"}}`); code != 0 {
		t.Fatalf("UpdateLocation success code = %d", code)
	}

	temp := testProfile("g3_user")
	temp.UserType = UserTypeTemp
	upgradeSvc := NewServiceWithRepo(&stubUserRepo{profile: temp, profileFound: true})
	upgradeSvc.SetCredentialWriter(&stubCredentialWriter{}, noopTxRunner)
	if _, code := g3CallHandler(t, NewHandler(upgradeSvc).Upgrade, ctx, `{"login_id":"username01","password":"pass"}`); code != 0 {
		t.Fatalf("Upgrade success code = %d", code)
	}

	deleteSvc := NewServiceWithRepo(&stubUserRepo{})
	deleteSvc.SetAccountPurger(&stubAccountPurger{})
	if _, code := g3CallHandler(t, NewHandler(deleteSvc).DeleteAccount, ctx, `{"confirmation":"DELETE_MY_ACCOUNT"}`); code != 0 {
		t.Fatalf("DeleteAccount success code = %d", code)
	}

	reactedRepo := &stubUserRepo{exists: true}
	reactedSvc := NewServiceWithRepo(reactedRepo)
	reactedSvc.SetContentProviders(nil, reactedRepo)
	if _, code := g3CallHandler(t, NewHandler(reactedSvc).ListReactedMoments, context.Background(), `{"user_id":"u2"}`); code != 0 {
		t.Fatalf("ListReactedMoments success code = %d", code)
	}

	settingsSvc := NewServiceWithRepo(&stubUserRepo{settingsFound: true, settings: defaultSettings()})
	if _, code := g3CallHandler(t, NewHandler(settingsSvc).UpdateMySettings, ctx, `{"settings":{"presence_visible":false}}`); code != 0 {
		t.Fatalf("UpdateMySettings success code = %d", code)
	}

	// PublicBatchGet 超过 100 个 id 的分支。
	ids := ""
	for i := 0; i < 101; i++ {
		if i > 0 {
			ids += ","
		}
		ids += `"u` + string(rune('a'+i%26)) + `"`
	}
	publicBatch := NewHandler(NewServiceWithRepo(&stubUserRepo{}))
	if _, code := g3CallHandler(t, publicBatch.PublicBatchGet, context.Background(), `{"user_ids":[`+ids+`]}`); code != int(httpx.CodeInvalidRequest) {
		t.Fatalf("PublicBatchGet oversized code = %d", code)
	}
}

func TestG3HandlerDeleteAccountServiceError(t *testing.T) {
	svc := NewServiceWithRepo(&stubUserRepo{})
	svc.SetAccountPurger(&stubAccountPurger{err: errors.New("purge boom")})
	if _, code := g3CallHandler(t, NewHandler(svc).DeleteAccount, userCtx("g3_user"), `{"confirmation":"DELETE_MY_ACCOUNT"}`); code != int(httpx.CodeInternalUnavailable) {
		t.Fatalf("DeleteAccount service error code = %d", code)
	}
}
