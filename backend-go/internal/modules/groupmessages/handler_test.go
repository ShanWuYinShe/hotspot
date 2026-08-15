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
	"hotspot/backend-go/internal/platform/httpx"
)

// --- stub repo ---

type stubGroupMessageRepo struct {
	sendResult  MessageSentResult
	sendErr     error
	histResult  PaginatedGroupMessages
	histErr     error
	updateMeta  MessageEventMeta
	updateErr   error
	recallMeta  MessageEventMeta
	recallErr   error
	markReadFn  func(context.Context, string, MarkGroupMessagesReadRequest) error
	markAllErr  error
	isMemberErr error
}

func (r *stubGroupMessageRepo) Send(_ context.Context, _ SendGroupMessageRequest) (MessageSentResult, error) {
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
func (r *stubGroupMessageRepo) RecallAsAdmin(_ context.Context, _ string) (MessageEventMeta, error) {
	return r.recallMeta, r.recallErr
}
func (r *stubGroupMessageRepo) MarkAsRead(ctx context.Context, userID string, req MarkGroupMessagesReadRequest) error {
	if r.markReadFn != nil {
		return r.markReadFn(ctx, userID, req)
	}
	return nil
}
func (r *stubGroupMessageRepo) GetMediaForMessage(_ context.Context, _ string, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (r *stubGroupMessageRepo) GetMessageForReply(_ context.Context, _ string) (*MessageBrief, bool, error) {
	return nil, false, nil
}

func (r *stubGroupMessageRepo) IsMember(_ context.Context, _ string, _ string) (bool, error) {
	if r.isMemberErr != nil {
		return false, r.isMemberErr
	}
	return true, nil // 便捷默认：视为成员；非成员场景用 isMemberErr = ErrNotGroupMember
}
func (r *stubGroupMessageRepo) FilterMemberIDs(_ context.Context, _ string, userIDs []string) ([]string, error) {
	// 便捷默认：全部视为成员（与 IsMember 的默认语义一致）
	return userIDs, nil
}
func (r *stubGroupMessageRepo) MarkAllRead(_ context.Context, _ string) ([]string, error) {
	return nil, r.markAllErr
}

func (r *stubGroupMessageRepo) GetReadCount(_ context.Context, _ string, _ string) (int, error) {
	return 0, nil
}

func (r *stubGroupMessageRepo) LatestMessageID(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (r *stubGroupMessageRepo) MemberMuteStatus(_ context.Context, _ string, _ string) (bool, string, bool, error) {
	return false, "", true, nil
}
func (r *stubGroupMessageRepo) MessageGroupID(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}

func (r *stubGroupMessageRepo) MessageExists(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

func (r *stubGroupMessageRepo) GetGroupName(_ context.Context, _ string) (string, bool, error) {
	return "test group", true, nil
}

// LookupMessageMeta defaults to a message owned by "u1" in "g1" so the
// handler-level Update/Recall tests that authorize as u1 succeed without
// extra wiring.
func (r *stubGroupMessageRepo) LookupMessageMeta(_ context.Context, _ string) (MessageOwnerMeta, bool, error) {
	return MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"}, true, nil
}

// --- helpers ---

func gmCtx(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	return req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: userID}))
}

func gmAssertCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	body := rr.Body.String()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %d (body: %s)", expected, resp.Code, body)
	}
}

// --- Send tests ---

func TestHandlerSendRejectsMissingIdempotencyKey(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"text","content":"hi"}`))

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerSendRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{bad`))
	req.Header.Set("Idempotency-Key", "aaaa1111aaaa1111aaaa1111aaaa1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerSendRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"type":"text","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "bbbb1111bbbb1111bbbb1111bbbb1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerSendRejectsMissingType(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "cccc1111cccc1111cccc1111cccc1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerSendHappyPath(t *testing.T) {
	repo := &stubGroupMessageRepo{sendResult: MessageSentResult{MessageID: "m1", SentAt: "2026-01-01T00:00:00Z"}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"text","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "dddd1111dddd1111dddd1111dddd1111")

	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
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
	gmAssertCode(t, rr, httpx.CodeInternalUnavailable)
}

// --- Update tests ---

func TestHandlerUpdateRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{bad`))
	req.Header.Set("Idempotency-Key", "upd_invalid_json_aaaaaaaaaaaaaaaa")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerUpdateRejectsMissingMessageID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"content":"edited"}`))
	req.Header.Set("Idempotency-Key", "upd_no_msg_id_bbbbbbbbbbbbbbbbbbbb")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerUpdateRejectsEmptyContent(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1","content":""}`))
	req.Header.Set("Idempotency-Key", "upd_empty_content_cccccccccccccc")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerUpdateMapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		expected httpx.Code
	}{
		{"not found", ErrMessageNotFound, httpx.CodeNotFound},
		{"permission denied", ErrPermissionDenied, httpx.CodeForbidden},
		{"recalled", ErrMessageRecalled, httpx.CodeConflict},
		{"not editable", ErrMessageNotEditable, httpx.CodeConflict},
		{"internal", errors.New("boom"), httpx.CodeInternalUnavailable},
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
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerRecallRejectsMissingMessageID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{}`))
	req.Header.Set("Idempotency-Key", "rec_no_msg_id_ffffffffffffffffff")

	rr := httptest.NewRecorder()
	handler.Recall(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerRecallMapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		expected httpx.Code
	}{
		{"not found", ErrMessageNotFound, httpx.CodeNotFound},
		{"permission denied", ErrPermissionDenied, httpx.CodeForbidden},
		{"already recalled", ErrMessageRecalled, httpx.CodeConflict},
		{"recall window expired", ErrMessageRecallWindow, httpx.CodeConflict},
		{"internal", errors.New("boom"), httpx.CodeInternalUnavailable},
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
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerHistoryRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerHistoryRejectsInvalidCursor(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","pagination":{"cursor":"not-a-valid-cursor"}}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerHistoryRejectsInvalidLimit(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","pagination":{"limit":999}}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerHistoryReturns10600OnError(t *testing.T) {
	repo := &stubGroupMessageRepo{histErr: errors.New("boom")}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1"}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestHandlerHistoryHappyPath(t *testing.T) {
	repo := &stubGroupMessageRepo{histResult: PaginatedGroupMessages{Items: []GroupMessageItem{{MessageID: "m1"}}}}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1"}`))

	rr := httptest.NewRecorder()
	handler.History(rr, req)
	var resp struct {
		Code httpx.Code             `json:"code"`
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
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerMarkReadRejectsMissingGroupID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"group"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestHandlerMarkReadRejectsUnknownFields(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"group","group_id":"g1","bogus":true}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
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
	gmAssertCode(t, rr, httpx.CodeNotFound)
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
	gmAssertCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestHandlerMarkReadRejectsBadScope(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"scope":"unknown"}`))

	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}
