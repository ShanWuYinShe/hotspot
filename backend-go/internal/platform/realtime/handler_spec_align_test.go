package realtime

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
)

// stubTypingActorProvider resolves actor info for resolveTypingActor.
type stubTypingActorProvider struct {
	infos map[string]actor.Info
	err   error
}

func (s stubTypingActorProvider) GetActorInfo(_ context.Context, userID string) (actor.Info, error) {
	if s.err != nil {
		return actor.Info{}, s.err
	}
	if info, ok := s.infos[userID]; ok {
		return info, nil
	}
	return actor.Info{UserID: userID, Nickname: userID}, nil
}

// eventSink records delivered events for hub-based assertions. The hub calls
// Send from its own goroutine while the test reads events from the test
// goroutine, so the slice is guarded by a mutex.
type eventSink struct {
	mu     sync.Mutex
	events []Event
}

func (s *eventSink) Send(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *eventSink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *eventSink) Close() error { return nil }

// --- resolveTypingActor ---

func TestResolveTypingActorWithProvider(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil)
	defer h.Close()
	h.SetActorProvider(stubTypingActorProvider{infos: map[string]actor.Info{
		"u1": {UserID: "u1", Nickname: "Alice", AvatarURL: "https://a.example/alice.png"},
	}})

	a := h.resolveTypingActor(context.Background(), "u1")
	if a["user_id"] != "u1" || a["nickname"] != "Alice" || a["avatar_url"] != "https://a.example/alice.png" {
		t.Fatalf("unexpected actor: %+v", a)
	}
}

func TestResolveTypingActorFallsBackWithoutProvider(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil) // actorProvider == nil
	defer h.Close()

	a := h.resolveTypingActor(context.Background(), "u1")
	if a["user_id"] != "u1" || a["nickname"] != "u1" {
		t.Fatalf("expected fallback actor, got %+v", a)
	}
	if _, has := a["avatar_url"]; has {
		t.Fatalf("fallback actor must not carry avatar_url, got %+v", a)
	}
}

func TestResolveTypingActorFallsBackOnProviderError(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil)
	defer h.Close()
	h.SetActorProvider(stubTypingActorProvider{err: errors.New("boom")})

	a := h.resolveTypingActor(context.Background(), "u2")
	if a["user_id"] != "u2" || a["nickname"] != "u2" {
		t.Fatalf("expected fallback actor on provider error, got %+v", a)
	}
}

// --- canSendPrivateTyping 分支 ---

func TestCanSendPrivateTypingRejectsEmptyOrMissingDeps(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil) // userRepo == nil
	defer h.Close()

	if h.canSendPrivateTyping(context.Background(), "", "b") {
		t.Fatal("empty viewer must be rejected")
	}
	if h.canSendPrivateTyping(context.Background(), "a", "") {
		t.Fatal("empty target must be rejected")
	}
	if h.canSendPrivateTyping(context.Background(), "a", "b") {
		t.Fatal("nil userRepo must be rejected")
	}
}

func TestCanSendPrivateTypingPermissionBranches(t *testing.T) {
	repo := stubUserRepo{
		settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{
			"target_nobody":   {allowPrivateChat: "nobody"},
			"target_friends":  {allowPrivateChat: "friends"},
			"target_everyone": {allowPrivateChat: "everyone"},
			// target_default: 无 settings 记录 → ok=false → 默认 everyone
		},
		relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{
			"viewer_mutual":  {"target_friends": {isFollowing: true, isFollowedBy: true}},
			"viewer_weak":    {"target_friends": {isFollowing: true, isFollowedBy: false}},
			"viewer_blocked": {"target_everyone": {isBlocked: true}},
		},
	}
	h := NewHandler(nil, nil, repo, nil)
	defer h.Close()
	ctx := context.Background()

	cases := []struct {
		name   string
		viewer string
		target string
		want   bool
	}{
		{name: "nobody setting", viewer: "anyone", target: "target_nobody", want: false},
		{name: "friends but not mutual", viewer: "viewer_weak", target: "target_friends", want: false},
		{name: "friends and mutual", viewer: "viewer_mutual", target: "target_friends", want: true},
		{name: "everyone", viewer: "anyone", target: "target_everyone", want: true},
		{name: "blocked overrides everyone", viewer: "viewer_blocked", target: "target_everyone", want: false},
		{name: "missing settings default everyone", viewer: "anyone", target: "target_default", want: true},
	}
	for _, tc := range cases {
		if got := h.canSendPrivateTyping(ctx, tc.viewer, tc.target); got != tc.want {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}

func TestCanSendPrivateTypingRejectsOnRepoErrors(t *testing.T) {
	h := NewHandler(nil, nil, errUserRepo{settingsErr: errors.New("boom")}, nil)
	defer h.Close()
	if h.canSendPrivateTyping(context.Background(), "a", "b") {
		t.Fatal("settings error must reject")
	}

	h2 := NewHandler(nil, nil, errUserRepo{relErr: errors.New("boom")}, nil)
	defer h2.Close()
	if h2.canSendPrivateTyping(context.Background(), "a", "b") {
		t.Fatal("relationship error must reject")
	}
}

type errUserRepo struct {
	settingsErr error
	relErr      error
}

func (r errUserRepo) GetSettings(context.Context, string) (bool, string, bool, error) {
	return false, "", false, r.settingsErr
}

func (r errUserRepo) Relationship(context.Context, string, string) (bool, bool, bool, error) {
	return false, false, false, r.relErr
}

// --- PrivateTyping 事件推送 payload 结构(target_user_id + actor) ---

func TestRealtimePrivateTypingPushesPayloadToTarget(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	handler := NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{}},
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{"user_bob": {allowPrivateChat: "everyone"}}, relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{}},
		stubGroupRepo{members: map[string][]string{}},
	)
	handler.SetActorProvider(stubTypingActorProvider{infos: map[string]actor.Info{
		"user_alice": {UserID: "user_alice", Nickname: "Alice"},
	}})
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}})
	var established Event
	_ = conn.ReadJSON(&established)

	// 目标用户 bob 注册到 hub(直接注册 sink,无需真实 WS 连接)。
	bobSink := &eventSink{}
	DefaultHub().RegisterUser("user_bob", "sess_bob", bobSink)

	if err := conn.WriteJSON(Event{Type: "PrivateTyping", Payload: PrivateTypingSendPayload{TargetUserID: "user_bob", IsTyping: true}}); err != nil {
		t.Fatalf("write typing: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(bobSink.Events()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got := bobSink.Events()
	if len(got) != 1 {
		t.Fatalf("expected 1 PrivateTyping event to target, got %+v", got)
	}
	event := got[0]
	if event.Type != EventTypePrivateTyping {
		t.Fatalf("expected PrivateTyping, got %s", event.Type)
	}
	payload, ok := event.Payload.(PrivateTypingPayload)
	if !ok {
		t.Fatalf("expected PrivateTypingPayload, got %T", event.Payload)
	}
	if payload.TargetUserID != "user_bob" {
		t.Fatalf("expected target_user_id=user_bob, got %q", payload.TargetUserID)
	}
	if !payload.IsTyping {
		t.Fatal("expected is_typing=true")
	}
	if payload.Actor["user_id"] != "user_alice" || payload.Actor["nickname"] != "Alice" {
		t.Fatalf("unexpected actor: %+v", payload.Actor)
	}
}
