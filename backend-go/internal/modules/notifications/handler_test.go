package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func withPrincipal(req *http.Request) *http.Request {
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	return req.WithContext(ctx)
}

func TestUnreadCountAndMarkRead(t *testing.T) {
	handler := NewHandler(NewService())
	unreadReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/unread_count", strings.NewReader(`{}`)))
	unreadRR := httptest.NewRecorder()
	handler.UnreadCount(unreadRR, unreadReq)

	var unreadResp struct {
		Code int               `json:"code"`
		Data UnreadCountResult `json:"data"`
	}
	if err := json.NewDecoder(unreadRR.Body).Decode(&unreadResp); err != nil {
		t.Fatalf("decode unread response: %v", err)
	}
	if unreadResp.Code != 0 || unreadResp.Data.Total == 0 {
		t.Fatalf("unexpected unread response: %#v", unreadResp)
	}

	listReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{}`)))
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	var listResp struct {
		Code int                    `json:"code"`
		Data PaginatedNotifications `json:"data"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	markReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_as_read", strings.NewReader(`{"notification_ids":["`+listResp.Data.Items[0].NotificationID+`"]}`)))
	markRR := httptest.NewRecorder()
	handler.MarkAsRead(markRR, markReq)

	unreadReq2 := withPrincipal(httptest.NewRequest(http.MethodPost, "/unread_count", strings.NewReader(`{}`)))
	unreadRR2 := httptest.NewRecorder()
	handler.UnreadCount(unreadRR2, unreadReq2)
	var unreadResp2 struct {
		Code int               `json:"code"`
		Data UnreadCountResult `json:"data"`
	}
	if err := json.NewDecoder(unreadRR2.Body).Decode(&unreadResp2); err != nil {
		t.Fatalf("decode unread response 2: %v", err)
	}
	if unreadResp2.Data.Total != 0 {
		t.Fatalf("expected zero unread, got %#v", unreadResp2)
	}
}
