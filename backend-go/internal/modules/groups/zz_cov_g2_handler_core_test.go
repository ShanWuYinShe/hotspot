package groups

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// TestCovHandlerCreateErrorPaths 覆盖 groups Create 的参数/服务错误映射。
func TestCovHandlerCreateErrorPaths(t *testing.T) {
	validBody := `{"name":"n","location":{"location_name":"loc"}}`
	repo := func() *covRepo { return covNewRepo(covOwnerGroup()) }
	covRunCases(t, []covCase{
		{name: "missing idempotency key", repo: repo(), fn: func(h *Handler) http.HandlerFunc { return h.Create }, body: validBody, want: httpx.CodeInvalidRequest},
		{name: "invalid json", repo: repo(), fn: func(h *Handler) http.HandlerFunc { return h.Create }, body: "{", idemKey: true, want: httpx.CodeInvalidRequest},
		{name: "empty name", repo: repo(), fn: func(h *Handler) http.HandlerFunc { return h.Create }, body: `{"name":"","location":{"location_name":"loc"}}`, idemKey: true, want: httpx.CodeInvalidRequest},
		{name: "invalid coordinates", repo: repo(), fn: func(h *Handler) http.HandlerFunc { return h.Create }, body: `{"name":"n","location":{"point":{"latitude":999,"longitude":0},"location_name":"loc"}}`, idemKey: true, want: httpx.CodeInvalidRequest},
		{name: "service validation api error", repo: repo(), fn: func(h *Handler) http.HandlerFunc { return h.Create }, body: `{"name":"` + strings.Repeat("a", 51) + `","location":{"location_name":"loc"}}`, idemKey: true, want: httpx.CodeInvalidRequest},
	})
}

func TestCovHandlerCreateRepoError(t *testing.T) {
	repo := covNewRepo(covOwnerGroup())
	repo.createErr = errors.New("boom")
	code := covRunHandler(t, repo, func(h *Handler) http.HandlerFunc { return h.Create },
		`{"name":"n","location":{"location_name":"loc"}}`, "user_alice", true)
	if code != httpx.CodeInternalUnavailable {
		t.Fatalf("code = %d, want 10600", code)
	}
}

type covBlockedTagChecker struct{ blocked string }

func (c covBlockedTagChecker) IsBlocked(tag string) bool { return tag == c.blocked }

func TestCovHandlerCreateBlockedTag(t *testing.T) {
	SetBlockedTagChecker(covBlockedTagChecker{blocked: "blocked-tag"})
	t.Cleanup(func() { SetBlockedTagChecker(nil) })
	code := covRunHandler(t, covNewRepo(covOwnerGroup()), func(h *Handler) http.HandlerFunc { return h.Create },
		`{"name":"n","tags":["blocked-tag"],"location":{"location_name":"loc"}}`, "user_alice", true)
	if code != httpx.CodeConflict {
		t.Fatalf("code = %d, want 10400", code)
	}
}

func TestCovHandlerCreateMalformedIdempotencyKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"n","location":{"location_name":"loc"}}`))
	req.Header.Set("Idempotency-Key", "short")
	rr := httptest.NewRecorder()
	NewHandler(NewServiceWithRepo(covNewRepo(covOwnerGroup()))).Create(rr, req)
	if code := decodeCovCode(t, rr); code != httpx.CodeIdempotencyConflict {
		t.Fatalf("code = %d, want 22000", code)
	}
}

func TestCovHandlerOpenTheDoorErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.Forbidden(httpx.ReasonPrivacySetting, "no")
	genericErrRepo := covNewRepo(covOwnerGroup())
	genericErrRepo.getErr = errors.New("boom")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.OpenTheDoor }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.OpenTheDoor }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.OpenTheDoor }, body: body, want: httpx.CodeForbidden},
		{name: "generic error", repo: genericErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.OpenTheDoor }, body: body, want: httpx.CodeInternalUnavailable},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.OpenTheDoor }, body: body, want: httpx.CodeNotFound},
	})
}

func TestCovHandlerSearchErrorPaths(t *testing.T) {
	keywordBody := `{"keyword":"hot"}`
	withSearchErr := func(err error) *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.searchErr = err
		return repo
	}
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "no criteria", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "radius out of range", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: `{"keyword":"hot","radius":1}`, want: httpx.CodeInvalidRequest},
		{name: "keyword too long", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: `{"keyword":"` + strings.Repeat("a", 101) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "unsupported sort", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: `{"keyword":"hot","sort_by":"magic"}`, want: httpx.CodeInvalidRequest},
		{name: "api error", repo: withSearchErr(httpx.InvalidRequest("bad")), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: keywordBody, want: httpx.CodeInvalidRequest},
		{name: "access restricted", repo: withSearchErr(ErrAccessRestricted), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: keywordBody, want: httpx.CodeForbidden},
		{name: "invalid cursor", repo: withSearchErr(ErrInvalidCursor), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: keywordBody, want: httpx.CodeInvalidRequest},
		{name: "generic error", repo: withSearchErr(errors.New("boom")), fn: func(h *Handler) http.HandlerFunc { return h.Search }, body: keywordBody, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerUpdateErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	memberRepo := covNewRepo(covOwnerGroup())
	memberRepo.getResult = covMemberGroup()
	restrictedRepo := covNewRepo(covOwnerGroup())
	restrictedRepo.updateErr = ErrAccessRestricted
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.updateErr = errors.New("boom")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "password conflict", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: `{"group_id":"g1","clear_password":true,"password":"abcdefgh"}`, want: httpx.CodeInvalidRequest},
		{name: "invalid coordinates", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: `{"group_id":"g1","location":{"point":{"latitude":-91,"longitude":0},"location_name":"loc"}}`, want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: body, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: body, want: httpx.CodeNotFound},
		{name: "non admin viewer", repo: memberRepo, fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: body, want: httpx.CodeForbidden},
		{name: "access restricted repo error", repo: restrictedRepo, fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: body, want: httpx.CodeForbidden},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.Update }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerDismissErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	ownerRepo := func() *covRepo { return covNewRepo(covOwnerGroup()) }
	memberRepo := covNewRepo(covOwnerGroup())
	memberRepo.getResult = covMemberGroup()
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	getErrRepo := covNewRepo(covOwnerGroup())
	getErrRepo.getErr = errors.New("load failed")
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.dismissErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: ownerRepo(), fn: func(h *Handler) http.HandlerFunc { return h.Dismiss }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: ownerRepo(), fn: func(h *Handler) http.HandlerFunc { return h.Dismiss }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "get error", repo: getErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.Dismiss }, body: body, want: httpx.CodeInternalUnavailable},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.Dismiss }, body: body, want: httpx.CodeNotFound},
		{name: "invalid reason", repo: ownerRepo(), fn: func(h *Handler) http.HandlerFunc { return h.Dismiss }, body: `{"group_id":"g1","reason":"` + strings.Repeat("a", 501) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "not owner", repo: memberRepo, fn: func(h *Handler) http.HandlerFunc { return h.Dismiss }, body: body, want: httpx.CodeForbidden},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.Dismiss }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerListMembersErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	cursorRepo := covNewRepo(covOwnerGroup())
	cursorRepo.isMemberResult = true
	cursorRepo.listMembersErr = ErrInvalidCursor
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.isMemberResult = true
	genericRepo.listMembersErr = errors.New("boom")
	isMemberErrRepo := covNewRepo(covOwnerGroup())
	isMemberErrRepo.isMemberErr = errors.New("probe failed")
	notEligibleRepo := covNewRepo(GroupItem{Group: GroupInfo{GroupID: "g1"}})
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: body, userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: body, userID: "user_alice", want: httpx.CodeNotFound},
		{name: "invalid cursor", repo: cursorRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: body, userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
		{name: "is member probe error", repo: isMemberErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
		{name: "not member and not searchable", repo: notEligibleRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListMembers }, body: body, want: httpx.CodeForbidden},
	})
}

func TestCovHandlerSearchMembersErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","keyword":"k"}`
	withErr := func(err error) *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.isMemberResult = true
		repo.searchMembersErr = err
		return repo
	}
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	isMemberErrRepo := covNewRepo(covOwnerGroup())
	isMemberErrRepo.isMemberErr = errors.New("probe failed")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing keyword", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "keyword too long", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: `{"group_id":"g1","keyword":"` + strings.Repeat("a", 101) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: body, userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: body, userID: "user_alice", want: httpx.CodeNotFound},
		{name: "not member", repo: withErr(ErrNotMember), fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: body, userID: "user_alice", want: httpx.CodeForbidden},
		{name: "invalid cursor", repo: withErr(ErrInvalidCursor), fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: body, userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "generic repo error", repo: withErr(errors.New("boom")), fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
		{name: "is member probe error", repo: isMemberErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.SearchMembers }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerListMyGroupsErrorPaths(t *testing.T) {
	withErr := func(err error) *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listMyGroupsErr = err
		return repo
	}
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListMyGroups }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "api error", repo: withErr(httpx.InvalidRequest("bad")), fn: func(h *Handler) http.HandlerFunc { return h.ListMyGroups }, body: "{}", userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "invalid cursor", repo: withErr(ErrInvalidCursor), fn: func(h *Handler) http.HandlerFunc { return h.ListMyGroups }, body: "{}", userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "generic error", repo: withErr(errors.New("boom")), fn: func(h *Handler) http.HandlerFunc { return h.ListMyGroups }, body: "{}", userID: "user_alice", want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerJoinGroupErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	plainGroup := func() GroupItem { return GroupItem{Group: GroupInfo{GroupID: "g1"}} }
	joinErrRepo := func(err error) *covRepo {
		repo := covNewRepo(plainGroup())
		repo.joinGroupErr = err
		return repo
	}
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	passwordRepo := covNewRepo(GroupItem{Group: GroupInfo{GroupID: "g1"}, JoinMethod: GroupJoinMethodPassword})
	hashNilRepo := covNewRepo(GroupItem{Group: GroupInfo{GroupID: "g1"}, JoinMethod: GroupJoinMethodPassword})
	inviteOnlyRepo := covNewRepo(GroupItem{Group: GroupInfo{GroupID: "g1"}, JoinMethod: GroupJoinMethodInviteOnly})
	bannedRepo := covNewRepo(plainGroup())
	bannedRepo.isBannedResult = true
	bannedErrRepo := covNewRepo(plainGroup())
	bannedErrRepo.isBannedErr = errors.New("probe failed")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "message too long", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: `{"group_id":"g1","message":"` + strings.Repeat("a", 201) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeNotFound},
		{name: "password required", repo: passwordRepo, fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeConflict},
		{name: "password hash missing", repo: hashNilRepo, fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: `{"group_id":"g1","password":"abcdefgh"}`, userID: "user_alice", want: httpx.CodeConflict},
		{name: "already applied", repo: joinErrRepo(ErrAlreadyApplied), fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeConflict},
		{name: "group full", repo: joinErrRepo(ErrGroupFull), fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeConflict},
		{name: "invite only", repo: inviteOnlyRepo, fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeConflict},
		{name: "access restricted", repo: joinErrRepo(ErrAccessRestricted), fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeForbidden},
		{name: "banned", repo: bannedRepo, fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeForbidden},
		{name: "ban probe error", repo: bannedErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
		{name: "generic repo error", repo: joinErrRepo(errors.New("boom")), fn: func(h *Handler) http.HandlerFunc { return h.JoinGroup }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerAnnouncementErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","title":"t","content":"c"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberRepo := covNewRepo(covOwnerGroup())
	memberRepo.getResult = covMemberGroup()
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.createAnnouncementErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "create missing key", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.CreateAnnouncement }, body: body, want: httpx.CodeInvalidRequest},
		{name: "create invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.CreateAnnouncement }, body: "{", idemKey: true, want: httpx.CodeInvalidRequest},
		{name: "create empty title", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.CreateAnnouncement }, body: `{"group_id":"g1","title":"","content":"c"}`, idemKey: true, want: httpx.CodeInvalidRequest},
		{name: "create api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.CreateAnnouncement }, body: body, idemKey: true, want: httpx.CodeInvalidRequest},
		{name: "create not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.CreateAnnouncement }, body: body, idemKey: true, want: httpx.CodeNotFound},
		{name: "create not admin", repo: memberRepo, fn: func(h *Handler) http.HandlerFunc { return h.CreateAnnouncement }, body: body, idemKey: true, want: httpx.CodeForbidden},
		{name: "create generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.CreateAnnouncement }, body: body, idemKey: true, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerListAnnouncementsErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.listAnnouncementsErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.listAnnouncementsErr = ErrGroupNotFound
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.listAnnouncementsErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListAnnouncements }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "empty group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListAnnouncements }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListAnnouncements }, body: body, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListAnnouncements }, body: body, want: httpx.CodeNotFound},
		{name: "generic error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.ListAnnouncements }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerUpdateAnnouncementErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","announcement_id":"a1"}`
	apiErrRepo := covNewRepo(covOwnerGroup())
	apiErrRepo.getErr = httpx.InvalidRequest("bad")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberRepo := covNewRepo(covOwnerGroup())
	memberRepo.getResult = covMemberGroup()
	notFoundRepo := covNewRepo(covOwnerGroup())
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.updateAnnouncementErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.UpdateAnnouncement }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing announcement id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.UpdateAnnouncement }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "api error", repo: apiErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdateAnnouncement }, body: body, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdateAnnouncement }, body: body, want: httpx.CodeNotFound},
		{name: "not admin", repo: memberRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdateAnnouncement }, body: body, want: httpx.CodeForbidden},
		{name: "announcement not found", repo: notFoundRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdateAnnouncement }, body: body, want: httpx.CodeNotFound},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdateAnnouncement }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerDeleteAnnouncementErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","announcement_id":"a1"}`
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	memberRepo := covNewRepo(covOwnerGroup())
	memberRepo.getResult = covMemberGroup()
	notFoundRepo := covNewRepo(covOwnerGroup())
	notFoundRepo.deleteAnnouncementErr = ErrAnnouncementNotFound
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.deleteAnnouncementErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.DeleteAnnouncement }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing announcement id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.DeleteAnnouncement }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.DeleteAnnouncement }, body: body, want: httpx.CodeNotFound},
		{name: "not admin", repo: memberRepo, fn: func(h *Handler) http.HandlerFunc { return h.DeleteAnnouncement }, body: body, want: httpx.CodeForbidden},
		{name: "announcement not found", repo: notFoundRepo, fn: func(h *Handler) http.HandlerFunc { return h.DeleteAnnouncement }, body: body, want: httpx.CodeNotFound},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.DeleteAnnouncement }, body: body, want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerUpdatePreferencesErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	notMemberRepo := covNewRepo(GroupItem{Group: GroupInfo{GroupID: "g1"}})
	restrictedRepo := covNewRepo(covOwnerGroup())
	restrictedRepo.updatePreferencesErr = ErrAccessRestricted
	genericRepo := covNewRepo(covOwnerGroup())
	genericRepo.updatePreferencesErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.UpdatePreferences }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.UpdatePreferences }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdatePreferences }, body: body, userID: "user_alice", want: httpx.CodeNotFound},
		{name: "not member", repo: notMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdatePreferences }, body: body, userID: "user_alice", want: httpx.CodeForbidden},
		{name: "access restricted", repo: restrictedRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdatePreferences }, body: body, userID: "user_alice", want: httpx.CodeForbidden},
		{name: "generic error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.UpdatePreferences }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerLeaveGroupErrorPaths(t *testing.T) {
	body := `{"group_id":"g1"}`
	getErrRepo := covNewRepo(covOwnerGroup())
	getErrRepo.getErr = errors.New("load failed")
	missingRepo := covNewRepo(covOwnerGroup())
	missingRepo.getFound = false
	notMemberRepo := covNewRepo(GroupItem{Group: GroupInfo{GroupID: "g1"}})
	ownerRepo := covNewRepo(covOwnerGroup())
	memberRepo := covNewRepo(covMemberGroup())
	memberRepo.leaveGroupErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.LeaveGroup }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing group id", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.LeaveGroup }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "get error", repo: getErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.LeaveGroup }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
		{name: "not found", repo: missingRepo, fn: func(h *Handler) http.HandlerFunc { return h.LeaveGroup }, body: body, userID: "user_alice", want: httpx.CodeNotFound},
		{name: "not member", repo: notMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.LeaveGroup }, body: body, userID: "user_alice", want: httpx.CodeConflict},
		{name: "owner cannot leave", repo: ownerRepo, fn: func(h *Handler) http.HandlerFunc { return h.LeaveGroup }, body: body, userID: "user_alice", want: httpx.CodeConflict},
		{name: "generic repo error", repo: memberRepo, fn: func(h *Handler) http.HandlerFunc { return h.LeaveGroup }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerListMyApplicationsErrorPaths(t *testing.T) {
	withErr := func(err error) *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listMyApplicationsErr = err
		return repo
	}
	covRunCases(t, []covCase{
		{name: "invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.ListMyApplications }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "api error", repo: withErr(httpx.InvalidRequest("bad")), fn: func(h *Handler) http.HandlerFunc { return h.ListMyApplications }, body: "{}", userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "invalid cursor", repo: withErr(ErrInvalidCursor), fn: func(h *Handler) http.HandlerFunc { return h.ListMyApplications }, body: "{}", userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "generic error", repo: withErr(errors.New("boom")), fn: func(h *Handler) http.HandlerFunc { return h.ListMyApplications }, body: "{}", userID: "user_alice", want: httpx.CodeInternalUnavailable},
	})
}

func TestCovHandlerCancelApplicationErrorPaths(t *testing.T) {
	body := `{"group_id":"g1","application_id":"a1"}`
	pendingApp := func() *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.listApplicationsItems = []GroupApplicationItem{{
			ApplicationID: "a1",
			GroupID:       "g1",
			Applicant:     UserInfo{UserID: "user_alice"},
			Status:        ApplicationStatusPending,
		}}
		return repo
	}
	invalidReasonRepo := pendingApp()
	alreadyRepo := pendingApp()
	alreadyRepo.cancelApplicationErr = ErrAlreadyProcessed
	genericRepo := pendingApp()
	genericRepo.cancelApplicationErr = errors.New("boom")
	covRunCases(t, []covCase{
		{name: "invalid json", repo: pendingApp(), fn: func(h *Handler) http.HandlerFunc { return h.CancelApplication }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "missing application id", repo: pendingApp(), fn: func(h *Handler) http.HandlerFunc { return h.CancelApplication }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "application not found", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.CancelApplication }, body: body, userID: "user_alice", want: httpx.CodeNotFound},
		{name: "invalid reason", repo: invalidReasonRepo, fn: func(h *Handler) http.HandlerFunc { return h.CancelApplication }, body: `{"group_id":"g1","application_id":"a1","reason":"` + strings.Repeat("a", 201) + `"}`, userID: "user_alice", want: httpx.CodeInvalidRequest},
		{name: "already processed", repo: alreadyRepo, fn: func(h *Handler) http.HandlerFunc { return h.CancelApplication }, body: body, userID: "user_alice", want: httpx.CodeConflict},
		{name: "generic repo error", repo: genericRepo, fn: func(h *Handler) http.HandlerFunc { return h.CancelApplication }, body: body, userID: "user_alice", want: httpx.CodeInternalUnavailable},
	})
}

// decodeCovCode 解析响应 envelope 中的业务错误码。
func decodeCovCode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.Code
}
