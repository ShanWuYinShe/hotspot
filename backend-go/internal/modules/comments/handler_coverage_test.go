package comments

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

func postHandler(t *testing.T, handler *Handler, call func(http.ResponseWriter, *http.Request), body any, userID string, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	req = withAuth(req, userID)
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	call(rr, req)
	return rr
}

// TestHandlerCreateMapsValidationErrors 覆盖 Create 的 10000 分支：
// 内容超长（ErrCommentTooLong）与 mentions 超上限（ErrTooManyMentions）。
func TestHandlerCreateMapsValidationErrors(t *testing.T) {
	t.Run("content too long", func(t *testing.T) {
		handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
		rr := postHandler(t, handler, handler.Create,
			CreateCommentRequest{MomentID: "m1", Content: strings.Repeat("评", defaultCommentMaxRunes+1)},
			"u1", "cover_create_too_long_key_________")
		assertCode(t, rr, httpx.CodeInvalidRequest)
	})

	t.Run("too many mentions", func(t *testing.T) {
		handler := newTestHandler(&stubCommentRepo{}, defaultMoments())
		mentions := make([]string, mentionsMaxCount+1)
		for i := range mentions {
			mentions[i] = "user_mention"
		}
		rr := postHandler(t, handler, handler.Create,
			CreateCommentRequest{MomentID: "m1", Content: "hi", Mentions: mentions},
			"u1", "cover_create_too_many_mentions___")
		assertCode(t, rr, httpx.CodeInvalidRequest)
	})
}

// TestHandlerCreateMapsBlockedByUser 覆盖 Create 的 10200 双向拉黑分支，
// 并断言契约要求的 data.forbidden_reason。
func TestHandlerCreateMapsBlockedByUser(t *testing.T) {
	newHandler := func(callerBlocked, authorBlocked bool) *Handler {
		svc := NewServiceWithDeps(&stubCommentRepo{}, defaultMoments(), &stubNotifWriter{}, nil, nil, nil)
		svc.SetBlockChecker(func(string, string) (bool, bool, error) {
			return callerBlocked, authorBlocked, nil
		})
		return NewHandler(svc)
	}

	cases := []struct {
		name          string
		callerBlocked bool
		authorBlocked bool
		idemKey       string
		message       string
	}{
		{"blocked by author", false, true, "cover_create_blocked_by_author___", "你已被该用户拉黑"},
		{"caller blocks author", true, false, "cover_create_blocking_author____", "你已拉黑该用户"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := newHandler(tc.callerBlocked, tc.authorBlocked)
			rr := postHandler(t, handler, handler.Create,
				CreateCommentRequest{MomentID: "m1", Content: "hi"},
				"u1", tc.idemKey)
			resp := decodeResponse[httpx.ForbiddenErrorData](t, rr)
			if resp.Code != httpx.CodeForbidden {
				t.Fatalf("expected 10200, got %d body=%s", resp.Code, rr.Body.String())
			}
			if resp.Data.ForbiddenReason != httpx.ReasonBlockedByUser {
				t.Fatalf("expected forbidden_reason=%s, got %q", httpx.ReasonBlockedByUser, resp.Data.ForbiddenReason)
			}
			if resp.Message != tc.message {
				t.Fatalf("expected message %q, got %q", tc.message, resp.Message)
			}
		})
	}
}

// TestHandlerListMapsRepoInvalidCursor 覆盖 List 的 ErrInvalidCursor → 10000
// 映射（请求体合法，由仓储层返回非法游标错误）。
func TestHandlerListMapsRepoInvalidCursor(t *testing.T) {
	repo := &stubCommentRepo{listErr: ErrInvalidCursor}
	handler := newTestHandler(repo, defaultMoments())
	rr := postHandler(t, handler, handler.List, ListCommentsRequest{MomentID: "m1"}, "u1", "")
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

// TestHandlerGetDetailMapsServiceError 覆盖 GetDetail 的仓储错误 → 10600。
func TestHandlerGetDetailMapsServiceError(t *testing.T) {
	repo := &stubCommentRepo{getErr: errors.New("get down")}
	handler := newTestHandler(repo, defaultMoments())
	rr := postHandler(t, handler, handler.GetDetail, GetCommentRequest{CommentID: "c1"}, "u1", "")
	assertCode(t, rr, httpx.CodeInternalUnavailable)
}

// TestHandlerListRepliesMapsErrors 覆盖 ListReplies 的非法游标（10000）与
// 仓储错误（10600）分支。
func TestHandlerListRepliesMapsErrors(t *testing.T) {
	t.Run("invalid cursor", func(t *testing.T) {
		repo := &stubCommentRepo{
			getResult:  CommentItem{CommentID: "c1", MomentID: "m1"},
			getFound:   true,
			repliesErr: ErrInvalidCursor,
		}
		handler := newTestHandler(repo, defaultMoments())
		rr := postHandler(t, handler, handler.ListReplies, CommentRepliesRequest{CommentID: "c1"}, "u1", "")
		assertCode(t, rr, httpx.CodeInvalidRequest)
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &stubCommentRepo{
			getResult:  CommentItem{CommentID: "c1", MomentID: "m1"},
			getFound:   true,
			repliesErr: errors.New("replies down"),
		}
		handler := newTestHandler(repo, defaultMoments())
		rr := postHandler(t, handler, handler.ListReplies, CommentRepliesRequest{CommentID: "c1"}, "u1", "")
		assertCode(t, rr, httpx.CodeInternalUnavailable)
	})
}
