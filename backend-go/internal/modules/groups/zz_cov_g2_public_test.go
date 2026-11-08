package groups

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notify"
	notifytest "hotspot/backend-go/internal/modules/notify/notifytest"
	"hotspot/backend-go/internal/platform/httpx"
)

func TestCovHandlerPublicErrorPaths(t *testing.T) {
	searchBody := `{"keyword":"hot"}`
	withSearchErr := func(err error) *covRepo {
		repo := covNewRepo(covOwnerGroup())
		repo.searchErr = err
		return repo
	}
	openAPIErrRepo := covNewRepo(covOwnerGroup())
	openAPIErrRepo.getErr = httpx.InvalidRequest("bad")
	openGenericErrRepo := covNewRepo(covOwnerGroup())
	openGenericErrRepo.getErr = errors.New("boom")
	openMissingRepo := covNewRepo(covOwnerGroup())
	openMissingRepo.getFound = false

	membersSearchableGroup := func() GroupItem {
		item := covOwnerGroup()
		item.MembersSearchable = true
		return item
	}
	membersAPIErrRepo := covNewRepo(membersSearchableGroup())
	membersAPIErrRepo.getErr = httpx.InvalidRequest("bad")
	membersMissingRepo := covNewRepo(membersSearchableGroup())
	membersMissingRepo.getFound = false
	membersCursorRepo := covNewRepo(membersSearchableGroup())
	membersCursorRepo.listMembersErr = ErrInvalidCursor
	membersGenericRepo := covNewRepo(membersSearchableGroup())
	membersGenericRepo.listMembersErr = errors.New("boom")

	searchMembersAPIErrRepo := covNewRepo(membersSearchableGroup())
	searchMembersAPIErrRepo.getErr = httpx.InvalidRequest("bad")
	searchMembersMissingRepo := covNewRepo(membersSearchableGroup())
	searchMembersMissingRepo.getFound = false
	searchMembersNotMemberRepo := covNewRepo(membersSearchableGroup())
	searchMembersNotMemberRepo.searchMembersErr = ErrNotMember
	searchMembersCursorRepo := covNewRepo(membersSearchableGroup())
	searchMembersCursorRepo.searchMembersErr = ErrInvalidCursor
	searchMembersGenericRepo := covNewRepo(membersSearchableGroup())
	searchMembersGenericRepo.searchMembersErr = errors.New("boom")

	covRunCases(t, []covCase{
		// PublicSearch
		{name: "public search invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "public search no criteria", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "public search radius out of range", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: `{"keyword":"hot","radius":1}`, want: httpx.CodeInvalidRequest},
		{name: "public search keyword too long", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: `{"keyword":"` + strings.Repeat("a", 101) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "public search unsupported sort", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: `{"keyword":"hot","sort_by":"magic"}`, want: httpx.CodeInvalidRequest},
		{name: "public search api error", repo: withSearchErr(httpx.InvalidRequest("bad")), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: searchBody, want: httpx.CodeInvalidRequest},
		{name: "public search access restricted", repo: withSearchErr(ErrAccessRestricted), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: searchBody, want: httpx.CodeForbidden},
		{name: "public search invalid cursor", repo: withSearchErr(ErrInvalidCursor), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: searchBody, want: httpx.CodeInvalidRequest},
		{name: "public search generic error", repo: withSearchErr(errors.New("boom")), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearch }, body: searchBody, want: httpx.CodeInternalUnavailable},
		// PublicOpenTheDoor
		{name: "public open invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicOpenTheDoor }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "public open empty group", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicOpenTheDoor }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "public open api error", repo: openAPIErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicOpenTheDoor }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "public open generic error", repo: openGenericErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicOpenTheDoor }, body: `{"group_id":"g1"}`, want: httpx.CodeInternalUnavailable},
		{name: "public open not found", repo: openMissingRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicOpenTheDoor }, body: `{"group_id":"g1"}`, want: httpx.CodeNotFound},
		// PublicListMembers
		{name: "public members invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicListMembers }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "public members empty group", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicListMembers }, body: "{}", want: httpx.CodeInvalidRequest},
		{name: "public members api error", repo: membersAPIErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicListMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "public members not found", repo: membersMissingRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicListMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeNotFound},
		{name: "public members invalid cursor", repo: membersCursorRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicListMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "public members generic error", repo: membersGenericRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicListMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInternalUnavailable},
		// PublicSearchMembers
		{name: "public search members invalid json", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: "{", want: httpx.CodeInvalidRequest},
		{name: "public search members missing keyword", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: `{"group_id":"g1"}`, want: httpx.CodeInvalidRequest},
		{name: "public search members keyword too long", repo: covNewRepo(covOwnerGroup()), fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: `{"group_id":"g1","keyword":"` + strings.Repeat("a", 101) + `"}`, want: httpx.CodeInvalidRequest},
		{name: "public search members api error", repo: searchMembersAPIErrRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: `{"group_id":"g1","keyword":"k"}`, want: httpx.CodeInvalidRequest},
		{name: "public search members not found", repo: searchMembersMissingRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: `{"group_id":"g1","keyword":"k"}`, want: httpx.CodeNotFound},
		{name: "public search members not member", repo: searchMembersNotMemberRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: `{"group_id":"g1","keyword":"k"}`, want: httpx.CodeForbidden},
		{name: "public search members invalid cursor", repo: searchMembersCursorRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: `{"group_id":"g1","keyword":"k"}`, want: httpx.CodeInvalidRequest},
		{name: "public search members generic error", repo: searchMembersGenericRepo, fn: func(h *Handler) http.HandlerFunc { return h.PublicSearchMembers }, body: `{"group_id":"g1","keyword":"k"}`, want: httpx.CodeInternalUnavailable},
	})
}

// TestCovGroupsModuleMountPublicRoutes 断言匿名节点 5 条路由全部挂载在 /groups 前缀下。
func TestCovGroupsModuleMountPublicRoutes(t *testing.T) {
	router := chi.NewRouter()
	NewModule(nil, nil, notify.NewServiceWithRepo(notifytest.New())).MountPublic(router)

	got := map[string]bool{}
	if err := chi.Walk(router, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == http.MethodPost {
			got[route] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("walk router: %v", err)
	}
	for _, want := range []string{
		"/groups/search",
		"/groups/open_the_door",
		"/groups/members/fetch",
		"/groups/members/search",
		"/groups/announcements/fetch",
	} {
		if !got[want] {
			t.Fatalf("expected public route %q to be mounted, got %v", want, got)
		}
	}
}
