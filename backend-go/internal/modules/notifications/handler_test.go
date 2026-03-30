package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type notificationErrorRepo struct {
	listErr      error
	unreadErr    error
	markReadErr  error
	markAllErr   error
}

func (r *notificationErrorRepo) List(context.Context, string, NotificationListRequest) (PaginatedNotifications, error) {
	return PaginatedNotifications{}, r.listErr
}
func (r *notificationErrorRepo) UnreadCount(context.Context, string) (UnreadCountResult, error) {
	return UnreadCountResult{}, r.unreadErr
}
func (r *notificationErrorRepo) MarkAsRead(context.Context, string, []string) error {
	return r.markReadErr
}
func (r *notificationErrorRepo) MarkAllRead(context.Context, string) error {
	return r.markAllErr
}

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
	if unreadResp.Code != 0 || unreadResp.Data.Total != 0 {
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
	if listResp.Code != 0 || len(listResp.Data.Items) != 0 {
		t.Fatalf("unexpected list response: %#v", listResp)
	}

	markReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_as_read", strings.NewReader(`{"notification_ids":["notif_1"]}`)))
	markRR := httptest.NewRecorder()
	handler.MarkAsRead(markRR, markReq)
	assertNotificationCode(t, markRR, 0)

	markAllReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_all_read", strings.NewReader(`{}`)))
	markAllRR := httptest.NewRecorder()
	handler.MarkAllRead(markAllRR, markAllReq)
	assertNotificationCode(t, markAllRR, 0)
}

func TestNotificationHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&notificationErrorRepo{listErr: errors.New("boom"), unreadErr: errors.New("boom"), markReadErr: errors.New("boom"), markAllErr: errors.New("boom")}))

	listReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{}`)))
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	assertNotificationCode(t, listRR, 10600)

	unreadReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/unread_count", strings.NewReader(`{}`)))
	unreadRR := httptest.NewRecorder()
	handler.UnreadCount(unreadRR, unreadReq)
	assertNotificationCode(t, unreadRR, 10600)

	markReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_as_read", strings.NewReader(`{"notification_ids":["notif_1"]}`)))
	markRR := httptest.NewRecorder()
	handler.MarkAsRead(markRR, markReq)
	assertNotificationCode(t, markRR, 10600)

	markAllReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_all_read", strings.NewReader(`{}`)))
	markAllRR := httptest.NewRecorder()
	handler.MarkAllRead(markAllRR, markAllReq)
	assertNotificationCode(t, markAllRR, 10600)
}

func TestNotificationHandlerValidatesListRequest(t *testing.T) {
	handler := NewHandler(NewService())

	badTypeReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"type":"bad"}`)))
	badTypeRR := httptest.NewRecorder()
	handler.List(badTypeRR, badTypeReq)
	assertNotificationCode(t, badTypeRR, 10000)

	badCursorReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"cursor":"%%%"}}`)))
	badCursorRR := httptest.NewRecorder()
	handler.List(badCursorRR, badCursorReq)
	assertNotificationCode(t, badCursorRR, 10000)
}

func assertNotificationCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
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
