package comments

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

func callPublic(t *testing.T, handler *Handler, call func(http.ResponseWriter, *http.Request), body string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/public/v1/comments", strings.NewReader(body))
	call(rr, req)
	return rr
}

func TestPublicFetchSuccessAndErrorBranches(t *testing.T) {
	repo := &stubCommentRepo{listResult: PaginatedComments{Items: []CommentItem{
		{CommentID: "c1", Content: "hello", Reactions: []Reaction{{Emoji: "👍", Count: 2, ReactedByMe: true}}},
	}}}
	handler := newTestHandler(repo, defaultMoments())
	rr := callPublic(t, handler, handler.PublicFetch, "{\"moment_id\":\"m1\"}")
	resp := decodeResponse[PublicPaginatedComments](t, rr)
	if resp.Code != 0 || len(resp.Data.Items) != 1 || resp.Data.Items[0].CommentID != "c1" {
		t.Fatalf("unexpected public fetch response: %#v body=%s", resp, rr.Body.String())
	}
	if len(resp.Data.Items[0].Reactions) != 1 || resp.Data.Items[0].Reactions[0].Emoji != "👍" || resp.Data.Items[0].Reactions[0].Count != 2 {
		t.Fatalf("public reactions must drop viewer state: %#v", resp.Data.Items[0].Reactions)
	}
	if strings.Contains(rr.Body.String(), "reacted_by_me") {
		t.Fatalf("public response must not expose reacted_by_me: %s", rr.Body.String())
	}

	repo.listErr = ErrInvalidCursor
	rr = callPublic(t, handler, handler.PublicFetch, "{\"moment_id\":\"m1\"}")
	assertCode(t, rr, httpx.CodeInvalidRequest)

	repo.listErr = errors.New("list down")
	rr = callPublic(t, handler, handler.PublicFetch, "{\"moment_id\":\"m1\"}")
	assertCode(t, rr, httpx.CodeInternalUnavailable)

	rr = callPublic(t, handler, handler.PublicFetch, "{bad")
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestPublicFetchRepliesSuccessAndErrorBranches(t *testing.T) {
	repo := &stubCommentRepo{
		getResult:  CommentItem{CommentID: "c1", MomentID: "m1"},
		getFound:   true,
		repliesRes: PaginatedComments{Items: []CommentItem{{CommentID: "r1", Content: "reply"}}},
	}
	handler := newTestHandler(repo, defaultMoments())
	rr := callPublic(t, handler, handler.PublicFetchReplies, "{\"comment_id\":\"c1\"}")
	resp := decodeResponse[PublicPaginatedComments](t, rr)
	if resp.Code != 0 || len(resp.Data.Items) != 1 || resp.Data.Items[0].CommentID != "r1" {
		t.Fatalf("unexpected public replies response: %#v body=%s", resp, rr.Body.String())
	}

	repo.repliesErr = ErrInvalidCursor
	rr = callPublic(t, handler, handler.PublicFetchReplies, "{\"comment_id\":\"c1\"}")
	assertCode(t, rr, httpx.CodeInvalidRequest)

	repo.repliesErr = errors.New("replies down")
	rr = callPublic(t, handler, handler.PublicFetchReplies, "{\"comment_id\":\"c1\"}")
	assertCode(t, rr, httpx.CodeInternalUnavailable)

	rr = callPublic(t, handler, handler.PublicFetchReplies, "{bad")
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestPublicOpenTheDoorSuccessAndErrorBranches(t *testing.T) {
	repo := &stubCommentRepo{
		getResult: CommentItem{CommentID: "c1", MomentID: "m1", Content: "hello", User: UserInfo{UserID: "u1"}},
		getFound:  true,
	}
	handler := newTestHandler(repo, defaultMoments())
	rr := callPublic(t, handler, handler.PublicOpenTheDoor, "{\"comment_id\":\"c1\"}")
	resp := decodeResponse[PublicCommentItem](t, rr)
	if resp.Code != 0 || resp.Data.CommentID != "c1" || resp.Data.Content != "hello" {
		t.Fatalf("unexpected public open response: %#v body=%s", resp, rr.Body.String())
	}

	repo.getErr = errors.New("get down")
	rr = callPublic(t, handler, handler.PublicOpenTheDoor, "{\"comment_id\":\"c1\"}")
	assertCode(t, rr, httpx.CodeInternalUnavailable)

	rr = callPublic(t, handler, handler.PublicOpenTheDoor, "{bad")
	assertCode(t, rr, httpx.CodeInvalidRequest)
}
