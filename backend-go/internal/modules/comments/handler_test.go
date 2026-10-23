package comments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

// --- stub repo ---

type stubCommentRepo struct {
	created    CommentItem
	createErr  error
	deleteErr  error
	listResult PaginatedComments
	listErr    error
	getResult  CommentItem
	getFound   bool
	getErr     error
	repliesRes PaginatedComments
	repliesErr error
}

func (r *stubCommentRepo) Create(_ context.Context, _ string, _ CreateCommentRequest) (CommentItem, error) {
	return r.created, r.createErr
}
func (r *stubCommentRepo) SoftDelete(_ context.Context, _ string) (bool, error) {
	return r.deleteErr == nil, r.deleteErr
}
func (r *stubCommentRepo) List(_ context.Context, _ string, _ int, _ string) (PaginatedComments, error) {
	return r.listResult, r.listErr
}
func (r *stubCommentRepo) Get(_ context.Context, _ string) (CommentItem, bool, error) {
	return r.getResult, r.getFound, r.getErr
}
func (r *stubCommentRepo) ListReplies(_ context.Context, _ string, _ int, _ string) (PaginatedComments, error) {
	return r.repliesRes, r.repliesErr
}
func (r *stubCommentRepo) ListTopRepliesByParentIDs(_ context.Context, _ []string, _ int) (map[string][]CommentItem, error) {
	return map[string][]CommentItem{}, nil
}

// --- stub moment provider ---

type stubMomentProvider struct {
	exists           bool
	existsErr        error
	commentableFound bool
	commentable      bool
	commentableErr   error
	incrErr          error
	decrErr          error
	notifCtx         MomentNotificationContext
	notifOk          bool
	notifErr         error
	visibleFound     bool
	visible          bool
	visibleErr       error
}

func (p *stubMomentProvider) Exists(_ context.Context, _ string) (bool, error) {
	return p.exists, p.existsErr
}
func (p *stubMomentProvider) CheckCommentable(_ context.Context, _ string) (bool, bool, error) {
	return p.commentableFound, p.commentable, p.commentableErr
}
func (p *stubMomentProvider) IsReactionTargetVisible(_ context.Context, _ string) (bool, bool, error) {
	return p.visibleFound, p.visible, p.visibleErr
}
func (p *stubMomentProvider) IncrementCommentCount(_ context.Context, _ string) error {
	return p.incrErr
}
func (p *stubMomentProvider) DecrementCommentCount(_ context.Context, _ string) error {
	return p.decrErr
}
func (p *stubMomentProvider) NotificationContext(_ context.Context, _ string) (MomentNotificationContext, bool, error) {
	return p.notifCtx, p.notifOk, p.notifErr
}

// --- stub notification writer ---

type stubNotifWriter struct{}

func (s *stubNotifWriter) AppendMomentCommented(_ context.Context, _ string, _ types.UserInfo, _ string, _ types.MomentSummary, _, _ string) (string, error) {
	return "notif_stub", nil
}
func (s *stubNotifWriter) AppendCommentReply(_ context.Context, _ string, _ types.UserInfo, _ string, _ types.MomentSummary, _, _, _, _ string) (string, error) {
	return "notif_stub", nil
}
func (s *stubNotifWriter) AppendMentionedBatch(_ context.Context, targetUserIDs []string, _ types.UserInfo, _, _ string, _ types.MomentSummary, _, _ string) (map[string]string, error) {
	ids := make(map[string]string, len(targetUserIDs))
	for i, targetUserID := range targetUserIDs {
		ids[targetUserID] = "notif_stub_" + strconv.Itoa(i)
	}
	return ids, nil
}

// --- helpers ---

func newTestHandler(repo *stubCommentRepo, moments *stubMomentProvider) *Handler {
	svc := NewServiceWithDeps(repo, moments, &stubNotifWriter{}, nil, nil, nil)
	return NewHandler(svc)
}

func withAuth(req *http.Request, userID string) *http.Request {
	return req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: userID}))
}

func setIDKey(req *http.Request) {
	req.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
}

type apiResponse[T any] struct {
	Code    httpx.Code `json:"code"`
	Data    T          `json:"data"`
	Message string     `json:"message,omitempty"`
}

func decodeResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) apiResponse[T] {
	t.Helper()
	var resp apiResponse[T]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func assertCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	resp := decodeResponse[json.RawMessage](t, rr)
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %d (body: %s)", expected, resp.Code, rr.Body.String())
	}
}

func defaultMoments() *stubMomentProvider {
	return &stubMomentProvider{exists: true, commentableFound: true, commentable: true, visibleFound: true, visible: true, notifCtx: MomentNotificationContext{UserID: "author1", Summary: types.MomentSummary{MomentID: "m1"}}, notifOk: true}
}

// --- Create tests ---

func TestHandlerCreateHappyPath(t *testing.T) {
	repo := &stubCommentRepo{created: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "u1"}, Content: "hi"}}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1","content":"hi"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "aaaa1111aaaa1111aaaa1111aaaa1111")

	handler.Create(rr, req)
	resp := decodeResponse[CommentItem](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected success, got code %d: %s", resp.Code, resp.Message)
	}
	if resp.Data.CommentID != "c1" {
		t.Fatalf("expected c1, got %s", resp.Data.CommentID)
	}
}

func TestHandlerCreateRejectsMissingIdempotencyKey(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1","content":"hi"}`))
	req = withAuth(req, "u1")
	// no Idempotency-Key header

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerCreateRejectsInvalidJSON(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "bbbb1111bbbb1111bbbb1111bbbb1111")

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerCreateRejectsMissingMomentID(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"content":"hi"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "cccc1111cccc1111cccc1111cccc1111")

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerCreateRejectsMissingContent(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "dddd1111dddd1111dddd1111dddd1111")

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerCreateRejectsMomentNotFound(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, &stubMomentProvider{commentableFound: false})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1","content":"hi"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "mmmm1234mmmm1234mmmm1234mmmm1234")

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerCreateRejectsMissingParent(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1","content":"hi","parent_id":"missing"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "qqqq1234qqqq1234qqqq1234qqqq1234")

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerCreatePropagatesError(t *testing.T) {
	repo := &stubCommentRepo{createErr: errors.New("boom")}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1","content":"hi"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "eeee1234eeee1234eeee1234eeee1234")

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestHandlerCreateRejectsPrivateMomentNoPermission(t *testing.T) {
	repo := &stubCommentRepo{created: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "u1"}, Content: "hi"}}
	handler := newTestHandler(repo, &stubMomentProvider{commentableFound: true, commentable: false})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1","content":"hi"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "pppp1234pppp1234pppp1234pppp1234")

	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeForbidden)
}

// --- List tests ---

func TestHandlerListHappyPath(t *testing.T) {
	repo := &stubCommentRepo{listResult: PaginatedComments{Items: []CommentItem{{CommentID: "c1"}}}}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1"}`))
	req = withAuth(req, "u1")

	handler.List(rr, req)
	resp := decodeResponse[PaginatedComments](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected success, got code %d", resp.Code)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
	}
}

func TestHandlerListRejectsInvalidJSON(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	req = withAuth(req, "u1")

	handler.List(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerListRejectsMissingMomentID(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req = withAuth(req, "u1")

	handler.List(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerListRejectsMomentNotFound(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, &stubMomentProvider{exists: false})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1"}`))
	req = withAuth(req, "u1")

	handler.List(rr, req)
	assertCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerListPropagatesError(t *testing.T) {
	repo := &stubCommentRepo{listErr: errors.New("boom")}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1"}`))
	req = withAuth(req, "u1")

	handler.List(rr, req)
	assertCode(t, rr, httpx.CodeInternalUnavailable)
}

// --- Delete tests ---

func TestHandlerDeleteHappyPath(t *testing.T) {
	repo := &stubCommentRepo{getResult: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "u1"}}, getFound: true}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "delete_happy_path_cmnt_aaaaaaaaaa")

	handler.Delete(rr, req)
	assertCode(t, rr, 0)
}

func TestHandlerDeleteRejectsInvalidJSON(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "delete_invalid_json_cmnt_bbbbbbbbb")

	handler.Delete(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerDeleteRejectsMissingCommentID(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "delete_no_comment_id_cmnt_ccccccc")

	handler.Delete(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerDeleteRejectsNotFound(t *testing.T) {
	repo := &stubCommentRepo{getFound: false}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "delete_not_found_cmnt_ddddddddddd")

	handler.Delete(rr, req)
	assertCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerDeleteRejectsNotOwner(t *testing.T) {
	repo := &stubCommentRepo{getResult: CommentItem{CommentID: "c1", User: UserInfo{UserID: "other"}}, getFound: true}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "delete_not_owner_cmnt_eeeeeeeeeeee")

	handler.Delete(rr, req)
	assertCode(t, rr, httpx.CodeForbidden)
}

func TestHandlerDeletePropagatesError(t *testing.T) {
	repo := &stubCommentRepo{getResult: CommentItem{CommentID: "c1", MomentID: "m1", User: UserInfo{UserID: "u1"}}, getFound: true, deleteErr: errors.New("boom")}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1"}`))
	req = withAuth(req, "u1")
	req.Header.Set("Idempotency-Key", "delete_propagate_cmnt_fffffffffffff")

	handler.Delete(rr, req)
	assertCode(t, rr, httpx.CodeInternalUnavailable)
}

// --- ListReplies tests ---

func TestHandlerListRepliesHappyPath(t *testing.T) {
	repo := &stubCommentRepo{getResult: CommentItem{CommentID: "c1"}, getFound: true, repliesRes: PaginatedComments{Items: []CommentItem{{CommentID: "r1"}}}}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1"}`))
	req = withAuth(req, "u1")

	handler.ListReplies(rr, req)
	resp := decodeResponse[PaginatedComments](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected success, got code %d", resp.Code)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
	}
}

func TestHandlerListRepliesRejectsNotFound(t *testing.T) {
	repo := &stubCommentRepo{getFound: false}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1"}`))
	req = withAuth(req, "u1")

	handler.ListReplies(rr, req)
	assertCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerListRepliesRejectsInvalidJSON(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	req = withAuth(req, "u1")

	handler.ListReplies(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

// TestHandlerListMapsInvalidCursorToParamError mirrors the above for the
// List endpoint, which shares the cursor decoder.
func TestHandlerListMapsInvalidCursorToParamError(t *testing.T) {
	repo := &stubCommentRepo{listErr: ErrInvalidCursor}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"moment_id":"m1","cursor":"@@malformed@@"}`))
	req = withAuth(req, "u1")

	handler.List(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

// TestHandlerListRepliesMapsInvalidCursorToParamError mirrors the above for
// the ListReplies endpoint, which also shares the cursor decoder.
func TestHandlerListRepliesMapsInvalidCursorToParamError(t *testing.T) {
	repo := &stubCommentRepo{repliesErr: ErrInvalidCursor}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1","cursor":"@@malformed@@"}`))
	req = withAuth(req, "u1")

	handler.ListReplies(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

// TestDecodeCommentCursorRejectsMalformed verifies the repo-level cursor
// decoder returns ErrInvalidCursor (not the catch-all persistence
// ErrNotImplemented) for every malformed-input path. This is what the
// handlers above rely on for the httpx.CodeInvalidRequest mapping.
func TestDecodeCommentCursorRejectsMalformed(t *testing.T) {
	cases := map[string]string{
		"not base64":        "@@not-base64@@",
		"empty json fields": encodeCommentCursor("", ""),
		"missing id":        encodeCommentCursor("2025-01-01T00:00:00Z", ""),
		"missing at":        encodeCommentCursor("", "u1"),
		"unparseable time":  encodeCommentCursor("not-a-time", "u1"),
	}
	for name, cursor := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := decodeCommentCursor(cursor)
			if !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("expected ErrInvalidCursor for %q, got %v", cursor, err)
			}
		})
	}
}

// TestDecodeCommentCursorAcceptsEmpty verifies an empty cursor is treated
// as "no cursor" (nil error) so the first page request still works.
func TestDecodeCommentCursorAcceptsEmpty(t *testing.T) {
	_, _, err := decodeCommentCursor("")
	if err != nil {
		t.Fatalf("expected nil error for empty cursor, got %v", err)
	}
}

// TestDecodeCommentCursorAcceptsValid verifies a well-formed cursor round-trips.
func TestDecodeCommentCursorAcceptsValid(t *testing.T) {
	const at = "2025-01-01T00:00:00Z"
	const id = "user_abc"
	v, vid, err := decodeCommentCursor(encodeCommentCursor(at, id))
	if err != nil {
		t.Fatalf("expected nil error for valid cursor, got %v", err)
	}
	if vid != id {
		t.Fatalf("expected id %q, got %q", id, vid)
	}
	if v == nil {
		t.Fatal("expected non-nil parsed timestamp")
	}
}

func TestHandlerGetDetailHappyPath(t *testing.T) {
	repo := &stubCommentRepo{
		getResult: CommentItem{CommentID: "c1", Content: "hello"},
		getFound:  true,
	}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := withAuth(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"c1"}`)), "u1")

	handler.GetDetail(rr, req)
	resp := decodeResponse[CommentItem](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.CommentID != "c1" {
		t.Fatalf("expected c1, got %s", resp.Data.CommentID)
	}
}

func TestHandlerGetDetailMapsNotFoundTo10300(t *testing.T) {
	repo := &stubCommentRepo{getFound: false}
	handler := newTestHandler(repo, defaultMoments())
	rr := httptest.NewRecorder()
	req := withAuth(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"comment_id":"missing"}`)), "u1")

	handler.GetDetail(rr, req)
	assertCode(t, rr, httpx.CodeNotFound)
}

func TestHandlerGetDetailRejectsMissingCommentID(t *testing.T) {
	handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
	rr := httptest.NewRecorder()
	req := withAuth(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), "u1")

	handler.GetDetail(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}
