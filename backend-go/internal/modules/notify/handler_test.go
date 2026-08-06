package notify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/types"
)

type notificationErrorRepo struct {
	listErr     error
	unreadErr   error
	markReadErr error
	markAllErr  error
	appendErr   error
}

func (r *notificationErrorRepo) Append(context.Context, string, NotifyItem) error {
	return r.appendErr
}
func (r *notificationErrorRepo) AppendMany(context.Context, []string, NotifyItem) error {
	return r.appendErr
}
func (r *notificationErrorRepo) List(context.Context, string, NotificationListRequest) (PaginatedNotify, error) {
	return PaginatedNotify{}, r.listErr
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
	unreadReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/get_unread_count", strings.NewReader(`{}`)))
	unreadRR := httptest.NewRecorder()
	handler.UnreadCount(unreadRR, unreadReq)

	var unreadResp struct {
		Code httpx.Code        `json:"code"`
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
		Code httpx.Code      `json:"code"`
		Data PaginatedNotify `json:"data"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResp.Code != 0 || len(listResp.Data.Items) != 0 {
		t.Fatalf("unexpected list response: %#v", listResp)
	}

	markReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"notify_ids":["notif_1"]}`)))
	markRR := httptest.NewRecorder()
	handler.MarkRead(markRR, markReq)
	assertNotificationCode(t, markRR, 0)

	markAllReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{}`)))
	markAllRR := httptest.NewRecorder()
	handler.MarkRead(markAllRR, markAllReq)
	assertNotificationCode(t, markAllRR, 0)
}

func TestNotificationHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&notificationErrorRepo{listErr: errors.New("boom"), unreadErr: errors.New("boom"), markReadErr: errors.New("boom"), markAllErr: errors.New("boom")}))

	listReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{}`)))
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	assertNotificationCode(t, listRR, httpx.CodeInternalUnavailable)

	unreadReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/get_unread_count", strings.NewReader(`{}`)))
	unreadRR := httptest.NewRecorder()
	handler.UnreadCount(unreadRR, unreadReq)
	assertNotificationCode(t, unreadRR, httpx.CodeInternalUnavailable)

	markReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"notify_ids":["notif_1"]}`)))
	markRR := httptest.NewRecorder()
	handler.MarkRead(markRR, markReq)
	assertNotificationCode(t, markRR, httpx.CodeInternalUnavailable)

	markAllReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{}`)))
	markAllRR := httptest.NewRecorder()
	handler.MarkRead(markAllRR, markAllReq)
	assertNotificationCode(t, markAllRR, httpx.CodeInternalUnavailable)
}

func TestNotificationHandlerValidatesListRequest(t *testing.T) {
	handler := NewHandler(NewService())

	badTypeReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"type":"bad"}`)))
	badTypeRR := httptest.NewRecorder()
	handler.List(badTypeRR, badTypeReq)
	assertNotificationCode(t, badTypeRR, httpx.CodeInvalidRequest)

	badCursorReq := withPrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"cursor":"%%%"}}`)))
	badCursorRR := httptest.NewRecorder()
	handler.List(badCursorRR, badCursorReq)
	assertNotificationCode(t, badCursorRR, httpx.CodeInvalidRequest)
}

func TestUnreadCountReturnsByTypeBreakdown(t *testing.T) {
	svc := NewService()
	handler := NewHandler(svc)

	// Insert notifications of each type category
	_, _ = svc.AppendNewFollower(context.Background(), "user_alice", types.UserInfo{UserID: "bob", Nickname: "Bob"}, "", "")
	_, _ = svc.AppendNewFollower(context.Background(), "user_alice", types.UserInfo{UserID: "charlie", Nickname: "Charlie"}, "", "")
	_, _ = svc.AppendReactionAdded(context.Background(), "user_alice", types.UserInfo{UserID: "bob", Nickname: "Bob"}, "moment", "m1", "👍")
	_, _ = svc.AppendGroupInviteReceived(context.Background(), "user_alice", types.UserInfo{UserID: "bob", Nickname: "Bob"}, "grp1", "Group One", "inv1", nil)
	_, _ = svc.AppendSystemAnnouncement(context.Background(), []string{"user_alice"}, systemmodule.SystemAnnouncementItem{AnnouncementID: "ann1", Title: "Test", Content: "Test content", Severity: systemmodule.AnnounceLevelInfo})

	req := withPrincipal(httptest.NewRequest(http.MethodPost, "/get_unread_count", strings.NewReader(`{}`)))
	rr := httptest.NewRecorder()
	handler.UnreadCount(rr, req)

	var resp struct {
		Code httpx.Code        `json:"code"`
		Data UnreadCountResult `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %#v", resp)
	}
	if resp.Data.Total != 5 {
		t.Fatalf("expected total 5, got %d", resp.Data.Total)
	}
	if resp.Data.ByType.Social != 2 {
		t.Fatalf("expected by_type.social=2, got %d", resp.Data.ByType.Social)
	}
	if resp.Data.ByType.Interaction != 1 {
		t.Fatalf("expected by_type.interaction=1, got %d", resp.Data.ByType.Interaction)
	}
	if resp.Data.ByType.Group != 1 {
		t.Fatalf("expected by_type.group=1, got %d", resp.Data.ByType.Group)
	}
	if resp.Data.ByType.System != 1 {
		t.Fatalf("expected by_type.system=1, got %d", resp.Data.ByType.System)
	}
}

func TestMarkReadRejectsEmptyArray(t *testing.T) {
	handler := NewHandler(NewService())

	// scope=selected with empty notification_ids and no type is invalid (neither provided)
	req := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"scope":"selected","notification_ids":[]}`)))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	assertNotificationCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMarkReadRejectsTooManyIDs(t *testing.T) {
	handler := NewHandler(NewService())

	var b strings.Builder
	b.WriteString(`{"scope":"selected","notification_ids":[`)
	for i := 0; i < 51; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`"id_`)
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`"`)
	}
	b.WriteString(`]}`)
	req := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(b.String())))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	assertNotificationCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMarkReadRejectsBothIDsAndType(t *testing.T) {
	handler := NewHandler(NewService())
	req := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"scope":"selected","notification_ids":["n1"],"type":"new_follower"}`)))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	assertNotificationCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMarkReadRejectsBadScope(t *testing.T) {
	handler := NewHandler(NewService())
	req := withPrincipal(httptest.NewRequest(http.MethodPost, "/mark_read", strings.NewReader(`{"scope":"unknown","notification_ids":["n1"]}`)))
	rr := httptest.NewRecorder()
	handler.MarkRead(rr, req)
	assertNotificationCode(t, rr, httpx.CodeInvalidRequest)
}

func assertNotificationCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
