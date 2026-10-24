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
	"hotspot/backend-go/internal/modules/notify"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
	"hotspot/backend-go/internal/platform/testdb"
	"hotspot/backend-go/internal/platform/types"
)

type failingMomentCounter struct {
	repo    *InMemoryRepository
	failErr error
}

func (c *failingMomentCounter) CheckCommentable(ctx context.Context, momentID string) (bool, bool, error) {
	_, ok, err := c.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return ok, false, err
	}
	return true, true, nil
}

func (c *failingMomentCounter) IsReactionTargetVisible(ctx context.Context, momentID string) (bool, bool, error) {
	item, ok, err := c.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return ok, false, err
	}
	if item.IsDeleted {
		return false, false, nil
	}
	return true, item.PrivacyLevel != "private", nil
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
	return comments.MomentNotificationContext{UserID: item.User.UserID, Summary: types.MomentSummary{MomentID: item.MomentID, DescriptionPreview: item.Description}}, true, nil
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
		Code httpx.Code `json:"code"`
		Data MomentItem `json:"data"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	var createResp2 struct {
		Code httpx.Code `json:"code"`
		Data MomentItem `json:"data"`
	}
	_ = json.NewDecoder(createRR2.Body).Decode(&createResp2)
	if createResp.Code != 0 || createResp.Data.MomentID == "" || createResp.Data.MomentID != createResp2.Data.MomentID {
		t.Fatalf("unexpected create response: %#v %#v", createResp, createResp2)
	}

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"moment_id":"`+createResp.Data.MomentID+`"}`))
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)

	listReq := httptest.NewRequest(http.MethodPost, "/list_created", strings.NewReader(`{}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	handler.ListCreated(listRR, listReq)
	var listResp struct {
		Code httpx.Code       `json:"code"`
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
		Code httpx.Code `json:"code"`
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
		Code httpx.Code `json:"code"`
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
	commentService := comments.NewServiceWithDeps(commentRepo, counter, notify.NewService(), nil, nil, persistence.NewMemoryTxRunner())
	commentHandler := comments.NewHandler(commentService)
	momentHandler := NewHandler(momentService)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createdMoment, err := momentRepo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	createCommentReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"moment_id":"`+createdMoment.MomentID+`","content":"nice"}`))
	createCommentReq.Header.Set("Idempotency-Key", "yyyy1234yyyy1234yyyy1234yyyy1234")
	createCommentReq = createCommentReq.WithContext(ctx)
	createCommentRR := httptest.NewRecorder()
	commentHandler.Create(createCommentRR, createCommentReq)
	var createCommentResp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(createCommentRR.Body).Decode(&createCommentResp); err != nil {
		t.Fatalf("decode create comment response: %v", err)
	}
	// 计数失败不阻断评论创建（避免客户端重试产生重复评论）——评论成功、计数陈旧。
	if createCommentResp.Code != 0 {
		t.Fatalf("comment create should succeed despite counter failure, got %#v", createCommentResp)
	}
	commentsList, err := commentRepo.List(context.Background(), createdMoment.MomentID, 20, "")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(commentsList.Items) != 1 {
		t.Fatalf("comment should be committed, got %#v", commentsList)
	}
	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"moment_id":"`+createdMoment.MomentID+`"}`))
	openRR := httptest.NewRecorder()
	momentHandler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code httpx.Code `json:"code"`
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
	assertCommentCode(t, createCommentRR, httpx.CodeNotFound)
}

func TestListCommentsReturns10300WhenMomentMissing(t *testing.T) {
	commentService := comments.NewService(comments.NewInMemoryRepository(), NewServiceWithRepo(NewInMemoryRepository()))
	commentHandler := comments.NewHandler(commentService)

	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"moment_id":"moment_missing"}`))
	listRR := httptest.NewRecorder()
	commentHandler.List(listRR, listReq)
	assertCommentCode(t, listRR, httpx.CodeNotFound)
}

func TestCreateMomentRecordsResourceGuard(t *testing.T) {
	checker := &momentRecordingChecker{}
	resourceguard.SetDefaultChecker(checker)
	defer resourceguard.SetDefaultChecker(resourceguard.NoopChecker{})

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	service := NewServiceWithRepo(NewInMemoryRepository())
	if _, err := service.Create(ctx, CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Somewhere"}}); err != nil {
		t.Fatalf("create moment: %v", err)
	}
	if len(checker.recorded) != 1 || !strings.HasPrefix(checker.recorded[0], "moment:") {
		t.Fatalf("unexpected records: %#v", checker.recorded)
	}
}

func TestCreateMomentSendsMentionNotificationsAndRealtime(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(NewInMemoryRepository(), notify.NewServiceWithRepo(notificationRepo), nil)
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	bobSink := &sinkCapture{}
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	created, err := service.Create(ctxAlice, CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Somewhere"}, Mentions: []string{"user_bob", "user_bob", "user_alice"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	if len(created.Mentions) != 2 || created.Mentions[0] != "user_bob" || created.Mentions[1] != "user_alice" {
		t.Fatalf("unexpected mentions: %#v", created.Mentions)
	}
	if len(bobSink.events) != 1 || bobSink.events[0].Type != "ContentMentioned" {
		t.Fatalf("unexpected realtime events: %#v", bobSink.events)
	}
	payload := decodePayload(t, bobSink.events[0].Payload)
	if payload["target_type"] != "moment" || payload["moment_id"] != created.MomentID {
		t.Fatalf("unexpected realtime payload: %#v", payload)
	}
	notificationID, _ := payload["notify_id"].(string)
	if notificationID == "" {
		t.Fatalf("expected non-empty notification_id in event payload: %#v", payload)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_bob", notify.NotifyListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "mentioned" {
		t.Fatalf("unexpected notifications page: %#v", notificationsPage)
	}
	if notificationsPage.Items[0].NotifyID != notificationID {
		t.Fatalf("expected event notification_id to match stored item, got event=%q stored=%q", notificationID, notificationsPage.Items[0].NotifyID)
	}
}

func TestUpdateMomentReplacesMentionNotificationsAndState(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(NewInMemoryRepository(), notify.NewServiceWithRepo(notificationRepo), nil)
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	bobSink := &sinkCapture{}
	carolSink := &sinkCapture{}
	hub.RegisterUser("user_bob", "sess_bob", bobSink)
	hub.RegisterUser("user_carol", "sess_carol", carolSink)

	created, err := service.Create(ctxAlice, CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Somewhere"}, Mentions: []string{"user_bob"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	updated, err := service.Update(ctxAlice, UpdateMomentRequest{MomentID: created.MomentID, Mentions: []string{"user_carol"}})
	if err != nil {
		t.Fatalf("update moment: %v", err)
	}
	if len(updated.Mentions) != 1 || updated.Mentions[0] != "user_carol" {
		t.Fatalf("unexpected updated mentions: %#v", updated.Mentions)
	}
	if len(carolSink.events) != 1 || carolSink.events[0].Type != "ContentMentioned" {
		t.Fatalf("unexpected carol events: %#v", carolSink.events)
	}
	carolNotifications, err := notificationRepo.List(context.Background(), "user_carol", notify.NotifyListRequest{})
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

	if _, err := service.Create(ctx, CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Somewhere"}}); err != nil {
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

	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"moment_id":"`+created.MomentID+`"}`))
	deleteReq.Header.Set("Idempotency-Key", "delete_perm_denied_moments_ggggggg")
	deleteReq = deleteReq.WithContext(ctxOther)
	deleteRR := httptest.NewRecorder()
	handler.Delete(deleteRR, deleteReq)

	var resp struct {
		Code    httpx.Code `json:"code"`
		Message string     `json:"message"`
	}
	if err := json.NewDecoder(deleteRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeForbidden {
		t.Fatalf("expected code httpx.CodeForbidden, got code=%d message=%s", resp.Code, resp.Message)
	}
	if resp.Message != "权限不足" {
		t.Fatalf("expected message '权限不足', got %s", resp.Message)
	}
}

func assertCommentCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
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
			body := `{"keyword":"test","sort_by":"` + invalid + `"}`
			req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(body))
			rr := httptest.NewRecorder()
			handler.Search(rr, req)
			var resp struct {
				Code httpx.Code `json:"code"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Code != httpx.CodeInvalidRequest {
				t.Fatalf("expected code httpx.CodeInvalidRequest for sort_by=%q, got %d", invalid, resp.Code)
			}
		})
	}
}

func TestSearchAcceptsValidSortBy(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	// popular 为热门模式：必须无输入条件（keyword/tag 缺省），与 keyword 同传
	// 视为参数冲突（spec moment.md → moments/search 热门模式互斥约束）。
	for _, valid := range []string{"relevance", "newest", ""} {
		t.Run("sort_by="+valid, func(t *testing.T) {
			sortField := ""
			if valid != "" {
				sortField = `,"sort_by":"` + valid + `"`
			}
			body := `{"keyword":"test"` + sortField + `}`
			req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(body))
			rr := httptest.NewRecorder()
			handler.Search(rr, req)
			var resp struct {
				Code httpx.Code `json:"code"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Code != 0 {
				t.Fatalf("expected code 0 for sort_by=%q, got %d", valid, resp.Code)
			}
		})
	}
	// 热门模式：仅显式 sort_by=popular、keyword/tag 均缺省
	t.Run("popular_mode_no_input", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"sort_by":"popular"}`))
		rr := httptest.NewRecorder()
		handler.Search(rr, req)
		var resp struct {
			Code httpx.Code `json:"code"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Code != 0 {
			t.Fatalf("expected code 0 for popular mode, got %d", resp.Code)
		}
	})
	// 热门模式与输入条件互斥
	for _, body := range []string{
		`{"sort_by":"popular","keyword":"test"}`,
		`{"sort_by":"popular","tag":"travel"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.Search(rr, req)
		var resp struct {
			Code httpx.Code `json:"code"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("expected httpx.CodeInvalidRequest for %s, got %d", body, resp.Code)
		}
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
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m2 := MomentItem{
		MomentID: "moment_b", Description: "beta test", PrivacyLevel: "public",
		CreatedAt: "2025-06-01T00:00:00Z",
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m3 := MomentItem{
		MomentID: "moment_c", Description: "gamma test", PrivacyLevel: "public",
		CreatedAt: "2025-03-01T00:00:00Z",
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	repo.moments[m1.MomentID] = m1
	repo.moments[m2.MomentID] = m2
	repo.moments[m3.MomentID] = m3

	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(
		`{"keyword":"test","sort_by":"newest"}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Search(rr, req)
	var resp struct {
		Code httpx.Code       `json:"code"`
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

func TestSearchSortByPopular(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_test", SessionID: "s1"})

	m1 := MomentItem{
		MomentID: "moment_unpopular", Description: "unpopular", PrivacyLevel: "public",
		CreatedAt: "2025-06-01T00:00:00Z", LikesCount: 1, CommentsCount: 0,
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m2 := MomentItem{
		MomentID: "moment_popular", Description: "popular", PrivacyLevel: "public",
		CreatedAt: "2025-01-01T00:00:00Z", LikesCount: 10, CommentsCount: 5,
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	m3 := MomentItem{
		MomentID: "moment_medium", Description: "medium", PrivacyLevel: "public",
		CreatedAt: "2025-03-01T00:00:00Z", LikesCount: 3, CommentsCount: 2,
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 0, Longitude: 0}, LocationName: "Here"},
		User: UserInfo{UserID: "user_test"},
	}
	repo.moments[m1.MomentID] = m1
	repo.moments[m2.MomentID] = m2
	repo.moments[m3.MomentID] = m3

	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(
		`{"keyword":"","sort_by":"popular"}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Search(rr, req)
	var resp struct {
		Code httpx.Code       `json:"code"`
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
		Code httpx.Code `json:"code"`
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

	// location 字段缺失（JSON null / 不传）→ 指针为 nil → 拒绝；
	// (0,0) 本身是合法坐标（赤道×本初子午线），不再被哨兵误拒。
	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hello"}`))
	createReq.Header.Set("Idempotency-Key", "yyyy1234yyyy1234yyyy1234yyyy1234")
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected code httpx.CodeInvalidRequest for missing location, got %d", resp.Code)
	}
}

// TestCreateMomentAcceptsZeroZeroCoordinates 验证 (0,0) 合法坐标不再被
// 旧的 (0,0)+空名哨兵误拒（修复回归测试）。
func TestCreateMomentAcceptsZeroZeroCoordinates(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"description":"hello","location":{"point":{"latitude":0,"longitude":0},"location_name":"几内亚湾"}}`))
	createReq.Header.Set("Idempotency-Key", "zzro1234zzro1234zzro1234zzro1234")
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected success for (0,0) coordinates, got %d", resp.Code)
	}
}

// --- Update/Delete handler tests ---

func TestMomentHandlerUpdateRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{invalid`))
	req.Header.Set("Idempotency-Key", "update_invalid_json_moments_aaaaaa")
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMomentHandlerUpdateRejectsMissingMomentID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"description":"new"}`))
	req.Header.Set("Idempotency-Key", "update_no_moment_id_moments_bbbbbb")
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMomentHandlerUpdateReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	created, err := repo.Create(ctx, "user_alice", CreateMomentRequest{Description: "original", Location: &Location{LocationName: "Here", Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 1, Longitude: 2}}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	newDesc := "updated"
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"moment_id":"`+created.MomentID+`","description":"`+newDesc+`"}`))
	req.Header.Set("Idempotency-Key", "update_success_moments_ccccccccccc")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
		Data MomentItem `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.Description != newDesc {
		t.Fatalf("expected description %q, got %q", newDesc, resp.Data.Description)
	}
}

func TestMomentHandlerUpdateReturns10600OnError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&errorRepo{}))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"moment_id":"any_moment","description":"x"}`))
	req.Header.Set("Idempotency-Key", "update_error_moments_ddddddddddddd")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	assertMomentCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestMomentHandlerDeleteRejectsMissingMomentID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{}`))
	req.Header.Set("Idempotency-Key", "delete_no_moment_id_moments_eeeeee")
	rr := httptest.NewRecorder()
	handler.Delete(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMomentHandlerDeleteReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	created, err := repo.Create(ctx, "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"moment_id":"`+created.MomentID+`"}`))
	req.Header.Set("Idempotency-Key", "delete_success_moments_fffffffffff")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}

	// 软删除（spec api_spec_moment.md「删除语义」）：记录保留，标记 is_deleted=true
	// 且 description 清空；对作者 fetch_created 仍返回占位项。
	item, ok, _ := repo.Get(ctx, created.MomentID)
	if !ok {
		t.Fatal("soft-deleted moment record must be retained")
	}
	if !item.IsDeleted || item.Description != "" {
		t.Fatalf("expected soft-delete placeholder (is_deleted=true, empty description), got %#v", item)
	}
}

// --- Favorite/Unfavorite handler tests ---

func TestMomentHandlerFavoriteRejectsMissingMomentID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/set_favorite", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.Favorite(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMomentHandlerFavoriteReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_2"})

	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/set_favorite", strings.NewReader(`{"moment_id":"`+created.MomentID+`","active":true}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Favorite(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestMomentHandlerUnfavoriteRejectsMissingMomentID(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/set_favorite", strings.NewReader(`{"active":false}`))
	rr := httptest.NewRecorder()
	handler.Favorite(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMomentHandlerUnfavoriteReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_2"})

	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	// 先收藏
	favReq := httptest.NewRequest(http.MethodPost, "/set_favorite", strings.NewReader(`{"moment_id":"`+created.MomentID+`","active":true}`))
	favReq = favReq.WithContext(ctx)
	favRR := httptest.NewRecorder()
	handler.Favorite(favRR, favReq)

	// 再取消收藏
	req := httptest.NewRequest(http.MethodPost, "/set_favorite", strings.NewReader(`{"moment_id":"`+created.MomentID+`","active":false}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Favorite(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

// --- ListFavorites handler tests ---

func TestMomentHandlerListFavoritesReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	created, err := repo.Create(ctxAlice, "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	// Alice 收藏自己的 moment
	favReq := httptest.NewRequest(http.MethodPost, "/set_favorite", strings.NewReader(`{"moment_id":"`+created.MomentID+`","active":true}`))
	favReq = favReq.WithContext(ctxAlice)
	favRR := httptest.NewRecorder()
	handler.Favorite(favRR, favReq)

	// 列出收藏
	req := httptest.NewRequest(http.MethodPost, "/list_favorites", strings.NewReader(`{}`))
	req = req.WithContext(ctxAlice)
	rr := httptest.NewRecorder()
	handler.ListFavorites(rr, req)

	var resp struct {
		Code httpx.Code       `json:"code"`
		Data PaginatedMoments `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 favorite item, got %d", len(resp.Data.Items))
	}
	if resp.Data.Items[0].MomentID != created.MomentID {
		t.Fatalf("expected moment_id %s, got %s", created.MomentID, resp.Data.Items[0].MomentID)
	}
}

// --- Search handler tests ---

func TestMomentHandlerSearchRejectsNoCriteria(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.Search(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMomentHandlerSearchRejectsInvalidSortBy(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"test","sort_by":"bogus"}`))
	rr := httptest.NewRecorder()
	handler.Search(rr, req)
	assertMomentCode(t, rr, httpx.CodeInvalidRequest)
}

func TestMomentHandlerSearchReturnsSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	_, err := repo.Create(ctx, "user_alice", CreateMomentRequest{Description: "findme moment", Location: &Location{LocationName: "Here", Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 1, Longitude: 2}}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"findme"}`))
	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	var resp struct {
		Code httpx.Code       `json:"code"`
		Data PaginatedMoments `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(resp.Data.Items))
	}
}

// --- helpers ---

func assertMomentCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %d", expected, resp.Code)
	}
}

// errorRepo is a Repository implementation that always returns errors for Get,
// used to trigger the httpx.CodeInternalUnavailable server error path in Update handler.
type errorRepo struct{}

func (r *errorRepo) Create(context.Context, string, CreateMomentRequest) (MomentItem, error) {
	return MomentItem{}, nil
}
func (r *errorRepo) Get(context.Context, string) (MomentItem, bool, error) {
	return MomentItem{}, false, errors.New("db error")
}
func (r *errorRepo) Search(context.Context, SearchMomentsRequest, string) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}
func (r *errorRepo) Update(context.Context, string, UpdateMomentRequest) error {
	return errors.New("db error")
}
func (r *errorRepo) Delete(context.Context, string) error { return nil }
func (r *errorRepo) ListMine(context.Context, string, int, string) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}
func (r *errorRepo) Exists(context.Context, string) (bool, error)           { return false, nil }
func (r *errorRepo) AdjustReactionCount(context.Context, string, int) error { return nil }

func (r *errorRepo) IncrementCommentCount(context.Context, string) error { return nil }
func (r *errorRepo) DecrementCommentCount(context.Context, string) error { return nil }
func (r *errorRepo) IncrementSharesCount(context.Context, string) error  { return nil }
func (r *errorRepo) Favorite(context.Context, string, string) error      { return nil }
func (r *errorRepo) Unfavorite(context.Context, string, string) error    { return nil }
func (r *errorRepo) ListFavorites(context.Context, string, int, string) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}
func (r *errorRepo) ListFavoriteUserIDs(context.Context, string, int) ([]string, error) {
	return nil, nil
}
func (r *errorRepo) LoadMyState(context.Context, string, string) *MyMomentState { return nil }
func (r *errorRepo) FetchMomentImageRows(context.Context, []string) (map[string][]momentImageRow, error) {
	return map[string][]momentImageRow{}, nil
}
func (r *errorRepo) ListFollowingMoments(context.Context, momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}
func (r *errorRepo) ListNearbyMoments(context.Context, momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}
func (r *errorRepo) NotificationContext(context.Context, string) (MomentItem, bool, error) {
	return MomentItem{}, false, nil
}

// --- image_file_ids 不存在或不属于当前用户 → 10300（spec api_spec_moment.md
// moments/create、moments/update 错误码；此前误映射 10000）---

func TestCreateMomentUnownedImagesReturns10300(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	resolverErr := errors.New("file not owned")
	handler := NewHandler(NewServiceWithDepsAndImages(repo, nil, nil, &momentsImageResolver{err: resolverErr}))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	body := `{"description":"hello","location":{"point":{"latitude":35.0,"longitude":112.0}},"image_file_ids":["file_foreign"]}`
	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(body))
	req.Header.Set("Idempotency-Key", "imgown1234imgown1234imgown123400")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("unowned image_file_ids on create: expected httpx.CodeNotFound(10300), got %d body=%s", code, rr.Body.String())
	}
}

func TestUpdateMomentUnownedImagesReturns10300(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Here"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}

	resolverErr := errors.New("file not owned")
	handler := NewHandler(NewServiceWithDepsAndImages(repo, nil, nil, &momentsImageResolver{err: resolverErr}))
	body := `{"moment_id":"` + created.MomentID + `","image_file_ids":["file_foreign"]}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("unowned image_file_ids on update: expected httpx.CodeNotFound(10300), got %d body=%s", code, rr.Body.String())
	}
}

func (r *errorRepo) ListMomentsByAuthor(context.Context, string, int, string, bool) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}
func (r *errorRepo) ListReactedByUser(context.Context, string, string, int, string) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}

// decodePayload 把 WS payload（现为类型化 struct）经 JSON round-trip 转回
// map，让既有键级断言继续验证「序列化后的形状」——那才是真正的线上契约。
// 注意数字经 JSON 解码为 float64。
func decodePayload(t *testing.T, p any) map[string]any {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("payload is not an object: %v (%#v)", err, p)
	}
	return m
}

// 契约（api_spec_base.md「搜索与地理约束」）：keyword 1-100、tag 1-30 字符。
func TestSearchRejectsOversizeKeywordAndTag(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	cases := []string{
		"{\"keyword\":\"" + strings.Repeat("a", 101) + "\"}",
		"{\"tag\":\"" + strings.Repeat("a", 31) + "\"}",
	}
	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.Search(rr, req)
		var resp struct {
			Code httpx.Code `json:"code"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("expected 10000 for %s, got %d", body, resp.Code)
		}
	}
}
