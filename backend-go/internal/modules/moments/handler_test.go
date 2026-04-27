package moments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/notifications"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type failingMomentCounter struct {
	repo    *InMemoryRepository
	failErr error
}

func (c *failingMomentCounter) Exists(ctx context.Context, momentID string) (bool, error) {
	return c.repo.Exists(ctx, momentID)
}

func (c *failingMomentCounter) CheckCommentable(ctx context.Context, momentID string) (bool, bool, error) {
	_, ok, err := c.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return ok, false, err
	}
	return true, true, nil
}

func (c *failingMomentCounter) IncrementCommentCount(ctx context.Context, momentID string) error {
	if c.failErr != nil {
		return c.failErr
	}
	return c.repo.IncrementCommentCount(ctx, momentID)
}

func (c *failingMomentCounter) DecrementCommentCount(ctx context.Context, momentID string) error {
	return c.repo.DecrementCommentCount(ctx, momentID)
}

func (c *failingMomentCounter) NotificationContext(ctx context.Context, momentID string) (comments.MomentNotificationContext, bool, error) {
	item, ok, err := c.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return comments.MomentNotificationContext{}, ok, err
	}
	return comments.MomentNotificationContext{UserID: item.User.UserID, Summary: map[string]any{"moment_id": item.MomentID, "description_preview": item.Description}}, true, nil
}

type sinkCapture struct{ events []realtime.Event }

func (s *sinkCapture) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkCapture) Close() error                { return nil }

func TestCreateOpenAndListMoment(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hello","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"}}`))
	createReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	createReq2 := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hello","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"}}`))
	createReq2.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	createReq2 = createReq2.WithContext(ctx)
	createRR2 := httptest.NewRecorder()
	handler.Create(createRR2, createReq2)

	var createResp struct {
		Code int        `json:"code"`
		Data MomentItem `json:"data"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	var createResp2 struct {
		Code int        `json:"code"`
		Data MomentItem `json:"data"`
	}
	_ = json.NewDecoder(createRR2.Body).Decode(&createResp2)
	if createResp.Code != 0 || createResp.Data.MomentID == "" || createResp.Data.MomentID != createResp2.Data.MomentID {
		t.Fatalf("unexpected create response: %#v %#v", createResp, createResp2)
	}

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"moment_id":"`+createResp.Data.MomentID+`"}`))
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)

	listReq := httptest.NewRequest(http.MethodPost, "/me/list", strings.NewReader(`{}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	handler.ListMine(listRR, listReq)
	var listResp struct {
		Code int              `json:"code"`
		Data PaginatedMoments `json:"data"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Data.Items) != 1 {
		t.Fatalf("unexpected list response: %#v", listResp)
	}
}

func TestCreateCommentIncrementsMomentCount(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	momentRepo := NewInMemoryRepository()
	momentService := NewServiceWithRepo(momentRepo)
	commentService := comments.NewService(comments.NewInMemoryRepository(), momentService)
	momentHandler := NewHandler(momentService)
	commentHandler := comments.NewHandler(commentService)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createMomentReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hello","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"}}`))
	createMomentReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	createMomentReq = createMomentReq.WithContext(ctx)
	createMomentRR := httptest.NewRecorder()
	momentHandler.Create(createMomentRR, createMomentReq)
	var momentResp struct {
		Code int        `json:"code"`
		Data MomentItem `json:"data"`
	}
	_ = json.NewDecoder(createMomentRR.Body).Decode(&momentResp)

	createCommentReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"moment_id":"`+momentResp.Data.MomentID+`","content":"nice"}`))
	createCommentReq.Header.Set("Idempotency-Key", "zzzz1234zzzz1234zzzz1234zzzz1234")
	createCommentReq = createCommentReq.WithContext(ctx)
	createCommentRR := httptest.NewRecorder()
	commentHandler.Create(createCommentRR, createCommentReq)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"moment_id":"`+momentResp.Data.MomentID+`"}`))
	openRR := httptest.NewRecorder()
	momentHandler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code int        `json:"code"`
		Data MomentItem `json:"data"`
	}
	_ = json.NewDecoder(openRR.Body).Decode(&openResp)
	if openResp.Data.CommentsCount != 1 {
		t.Fatalf("expected comments_count=1, got %#v", openResp)
	}
}

func TestCreateCommentDoesNotIncrementMomentCountWhenCounterFails(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	momentRepo := NewInMemoryRepository()
	momentService := NewServiceWithRepo(momentRepo)
	counter := &failingMomentCounter{repo: momentRepo, failErr: errors.New("update failed")}
	commentRepo := comments.NewInMemoryRepository()
	commentService := comments.NewService(commentRepo, counter)
	commentHandler := comments.NewHandler(commentService)
	momentHandler := NewHandler(momentService)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createdMoment, err := momentRepo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	createCommentReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"moment_id":"`+createdMoment.MomentID+`","content":"nice"}`))
	createCommentReq.Header.Set("Idempotency-Key", "yyyy1234yyyy1234yyyy1234yyyy1234")
	createCommentReq = createCommentReq.WithContext(ctx)
	createCommentRR := httptest.NewRecorder()
	commentHandler.Create(createCommentRR, createCommentReq)
	var createCommentResp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(createCommentRR.Body).Decode(&createCommentResp); err != nil {
		t.Fatalf("decode create comment response: %v", err)
	}
	if createCommentResp.Code != 10600 {
		t.Fatalf("expected code 10600, got %#v", createCommentResp)
	}
	commentsList, err := commentRepo.List(context.Background(), createdMoment.MomentID, 20, "")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(commentsList.Items) != 0 {
		t.Fatalf("comment should be rolled back, got %#v", commentsList)
	}
	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"moment_id":"`+createdMoment.MomentID+`"}`))
	openRR := httptest.NewRecorder()
	momentHandler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code int        `json:"code"`
		Data MomentItem `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Data.CommentsCount != 0 {
		t.Fatalf("expected comments_count=0, got %#v", openResp)
	}
}

func TestCreateCommentReturns10300WhenMomentMissing(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	momentRepo := NewServiceWithRepo(NewInMemoryRepository())
	commentService := comments.NewService(comments.NewInMemoryRepository(), momentRepo)
	commentHandler := comments.NewHandler(commentService)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createCommentReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"moment_id":"moment_missing","content":"nice"}`))
	createCommentReq.Header.Set("Idempotency-Key", "xxxx1234xxxx1234xxxx1234xxxx1234")
	createCommentReq = createCommentReq.WithContext(ctx)
	createCommentRR := httptest.NewRecorder()
	commentHandler.Create(createCommentRR, createCommentReq)
	assertCommentCode(t, createCommentRR, 10300)
}

func TestListCommentsReturns10300WhenMomentMissing(t *testing.T) {
	commentService := comments.NewService(comments.NewInMemoryRepository(), NewServiceWithRepo(NewInMemoryRepository()))
	commentHandler := comments.NewHandler(commentService)

	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"moment_id":"moment_missing"}`))
	listRR := httptest.NewRecorder()
	commentHandler.List(listRR, listReq)
	assertCommentCode(t, listRR, 10300)
}

func TestCreateMomentRecordsResourceGuard(t *testing.T) {
	checker := &momentRecordingChecker{}
	resourceguard.SetDefaultChecker(checker)
	defer resourceguard.SetDefaultChecker(resourceguard.NoopChecker{})

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	service := NewServiceWithRepo(NewInMemoryRepository())
	if _, err := service.Create(ctx, CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}}); err != nil {
		t.Fatalf("create moment: %v", err)
	}
	if len(checker.recorded) != 1 || !strings.HasPrefix(checker.recorded[0], "moment:") {
		t.Fatalf("unexpected records: %#v", checker.recorded)
	}
}

func TestLikeMomentSendsRealtimeAndNotificationToAuthor(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo), nil)
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	ctxBob := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_2"})
	created, err := service.Create(ctxAlice, CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if err := service.Like(ctxBob, created.MomentID); err != nil {
		t.Fatalf("like moment: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected one event, got %#v", aliceSink.events)
	}
	if aliceSink.events[0].Type != "MomentLiked" {
		t.Fatalf("unexpected event type: %#v", aliceSink.events)
	}
	payload, ok := aliceSink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", aliceSink.events[0].Payload)
	}
	if payload["moment_id"] != created.MomentID {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "moment_liked" {
		t.Fatalf("unexpected notifications page: %#v", notificationsPage)
	}
}

func TestLikeMomentSkipsSelfNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo), nil)
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	created, err := service.Create(ctxAlice, CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if err := service.Like(ctxAlice, created.MomentID); err != nil {
		t.Fatalf("like own moment: %v", err)
	}
	if len(aliceSink.events) != 0 {
		t.Fatalf("expected no self event, got %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 0 {
		t.Fatalf("expected no self notifications, got %#v", notificationsPage)
	}
}

func TestLikeMomentDeduplicatesRepeatedLike(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo), nil)
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	ctxBob := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_2"})
	created, err := service.Create(ctxAlice, CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if err := service.Like(ctxBob, created.MomentID); err != nil {
		t.Fatalf("first like: %v", err)
	}
	if err := service.Like(ctxBob, created.MomentID); err != nil {
		t.Fatalf("second like: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected one event, got %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "moment_liked" {
		t.Fatalf("expected one moment_liked notification, got %#v", notificationsPage)
	}
}

func TestCreateMomentSendsMentionNotificationsAndRealtime(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo), nil)
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	bobSink := &sinkCapture{}
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	created, err := service.Create(ctxAlice, CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}, Mentions: []string{"user_bob", "user_bob", "user_alice"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	if len(created.Mentions) != 2 || created.Mentions[0] != "user_bob" || created.Mentions[1] != "user_alice" {
		t.Fatalf("unexpected mentions: %#v", created.Mentions)
	}
	if len(bobSink.events) != 1 || bobSink.events[0].Type != "MessageMentioned" {
		t.Fatalf("unexpected realtime events: %#v", bobSink.events)
	}
	payload, ok := bobSink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", bobSink.events[0].Payload)
	}
	if payload["target_type"] != "moment" || payload["moment_id"] != created.MomentID {
		t.Fatalf("unexpected realtime payload: %#v", payload)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_bob", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "mentioned" {
		t.Fatalf("unexpected notifications page: %#v", notificationsPage)
	}
}

func TestUpdateMomentReplacesMentionNotificationsAndState(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo), nil)
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	bobSink := &sinkCapture{}
	carolSink := &sinkCapture{}
	hub.RegisterUser("user_bob", "sess_bob", bobSink)
	hub.RegisterUser("user_carol", "sess_carol", carolSink)

	created, err := service.Create(ctxAlice, CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}, Mentions: []string{"user_bob"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	updated, err := service.Update(ctxAlice, EditMomentRequest{MomentID: created.MomentID, Mentions: []string{"user_carol"}})
	if err != nil {
		t.Fatalf("update moment: %v", err)
	}
	if len(updated.Mentions) != 1 || updated.Mentions[0] != "user_carol" {
		t.Fatalf("unexpected updated mentions: %#v", updated.Mentions)
	}
	if len(carolSink.events) != 1 || carolSink.events[0].Type != "MessageMentioned" {
		t.Fatalf("unexpected carol events: %#v", carolSink.events)
	}
	carolNotifications, err := notificationRepo.List(context.Background(), "user_carol", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list carol notifications: %v", err)
	}
	if len(carolNotifications.Items) != 1 || carolNotifications.Items[0].Type != "mentioned" {
		t.Fatalf("unexpected carol notifications: %#v", carolNotifications)
	}
}

func TestCreateMomentWithoutMentionsDoesNotBroadcastMentionEvent(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	sink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", sink)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	service := NewServiceWithRepo(NewInMemoryRepository())

	if _, err := service.Create(ctx, CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}}); err != nil {
		t.Fatalf("create moment: %v", err)
	}
	if len(sink.events) != 0 {
		t.Fatalf("expected no realtime events, got %#v", sink.events)
	}
}

func TestDeleteMomentReturns10200WhenPermissionDenied(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	_ = platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	ctxOther := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_2"})

	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"moment_id":"`+created.MomentID+`"}`))
	deleteReq = deleteReq.WithContext(ctxOther)
	deleteRR := httptest.NewRecorder()
	handler.Delete(deleteRR, deleteReq)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(deleteRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10200 {
		t.Fatalf("expected code 10200, got code=%d message=%s", resp.Code, resp.Message)
	}
	if resp.Message != "权限不足" {
		t.Fatalf("expected message '权限不足', got %s", resp.Message)
	}
}

func assertCommentCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
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

type momentRecordingChecker struct {
	recorded []string
}

func (c *momentRecordingChecker) Ready(context.Context) bool { return true }
func (c *momentRecordingChecker) MayExist(context.Context, resourceguard.ResourceKind, string) (bool, error) {
	return true, nil
}
func (c *momentRecordingChecker) Record(_ context.Context, kind resourceguard.ResourceKind, id string) error {
	c.recorded = append(c.recorded, string(kind)+":"+id)
	return nil
}

func TestSearchRejectsInvalidSortBy(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	for _, invalid := range []string{"random", "RELEVANCE", "distancee", "new", "popularity"} {
		t.Run(invalid, func(t *testing.T) {
			body := `{"keyword":"test","sort_by":"` + invalid + `","reference_point":{"latitude":0,"longitude":0}}`
			req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(body))
			rr := httptest.NewRecorder()
			handler.Search(rr, req)
			var resp struct {
				Code int `json:"code"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Code != 10000 {
				t.Fatalf("expected code 10000 for sort_by=%q, got %d", invalid, resp.Code)
			}
		})
	}
}

func TestSearchAcceptsValidSortBy(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	for _, valid := range []string{"relevance", "distance", "newest", "popular", ""} {
		t.Run("sort_by="+valid, func(t *testing.T) {
			sortField := ""
			if valid != "" {
				sortField = `,"sort_by":"` + valid + `"`
			}
			body := `{"keyword":"test"` + sortField + `,"reference_point":{"latitude":0,"longitude":0}}`
			req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(body))
			rr := httptest.NewRecorder()
			handler.Search(rr, req)
			var resp struct {
				Code int `json:"code"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Code != 0 {
				t.Fatalf("expected code 0 for sort_by=%q, got %d", valid, resp.Code)
			}
		})
	}
}

func TestSearchSortByNewest(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_test", SessionID: "s1"})

	// Create moments with known timestamps by inserting directly
	m1 := MomentItem{
		MomentID: "moment_a", Description: "alpha test", PrivacyLevel: "public",
		CreatedAt: "2025-01-01T00:00:00Z",
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m2 := MomentItem{
		MomentID: "moment_b", Description: "beta test", PrivacyLevel: "public",
		CreatedAt: "2025-06-01T00:00:00Z",
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m3 := MomentItem{
		MomentID: "moment_c", Description: "gamma test", PrivacyLevel: "public",
		CreatedAt: "2025-03-01T00:00:00Z",
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	repo.moments[m1.MomentID] = m1
	repo.moments[m2.MomentID] = m2
	repo.moments[m3.MomentID] = m3

	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(
		`{"keyword":"test","sort_by":"newest","reference_point":{"latitude":0,"longitude":0}}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Search(rr, req)
	var resp struct {
		Code int              `json:"code"`
		Data PaginatedMoments `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Data.Items))
	}
	// newest first: b (June), c (March), a (January)
	if resp.Data.Items[0].MomentID != "moment_b" || resp.Data.Items[1].MomentID != "moment_c" || resp.Data.Items[2].MomentID != "moment_a" {
		var order []string
		for _, it := range resp.Data.Items {
			order = append(order, it.MomentID)
		}
		t.Fatalf("expected [moment_b, moment_c, moment_a], got %v", order)
	}
}

func TestSearchSortByDistance(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_test", SessionID: "s1"})

	m1 := MomentItem{
		MomentID: "moment_far", Description: "far away", PrivacyLevel: "public",
		CreatedAt: "2025-01-01T00:00:00Z",
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 40.0, Longitude: -74.0}, LocationName: "NYC"},
		User: UserInfo{UserID: "user_test"},
	}
	m2 := MomentItem{
		MomentID: "moment_near", Description: "nearby", PrivacyLevel: "public",
		CreatedAt: "2025-06-01T00:00:00Z",
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 39.0, Longitude: -77.0}, LocationName: "DC"},
		User: UserInfo{UserID: "user_test"},
	}
	repo.moments[m1.MomentID] = m1
	repo.moments[m2.MomentID] = m2

	// reference point near DC (39.0, -77.0)
	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(
		`{"radius":500000,"sort_by":"distance","reference_point":{"latitude":39.0,"longitude":-77.0}}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Search(rr, req)
	var resp struct {
		Code int              `json:"code"`
		Data PaginatedMoments `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Data.Items))
	}
	// nearest first: near (DC) then far (NYC)
	if resp.Data.Items[0].MomentID != "moment_near" || resp.Data.Items[1].MomentID != "moment_far" {
		var order []string
		for _, it := range resp.Data.Items {
			order = append(order, it.MomentID)
		}
		t.Fatalf("expected [moment_near, moment_far], got %v", order)
	}
}

func TestSearchSortByPopular(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_test", SessionID: "s1"})

	m1 := MomentItem{
		MomentID: "moment_unpopular", Description: "unpopular", PrivacyLevel: "public",
		CreatedAt: "2025-06-01T00:00:00Z", LikesCount: 1, CommentsCount: 0,
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m2 := MomentItem{
		MomentID: "moment_popular", Description: "popular", PrivacyLevel: "public",
		CreatedAt: "2025-01-01T00:00:00Z", LikesCount: 10, CommentsCount: 5,
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m3 := MomentItem{
		MomentID: "moment_medium", Description: "medium", PrivacyLevel: "public",
		CreatedAt: "2025-03-01T00:00:00Z", LikesCount: 3, CommentsCount: 2,
		Location: Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	repo.moments[m1.MomentID] = m1
	repo.moments[m2.MomentID] = m2
	repo.moments[m3.MomentID] = m3

	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(
		`{"keyword":"","radius":50000,"sort_by":"popular","reference_point":{"latitude":0,"longitude":0}}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Search(rr, req)
	var resp struct {
		Code int              `json:"code"`
		Data PaginatedMoments `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Data.Items))
	}
	// popular first: popular(15), medium(5), unpopular(1)
	if resp.Data.Items[0].MomentID != "moment_popular" || resp.Data.Items[1].MomentID != "moment_medium" || resp.Data.Items[2].MomentID != "moment_unpopular" {
		var order []string
		for _, it := range resp.Data.Items {
			order = append(order, it.MomentID)
		}
		t.Fatalf("expected [moment_popular, moment_medium, moment_unpopular], got %v", order)
	}
}

func TestCreateMomentAcceptsLocationWithoutName(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hello","location":{"point":{"latitude":35.68,"longitude":139.69}}}`))
	createReq.Header.Set("Idempotency-Key", "zzzz1234zzzz1234zzzz1234zzzz1234")
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			MomentID string `json:"moment_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected success (code 0), got code %d", resp.Code)
	}
	if resp.Data.MomentID == "" {
		t.Fatal("expected moment_id in response")
	}
}

func TestCreateMomentRejectsZeroLocation(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hello","location":{"point":{"latitude":0,"longitude":0}}}`))
	createReq.Header.Set("Idempotency-Key", "yyyy1234yyyy1234yyyy1234yyyy1234")
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10000 {
		t.Fatalf("expected code 10000 for zero coordinates, got %d", resp.Code)
	}
}
