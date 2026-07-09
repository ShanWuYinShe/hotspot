package private_messages

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/realtime"
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
func (r *privateMessageErrorRepo) History(context.Context, string, GetHistoryRequest) (PaginatedMessages, error) {
	return PaginatedMessages{}, r.historyErr
}
func (r *privateMessageErrorRepo) Recent(context.Context, string, string, int, *time.Time) (PaginatedMessages, error) {
	return PaginatedMessages{}, r.recentErr
}
func (r *privateMessageErrorRepo) Last(context.Context, string, string, *time.Time) (*PrivateMessageItem, error) {
	return nil, r.lastErr
}
func (r *privateMessageErrorRepo) UnreadCount(context.Context, string, string, *time.Time) (int, error) {
	return 0, r.recentErr
}
func (r *privateMessageErrorRepo) Update(context.Context, string, UpdatePrivateMessageRequest) (MessageEventMeta, error) {
	return MessageEventMeta{}, r.updateErr
}
func (r *privateMessageErrorRepo) Recall(context.Context, string, RecallPrivateMessageRequest) (MessageEventMeta, error) {
	return MessageEventMeta{}, r.recallErr
}
func (r *privateMessageErrorRepo) MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) (MessageEventMeta, error) {
	return MessageEventMeta{}, r.markErr
}
func (r *privateMessageErrorRepo) MarkAllRead(context.Context, string) error {
	return r.markAllErr
}
func (r *privateMessageErrorRepo) GetMessageForReply(context.Context, string, string) (map[string]any, bool, error) {
	return nil, false, nil
}
func (r *privateMessageErrorRepo) GetMediaForMessage(context.Context, string, string) (map[string]any, bool, error) {
	return nil, false, nil
}

type privateMessageUserStub struct {
	item     users.UserItem
	ok       bool
	openErr  error
	settings users.UserSettings
	setErr   error
}

func (s *privateMessageUserStub) OpenTheDoor(context.Context, string) (users.UserItem, bool, error) {
	return s.item, s.ok, s.openErr
}

func (s *privateMessageUserStub) GetMySettings(context.Context) (users.UserSettings, error) {
	return s.settings, s.setErr
}

func withPrivateMessagePrincipal(req *http.Request) *http.Request {
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	return req.WithContext(ctx)
}

// --- realtime event capture ---

type eventSink struct {
	events []realtime.Event
}

func (s *eventSink) Send(e realtime.Event) error {
	s.events = append(s.events, e)
	return nil
}
func (s *eventSink) Close() error { return nil }

// --- realtime broadcast tests ---

func TestSendBroadcastsPrivateMessageNewToTarget(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)

	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone},
	}
	service := NewServiceWithDeps(NewInMemoryRepository(), userStub)

	bobSink := &eventSink{}
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	msg, err := service.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if len(bobSink.events) != 1 {
		t.Fatalf("expected 1 event for target, got %d", len(bobSink.events))
	}
	event := bobSink.events[0]
	if event.Type != "PrivateMessageNew" {
		t.Fatalf("expected PrivateMessageNew, got %s", event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", event.Payload)
	}
	if payload["message_id"] != msg.MessageID {
		t.Fatalf("expected message_id %s, got %v", msg.MessageID, payload["message_id"])
	}
}

func TestUpdateBroadcastsPrivateMessageEditedToBothParties(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)

	repo := NewInMemoryRepository()
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone},
	}
	service := NewServiceWithDeps(repo, userStub)

	// alice sends a message to bob
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	sent, err := service.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "original"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	aliceSink := &eventSink{}
	bobSink := &eventSink{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	// alice updates the message
	updateCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	if err := service.Update(updateCtx, UpdatePrivateMessageRequest{MessageID: sent.MessageID, Content: "updated"}); err != nil {
		t.Fatalf("update: %v", err)
	}

	if len(aliceSink.events) != 1 {
		t.Fatalf("expected 1 event for alice, got %d", len(aliceSink.events))
	}
	if aliceSink.events[0].Type != "PrivateMessageEdited" {
		t.Fatalf("expected PrivateMessageEdited, got %s", aliceSink.events[0].Type)
	}

	if len(bobSink.events) != 1 {
		t.Fatalf("expected 1 event for bob, got %d", len(bobSink.events))
	}
	if bobSink.events[0].Type != "PrivateMessageEdited" {
		t.Fatalf("expected PrivateMessageEdited, got %s", bobSink.events[0].Type)
	}
}

func TestRecallBroadcastsPrivateMessageRecalledToBothParties(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)

	repo := NewInMemoryRepository()
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone},
	}
	service := NewServiceWithDeps(repo, userStub)

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	sent, err := service.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "to recall"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	aliceSink := &eventSink{}
	bobSink := &eventSink{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	recallCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	if err := service.Recall(recallCtx, RecallPrivateMessageRequest{MessageID: sent.MessageID}); err != nil {
		t.Fatalf("recall: %v", err)
	}

	if len(aliceSink.events) != 1 || aliceSink.events[0].Type != "PrivateMessageRecalled" {
		t.Fatalf("expected PrivateMessageRecalled for alice, got %v", aliceSink.events)
	}
	if len(bobSink.events) != 1 || bobSink.events[0].Type != "PrivateMessageRecalled" {
		t.Fatalf("expected PrivateMessageRecalled for bob, got %v", bobSink.events)
	}
}

func TestMarkAsReadBroadcastsPrivateMessageReadToTarget(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)

	repo := NewInMemoryRepository()
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone},
	}
	service := NewServiceWithDeps(repo, userStub)

	// alice sends to bob
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	sent, err := service.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// alice marks as read
	aliceSink := &eventSink{}
	bobSink := &eventSink{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	markCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	if err := service.MarkAsRead(markCtx, MarkPrivateMessagesReadRequest{TargetUserID: "user_bob", UpToMessageID: sent.MessageID}); err != nil {
		t.Fatalf("mark as read: %v", err)
	}

	// Only bob should receive the read event (alice is the marker, not the target)
	if len(bobSink.events) != 1 {
		t.Fatalf("expected 1 event for bob, got %d", len(bobSink.events))
	}
	if bobSink.events[0].Type != "PrivateMessageRead" {
		t.Fatalf("expected PrivateMessageRead, got %s", bobSink.events[0].Type)
	}
}

func TestMarkAllReadBroadcastsSelfReadSync(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)

	repo := NewInMemoryRepository()
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone},
	}
	service := NewServiceWithDeps(repo, userStub)

	sink := &eventSink{}
	hub.RegisterUser("user_alice", "sess_alice", sink)

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	if err := service.MarkAllRead(ctx); err != nil {
		t.Fatalf("mark all read: %v", err)
	}

	if len(sink.events) != 1 {
		t.Fatalf("expected 1 SelfReadSync event, got %d", len(sink.events))
	}
	if sink.events[0].Type != realtime.EventTypeSelfReadSync {
		t.Fatalf("expected SelfReadSync event, got %s", sink.events[0].Type)
	}
}

func TestSendIgnoresRealtimeErrors(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)

	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone},
	}
	service := NewServiceWithDeps(NewInMemoryRepository(), userStub)

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	_, err := service.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("send should succeed even if realtime fails: %v", err)
	}
}

// --- existing handler tests (retained) ---

func TestSendAndHistoryPrivateMessages(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	userStub := &privateMessageUserStub{
		item: users.UserItem{Relation: &users.Relationship{}},
		ok:   true,
		settings: users.UserSettings{
			AllowPrivateChat: users.PrivateChatEveryone,
		},
	}
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), userStub))

	sendReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
	sendReq.Header.Set("Idempotency-Key", "pm_snd_0000000000000000000000001")
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

func TestPrivateMessagesSendValidatesByType(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), &privateMessageUserStub{}))
	cases := []string{
		`{"target_user_id":"user_bob","type":"text"}`,
		`{"target_user_id":"user_bob","type":"image"}`,
		`{"target_user_id":"user_bob","type":"share"}`,
		`{"target_user_id":"user_bob","type":"unknown","content":"x"}`,
	}
	for _, body := range cases {
		req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(body)))
		req.Header.Set("Idempotency-Key", "bbbb1234bbbb1234bbbb1234bbbb1234")
		rr := httptest.NewRecorder()
		handler.Send(rr, req)
		assertPrivateMessageCode(t, rr, 10000)
	}
}

func TestPrivateMessagesGetHistoryRejectsInvalidPaginationAndUnknownFields(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), &privateMessageUserStub{}))

	invalidPaginationReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/get_history", strings.NewReader(`{"target_user_id":"user_bob","pagination":{"limit":101}}`)))
	invalidPaginationRR := httptest.NewRecorder()
	handler.GetHistory(invalidPaginationRR, invalidPaginationReq)
	assertPrivateMessageCode(t, invalidPaginationRR, 10000)

	unknownFieldReq := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/get_history", strings.NewReader(`{"target_user_id":"user_bob","extra":true}`)))
	unknownFieldRR := httptest.NewRecorder()
	handler.GetHistory(unknownFieldRR, unknownFieldReq)
	assertPrivateMessageCode(t, unknownFieldRR, 10000)
}

func TestPrivateMessagesHandlerMapsErrors(t *testing.T) {
	t.Run("send and history generic errors", func(t *testing.T) {
		handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{sendErr: errors.New("boom"), historyErr: errors.New("boom"), markAllErr: errors.New("boom")}, &privateMessageUserStub{}))

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
				handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{updateErr: tc.err}, &privateMessageUserStub{}))
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
				handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{recallErr: tc.err}, &privateMessageUserStub{}))
				req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"message_id":"pm_1"}`)))
				rr := httptest.NewRecorder()
				handler.Recall(rr, req)
				assertPrivateMessageCode(t, rr, tc.code)
			})
		}
	})

	t.Run("mark as read maps not found", func(t *testing.T) {
		handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{markErr: ErrMessageNotFound}, &privateMessageUserStub{}))
		req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_as_read", strings.NewReader(`{"target_user_id":"user_bob","up_to_message_id":"pm_1"}`)))
		rr := httptest.NewRecorder()
		handler.MarkAsRead(rr, req)
		assertPrivateMessageCode(t, rr, 10300)
	})

	t.Run("mark as read rejects missing target user", func(t *testing.T) {
		handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{}, &privateMessageUserStub{}))
		req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_as_read", strings.NewReader(`{"up_to_message_id":"pm_1"}`)))
		rr := httptest.NewRecorder()
		handler.MarkAsRead(rr, req)
		assertPrivateMessageCode(t, rr, 10000)
	})

	t.Run("mark all read requires empty object", func(t *testing.T) {
		handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{}, &privateMessageUserStub{}))
		req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/mark_all_read", strings.NewReader(`{"extra":true}`)))
		rr := httptest.NewRecorder()
		handler.MarkAllRead(rr, req)
		assertPrivateMessageCode(t, rr, 10000)
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
