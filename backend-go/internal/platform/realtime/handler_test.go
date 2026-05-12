package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/ratelimit"
)

func wsURL(serverURL string) string { return "ws" + strings.TrimPrefix(serverURL, "http") + "/ws/v1" }

type stubUserPresenceService struct {
	items map[string]string
}

func (s stubUserPresenceService) OpenTheDoor(_ context.Context, userID string) (string, bool, error) {
	item, ok := s.items[userID]
	return item, ok, nil
}

type stubUserRepo struct {
	settings map[string]struct {
		presenceVisible  bool
		allowPrivateChat string
	}
	relations map[string]map[string]struct {
		isFollowing  bool
		isFollowedBy bool
		isBlocked    bool
	}
}

func (s stubUserRepo) GetSettings(_ context.Context, userID string) (bool, string, bool, error) {
	item, ok := s.settings[userID]
	return item.presenceVisible, item.allowPrivateChat, ok, nil
}

func (s stubUserRepo) Relationship(_ context.Context, userID, targetUserID string) (bool, bool, bool, error) {
	if s.relations[userID] == nil {
		return false, false, false, nil
	}
	item := s.relations[userID][targetUserID]
	return item.isFollowing, item.isFollowedBy, item.isBlocked, nil
}

type stubGroupRepo struct {
	members map[string][]string
}

func (s stubGroupRepo) ListMembers(_ context.Context, groupID string, limit int, cursor string) ([]string, string, error) {
	items := s.members[groupID]
	start := 0
	if cursor != "" {
		for i, userID := range items {
			if userID == cursor {
				start = i + 1
				break
			}
		}
	}
	if start >= len(items) {
		return nil, "", nil
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	nextCursor := ""
	if end < len(items) {
		nextCursor = items[end-1]
	}
	return append([]string(nil), items[start:end]...), nextCursor, nil
}

func (s stubGroupRepo) IsMember(_ context.Context, groupID string, userID string) (bool, error) {
	for _, id := range s.members[groupID] {
		if id == userID {
			return true, nil
		}
	}
	return false, nil
}

func newRealtimeHandlerForTest(authService *platformauth.Service) *Handler {
	return NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{
			"user_alice": "online",
			"user_bob":   "online",
		}},
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{}, relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{}},
		stubGroupRepo{members: map[string][]string{}},
	)
}

func TestRealtimeWebsocketLifecycle(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession("user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	handler := newRealtimeHandlerForTest(authService)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	var established Event
	if err := conn.ReadJSON(&established); err != nil {
		t.Fatalf("read established: %v", err)
	}
	if established.Type != "ConnectionEstablished" {
		t.Fatalf("unexpected event: %#v", established)
	}

	if err := conn.WriteJSON(Event{Type: "Ping", Payload: PingPayload{}}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	var pong Event
	if err := conn.ReadJSON(&pong); err != nil {
		t.Fatalf("read pong: %v", err)
	}
	if pong.Type != "Pong" {
		t.Fatalf("unexpected pong: %#v", pong)
	}

	if err := conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req1", Operation: "set", Type: "user", IDs: []string{"user_bob"}}}); err != nil {
		t.Fatalf("write subscriptions: %v", err)
	}
	var ack Event
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Type != "SubscribeAck" {
		t.Fatalf("unexpected ack: %#v", ack)
	}
	var snapshot Event
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if snapshot.Type != "SubscriptionsSnapshot" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}

	otherResult, _ := authService.IssueSession("user_bob")
	otherConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial other websocket: %v", err)
	}
	defer otherConn.Close()
	_ = otherConn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: otherResult.AccessToken}})
	var establishedOther Event
	_ = otherConn.ReadJSON(&establishedOther)
	var online Event
	if err := conn.ReadJSON(&online); err != nil {
		t.Fatalf("read online event: %v", err)
	}
	if online.Type != "UserOnline" {
		t.Fatalf("unexpected online event: %#v", online)
	}
}

func TestRealtimeRequiresAuthFirst(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	handler := newRealtimeHandlerForTest(authService)
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(Event{Type: "Ping", Payload: PingPayload{}}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	var event struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read error: %v", err)
	}
	if event.Type != "Error" || event.Payload.Severity != "fatal" {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestRealtimeRejectsLegacyStyleAccessToken(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	handler := newRealtimeHandlerForTest(authService)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: "dXNlcl9sZWdhY3kuc2Vzc19sZWdhY3kuMTcxMjAwMDAwMC5zaWc"}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	var event struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read error: %v", err)
	}
	if event.Type != "Error" || event.Payload.Severity != "fatal" {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestRealtimeSubscriptionLimit(t *testing.T) {
	SetDefaultHub(NewHub())
	ratelimit.SetSource(appconfig.NewStaticSource(config.Config{}))
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := newRealtimeHandlerForTest(authService)
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
	ids := make([]string, 501)
	for i := range ids {
		ids[i] = "u"
	}
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req2", Operation: "set", Type: "user", IDs: ids}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Type != "SubscribeAck" || ack.Payload.Code == 0 {
		payload, _ := json.Marshal(ack)
		t.Fatalf("unexpected ack: %s", payload)
	}
}

func TestRealtimeSubscriptionRateLimit(t *testing.T) {
	SetDefaultHub(NewHub())
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(nil, func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.RateLimits.WSSubscribePerSecond = 1
		return cfg
	})
	ratelimit.SetSource(source)
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := newRealtimeHandlerForTest(authService)
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "conn-1", Operation: "set", Type: "user", IDs: []string{"user_bob"}}})
	var ack Event
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	var snapshot Event
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "conn-1", Operation: "set", Type: "user", IDs: []string{"user_charlie"}}})
	var limited struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&limited); err != nil {
		t.Fatalf("read limited event: %v", err)
	}
	if limited.Type != "Error" || limited.Payload.Code != 20300 || limited.Payload.RetryAfter <= 0 {
		payload, _ := json.Marshal(limited)
		t.Fatalf("unexpected limited event: %s", payload)
	}
}

func TestRealtimeRejectsUserSubscriptionWhenPresenceHidden(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{"user_hidden": ""}},
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{"user_hidden": {presenceVisible: false}}, relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{}},
		stubGroupRepo{members: map[string][]string{}},
	)
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-hidden", Operation: "set", Type: "user", IDs: []string{"user_hidden"}}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Payload.Code != 20100 || len(ack.Payload.FailedIDs) != 1 || ack.Payload.FailedIDs[0].Reason != "denied" {
		payload, _ := json.Marshal(ack)
		t.Fatalf("unexpected ack: %s", payload)
	}
}

func TestRealtimeRejectsGroupSubscriptionForNonMember(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{}},
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{}, relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{}},
		stubGroupRepo{members: map[string][]string{"group_1": {"user_bob"}}},
	)
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-group", Operation: "set", Type: "group", IDs: []string{"group_1"}}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Payload.Code != 20100 || len(ack.Payload.FailedIDs) != 1 || ack.Payload.FailedIDs[0].Reason != "denied" {
		payload, _ := json.Marshal(ack)
		t.Fatalf("unexpected ack: %s", payload)
	}
}

func TestRealtimeAllowsUnsubscribeEvenWhenTargetNowDenied(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{"user_hidden": ""}},
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{"user_hidden": {presenceVisible: false}}, relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{}},
		stubGroupRepo{members: map[string][]string{}},
	)
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-unsub", Operation: "unsubscribe", Type: "user", IDs: []string{"user_hidden"}}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Payload.Code != 0 || len(ack.Payload.FailedIDs) != 0 {
		payload, _ := json.Marshal(ack)
		t.Fatalf("unexpected ack: %s", payload)
	}
}

func TestRealtimeRejectsPrivateTypingWhenPrivateChatDenied(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{}},
		stubUserRepo{
			settings: map[string]struct {
				presenceVisible  bool
				allowPrivateChat string
			}{"user_bob": {allowPrivateChat: "nobody"}},
			relations: map[string]map[string]struct {
				isFollowing  bool
				isFollowedBy bool
				isBlocked    bool
			}{},
		},
		stubGroupRepo{members: map[string][]string{}},
	)
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
	_ = conn.WriteJSON(Event{Type: "PrivateTyping", Payload: PrivateTypingPayload{TargetUserID: "user_bob", IsTyping: true}})
	var denied struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&denied); err != nil {
		t.Fatalf("read denied event: %v", err)
	}
	if denied.Type != "Error" || denied.Payload.Code != 20400 {
		payload, _ := json.Marshal(denied)
		t.Fatalf("unexpected denied event: %s", payload)
	}
}

func TestRealtimeRejectsGroupTypingForNonMember(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	handler := NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{}},
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{}, relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{}},
		stubGroupRepo{members: map[string][]string{"group_1": {"user_bob"}}},
	)
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
	_ = conn.WriteJSON(Event{Type: "GroupTyping", Payload: GroupTypingPayload{GroupID: "group_1", IsTyping: true}})
	var denied struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&denied); err != nil {
		t.Fatalf("read denied event: %v", err)
	}
	if denied.Type != "Error" || denied.Payload.Code != 20400 {
		payload, _ := json.Marshal(denied)
		t.Fatalf("unexpected denied event: %s", payload)
	}
}

func TestRealtimeAllowsGroupSubscriptionAndTypingBeyondFirstHundredMembers(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession("user_alice")
	members := make([]string, 0, 121)
	for i := 0; i < 120; i++ {
		members = append(members, fmt.Sprintf("user_member_%03d", i))
	}
	members = append(members, "user_alice")
	handler := NewHandler(
		authService,
		stubUserPresenceService{items: map[string]string{}},
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{}, relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{}},
		stubGroupRepo{members: map[string][]string{"group_big": members}},
	)
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-big", Operation: "set", Type: "group", IDs: []string{"group_big"}}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Payload.Code != 0 || len(ack.Payload.FailedIDs) != 0 {
		payload, _ := json.Marshal(ack)
		t.Fatalf("unexpected ack: %s", payload)
	}
	var snapshot Event
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if snapshot.Type != "SubscriptionsSnapshot" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	_ = conn.WriteJSON(Event{Type: "GroupTyping", Payload: GroupTypingPayload{GroupID: "group_big", IsTyping: true}})
	var typing Event
	if err := conn.ReadJSON(&typing); err != nil {
		t.Fatalf("read typing event: %v", err)
	}
	if typing.Type != "GroupUserTyping" {
		payload, _ := json.Marshal(typing)
		t.Fatalf("unexpected typing event: %s", payload)
	}
}
