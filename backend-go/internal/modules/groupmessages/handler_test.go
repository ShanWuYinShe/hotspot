package groupmessages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
