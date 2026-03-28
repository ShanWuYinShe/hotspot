package groupmessages

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type groupMessageErrorRepo struct {
	sendErr    error
	historyErr error
}

func (r *groupMessageErrorRepo) Send(context.Context, SendGroupMessageRequest) (MessageSent, error) {
	return MessageSent{}, r.sendErr
}
func (r *groupMessageErrorRepo) History(context.Context, string) (PaginatedGroupMessages, error) {
	return PaginatedGroupMessages{}, r.historyErr
}
func (r *groupMessageErrorRepo) Update(context.Context, UpdateGroupMessageRequest) error {
	return nil
}
func (r *groupMessageErrorRepo) Recall(context.Context, RecallGroupMessageRequest) error {
	return nil
}
func (r *groupMessageErrorRepo) MarkAsRead(context.Context, string, string) error {
	return nil
}
func (r *groupMessageErrorRepo) MarkAllRead(context.Context) error {
	return nil
}

func TestSendAndHistoryGroupMessages(t *testing.T) {
	handler := NewHandler(NewService())
	sendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"group_id":"group_1","type":"text","content":"hello"}`))
	sendReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	sendRR := httptest.NewRecorder()
	handler.Send(sendRR, sendReq)

	historyReq := httptest.NewRequest(http.MethodPost, "/get_history", strings.NewReader(`{"group_id":"group_1"}`))
	historyRR := httptest.NewRecorder()
	handler.History(historyRR, historyReq)
	var historyResp struct {
		Code int                    `json:"code"`
		Data PaginatedGroupMessages `json:"data"`
	}
	_ = json.NewDecoder(historyRR.Body).Decode(&historyResp)
	if historyResp.Code != 0 || len(historyResp.Data.Items) != 1 {
		t.Fatalf("unexpected history response: %#v", historyResp)
	}
}

func TestGroupMessagesHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupMessageErrorRepo{sendErr: errors.New("boom"), historyErr: errors.New("boom")}))

	sendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"group_id":"group_1","type":"text","content":"hello"}`))
	sendReq.Header.Set("Idempotency-Key", "efgh1234efgh1234efgh1234efgh1234")
	sendRR := httptest.NewRecorder()
	handler.Send(sendRR, sendReq)
	assertGroupMessageCode(t, sendRR, 10600)

	historyReq := httptest.NewRequest(http.MethodPost, "/get_history", strings.NewReader(`{"group_id":"group_1"}`))
	historyRR := httptest.NewRecorder()
	handler.History(historyRR, historyReq)
	assertGroupMessageCode(t, historyRR, 10600)
}

func assertGroupMessageCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
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
