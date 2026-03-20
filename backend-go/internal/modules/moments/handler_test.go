package moments

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/modules/comments"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

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
