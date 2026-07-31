package groupmessages

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// --- stub repo ---

type stubGroupMessageRepo struct {
	sendResult  MessageSent
	sendErr     error
	histResult  PaginatedGroupMessages
	histErr     error
	updateMeta  MessageEventMeta
	updateErr   error
	recallMeta  MessageEventMeta
	recallErr   error
	markReadFn  func(context.Context, string, MarkGroupMessagesReadRequest) error
	markAllErr  error
	isMemberVal bool
	isMemberErr error
}

func (r *stubGroupMessageRepo) Send(_ context.Context, _ SendGroupMessageRequest) (MessageSent, error) {
	return r.sendResult, r.sendErr
}
func (r *stubGroupMessageRepo) History(_ context.Context, _ string, _ *PaginationParams) (PaginatedGroupMessages, error) {
	return r.histResult, r.histErr
}
func (r *stubGroupMessageRepo) Update(_ context.Context, _ UpdateGroupMessageRequest) (MessageEventMeta, error) {
	return r.updateMeta, r.updateErr
}
func (r *stubGroupMessageRepo) Recall(_ context.Context, _ RecallGroupMessageRequest) (MessageEventMeta, error) {
	return r.recallMeta, r.recallErr
}
func (r *stubGroupMessageRepo) MarkAsRead(ctx context.Context, userID string, req MarkGroupMessagesReadRequest) error {
	if r.markReadFn != nil {
		return r.markReadFn(ctx, userID, req)
	}
	return nil
}
func (r *stubGroupMessageRepo) GetMediaForMessage(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (r *stubGroupMessageRepo) GetMessageForReply(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (r *stubGroupMessageRepo) IsMember(_ context.Context, _ string, _ string) (bool, error) {
	if !r.isMemberVal && r.isMemberErr == nil {
		return true, nil
	}
	return r.isMemberVal, r.isMemberErr
}
func (r *stubGroupMessageRepo) MarkAllRead(_ context.Context, _ string) error {
	return r.markAllErr
}

func (r *stubGroupMessageRepo) MemberMuteStatus(_ context.Context, _ string, _ string) (bool, string, bool, error) {
	return false, "", true, nil
}
func (r *stubGroupMessageRepo) MessageExists(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

// --- helpers ---

func gmCtx(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	return req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: userID}))
}

func gmAssertCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %d (body: %s)", expected, resp.Code, rr.Body.String())
	}
}

// --- Send tests ---

func TestHandlerSendRejectsMissingIdempotencyKey(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"text","content":"hi"}`))

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerSendRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{bad`))
	req.Header.Set("Idempotency-Key", "aaaa1111aaaa1111aaaa1111aaaa1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerSendRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"type":"text","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "bbbb1111bbbb1111bbbb1111bbbb1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerSendRejectsMissingType(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "cccc1111cccc1111cccc1111cccc1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerSendHappyPath(t *testing.T) {
	repo := &stubGroupMessageRepo{sendResult: MessageSent{MessageID: "m1", SentAt: "2026-01-01T00:00:00Z"}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"text","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "dddd1111dddd1111dddd1111dddd1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != 0 || resp.Data.MessageID != "m1" {
		t.Fatalf("unexpected: %+v", resp)
	}
}

func TestHandlerSendReturns10600OnError(t *testing.T) {
	repo := &stubGroupMessageRepo{sendErr: errors.New("boom")}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"text","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "eeee1111eeee1111eeee1111eeee1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, 10600)
}

// --- Update tests ---

func TestHandlerUpdateRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{bad`))
	req.Header.Set("Idempotency-Key", "upd_invalid_json_aaaaaaaaaaaaaaaa")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerUpdateRejectsMissingMessageID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"content":"edited"}`))
	req.Header.Set("Idempotency-Key", "upd_no_msg_id_bbbbbbbbbbbbbbbbbbbb")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerUpdateRejectsEmptyContent(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1","content":""}`))
	req.Header.Set("Idempotency-Key", "upd_empty_content_cccccccccccccc")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerUpdateMapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		expected int
	}{
		{"not found", ErrMessageNotFound, 10300},
		{"permission denied", ErrPermissionDenied, 10200},
		{"recalled", ErrMessageRecalled, 10400},
		{"not editable", ErrMessageNotEditable, 10400},
		{"internal", errors.New("boom"), 10600},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubGroupMessageRepo{updateErr: tc.repoErr}
			handler := NewHandler(NewServiceWithRepo(repo))
			req := gmCtx("u1")
			req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1","content":"edited"}`))
			req.Header.Set("Idempotency-Key", "test_update_maps_"+strings.ReplaceAll(tc.name, " ", "_")+"_000000000000")

			rr := httptest.NewRecorder()
			handler.Update(rr, req)
			gmAssertCode(t, rr, tc.expected)
		})
	}
}

func TestHandlerUpdateHappyPath(t *testing.T) {
	repo := &stubGroupMessageRepo{updateMeta: MessageEventMeta{GroupID: "g1", MessageID: "m1"}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1","content":"edited"}`))
	req.Header.Set("Idempotency-Key", "upd_happy_path_ddddddddddddddddd")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, 0)
}

// --- Recall tests ---

func TestHandlerRecallRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{bad`))
	req.Header.Set("Idempotency-Key", "rec_invalid_json_eeeeeeeeeeeeeeee")

	rr := httptest.NewRecorder()
	handler.Recall(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerRecallRejectsMissingMessageID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{}`))
	req.Header.Set("Idempotency-Key", "rec_no_msg_id_ffffffffffffffffff")

	rr := httptest.NewRecorder()
	handler.Recall(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerRecallMapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		expected int
	}{
		{"not found", ErrMessageNotFound, 10300},
		{"permission denied", ErrPermissionDenied, 10200},
		{"already recalled", ErrMessageRecalled, 10400},
		{"recall window expired", ErrMessageRecallWindow, 10400},
		{"internal", errors.New("boom"), 10600},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubGroupMessageRepo{recallErr: tc.repoErr}
			handler := NewHandler(NewServiceWithRepo(repo))
			req := gmCtx("u1")
			req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1"}`))
			req.Header.Set("Idempotency-Key", "test_recall_maps_"+strings.ReplaceAll(tc.name, " ", "_")+"_000000000000")

			rr := httptest.NewRecorder()
			handler.Recall(rr, req)
			gmAssertCode(t, rr, tc.expected)
		})
	}
}

func TestHandlerRecallHappyPath(t *testing.T) {
	repo := &stubGroupMessageRepo{recallMeta: MessageEventMeta{GroupID: "g1", MessageID: "m1"}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1"}`))
	req.Header.Set("Idempotency-Key", "rec_happy_path_ggggggggggggggggg")

	rr := httptest.NewRecorder()
	handler.Recall(rr, req)
	gmAssertCode(t, rr, 0)
}

// --- History tests ---

func TestHandlerHistoryRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{bad`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerHistoryRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerHistoryRejectsInvalidCursor(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","pagination":{"cursor":"not-a-valid-cursor"}}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerHistoryRejectsInvalidLimit(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","pagination":{"limit":999}}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerHistoryReturns10600OnError(t *testing.T) {
	repo := &stubGroupMessageRepo{histErr: errors.New("boom")}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1"}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, 10600)
}

func TestHandlerHistoryHappyPath(t *testing.T) {
	repo := &stubGroupMessageRepo{histResult: PaginatedGroupMessages{Items: []GroupMessageItem{{MessageID: "m1"}}}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1"}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	var resp struct {
		Code int                    `json:"code"`
		Data PaginatedGroupMessages `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != 0 || len(resp.Data.Items) != 1 {
		t.Fatalf("unexpected: %+v", resp)
	}
}

// --- MarkRead (group scope) tests ---

func TestHandlerMarkReadRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{bad`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerMarkReadRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"group"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerMarkReadRejectsUnknownFields(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"group","group_id":"g1","bogus":true}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 10000)
}

func TestHandlerMarkReadHappyPath(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"group","group_id":"g1"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 0)
}

func TestHandlerMarkReadMapsNotFound(t *testing.T) {
	repo := &stubGroupMessageRepo{markReadFn: func(_ context.Context, _ string, _ MarkGroupMessagesReadRequest) error {
		return ErrMessageNotFound
	}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"group","group_id":"g1"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 10300)
}

// --- MarkRead (all scope) tests ---

func TestHandlerMarkAllReadHappyPath(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"all"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 0)
}

func TestHandlerMarkAllReadReturns10600OnError(t *testing.T) {
	repo := &stubGroupMessageRepo{markAllErr: errors.New("boom")}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"all"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 10600)
}

func TestHandlerMarkReadRejectsBadScope(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"unknown"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, 10000)
}
