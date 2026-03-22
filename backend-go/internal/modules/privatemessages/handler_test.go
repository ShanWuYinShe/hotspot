package private_messages

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

type privateMessageErrorRepo struct {
	sendErr    error
	historyErr error
	recentErr  error
	lastErr    error
	updateErr  error
	recallErr  error
	markErr    error
	markAllErr error
}

func (r *privateMessageErrorRepo) Send(context.Context, string, string, SendPrivateMessageRequest) (MessageSent, error) {
	return MessageSent{}, r.sendErr
}
func (r *privateMessageErrorRepo) History(context.Context, string, string) (PaginatedMessages, error) {
	return PaginatedMessages{}, r.historyErr
}
func (r *privateMessageErrorRepo) Recent(context.Context, string, string, int) (PaginatedMessages, error) {
	return PaginatedMessages{}, r.recentErr
}
func (r *privateMessageErrorRepo) Last(context.Context, string, string) (*PrivateMessageItem, error) {
	return nil, r.lastErr
}
func (r *privateMessageErrorRepo) Update(context.Context, string, UpdatePrivateMessageRequest) error {
	return r.updateErr
}
func (r *privateMessageErrorRepo) Recall(context.Context, string, RecallPrivateMessageRequest) error {
	return r.recallErr
}
func (r *privateMessageErrorRepo) MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) error {
	return r.markErr
}
func (r *privateMessageErrorRepo) MarkAllRead(context.Context, string) error {
	return r.markAllErr
}

func withPrivateMessagePrincipal(req *http.Request) *http.Request {
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	return req.WithContext(ctx)
}

func TestSendAndHistoryPrivateMessages(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewService())

	sendReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
	sendReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	sendRR := httptest.NewRecorder()
	handler.Send(sendRR, sendReq)

	assertPrivateMessageCode(t, sendRR, 0)

	historyReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/get_history", strings.NewReader(`{"target_user_id":"user_bob"}`)))
	historyRR := httptest.NewRecorder()
	handler.GetHistory(historyRR, historyReq)

	var resp struct {
		Code int               `json:"code"`
		Data PaginatedMessages `json:"data"`
	}
	if err := json.NewDecoder(historyRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) != 1 || resp.Data.Items[0].Content != "hello" {
		t.Fatalf("unexpected history response: %#v", resp)
	}
}

func TestPrivateMessagesHandlerMapsErrors(t *testing.T) {
	t.Run("send and history generic errors", func(t *testing.T) {
		handler := NewHandler(NewServiceWithRepo(&privateMessageErrorRepo{sendErr: errors.New("boom"), historyErr: errors.New("boom"), markAllErr: errors.New("boom")}))

		sendReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
		sendReq.Header.Set("Idempotency-Key", "efgh1234efgh1234efgh1234efgh1234")
		sendRR := httptest.NewRecorder()
		handler.Send(sendRR, sendReq)
		assertPrivateMessageCode(t, sendRR, 10600)

		historyReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/get_history", strings.NewReader(`{"target_user_id":"user_bob"}`)))
		historyRR := httptest.NewRecorder()
		handler.GetHistory(historyRR, historyReq)
		assertPrivateMessageCode(t, historyRR, 10600)

		markAllReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_all_read", strings.NewReader(`{}`)))
		markAllRR := httptest.NewRecorder()
		handler.MarkAllRead(markAllRR, markAllReq)
		assertPrivateMessageCode(t, markAllRR, 10600)
	})

	t.Run("update maps domain errors", func(t *testing.T) {
		cases := []struct {
			name string
			err  error
			code int
		}{
			{name: "not found", err: ErrMessageNotFound, code: 10300},
			{name: "permission", err: ErrMessagePermissionDenied, code: 10200},
			{name: "not editable", err: ErrMessageNotEditable, code: 10400},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				handler := NewHandler(NewServiceWithRepo(&privateMessageErrorRepo{updateErr: tc.err}))
				req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"message_id":"pm_1","content":"updated"}`)))
				rr := httptest.NewRecorder()
				handler.Update(rr, req)
				assertPrivateMessageCode(t, rr, tc.code)
			})
		}
	})

	t.Run("recall maps domain errors", func(t *testing.T) {
		cases := []struct {
			name string
			err  error
			code int
		}{
			{name: "not found", err: ErrMessageNotFound, code: 10300},
			{name: "permission", err: ErrMessagePermissionDenied, code: 10200},
			{name: "not recallable", err: ErrMessageNotRecallable, code: 10400},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				handler := NewHandler(NewServiceWithRepo(&privateMessageErrorRepo{recallErr: tc.err}))
				req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"message_id":"pm_1"}`)))
				rr := httptest.NewRecorder()
				handler.Recall(rr, req)
				assertPrivateMessageCode(t, rr, tc.code)
			})
		}
	})

	t.Run("mark as read maps not found", func(t *testing.T) {
		handler := NewHandler(NewServiceWithRepo(&privateMessageErrorRepo{markErr: ErrMessageNotFound}))
		req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_as_read", strings.NewReader(`{"target_user_id":"user_bob","up_to_message_id":"pm_1"}`)))
		rr := httptest.NewRecorder()
		handler.MarkAsRead(rr, req)
		assertPrivateMessageCode(t, rr, 10300)
	})
}

func assertPrivateMessageCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
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
