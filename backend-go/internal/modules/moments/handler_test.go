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
	commentsList, err := commentRepo.List(context.Background(), createdMoment.MomentID)
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
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo))
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
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo))
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
	service := NewServiceWithDeps(NewInMemoryRepository(), notifications.NewServiceWithRepo(notificationRepo))
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

func TestCreateMomentDoesNotBroadcastMomentCommented(t *testing.T) {
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
