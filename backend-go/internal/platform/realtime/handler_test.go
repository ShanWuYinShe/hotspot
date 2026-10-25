package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/ratelimit"

	"github.com/gorilla/websocket"
)

func wsURL(serverURL string) string { return "ws" + strings.TrimPrefix(serverURL, "http") + "/ws/v1" }

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

func (s stubUserRepo) BlockedEitherWay(_ context.Context, aUserID, bUserID string) (callerBlocked, targetBlocked bool, err error) {
	if s.relations[aUserID] == nil {
		return false, false, nil
	}
	return s.relations[aUserID][bUserID].isBlocked, false, nil
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

func (s stubGroupRepo) IsMember(_ context.Context, groupID, userID string) (bool, error) {
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
	result, err := authService.IssueSession(context.Background(), "user_alice")
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

	if err := conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req1", Operation: "set", Items: []SubscriptionItem{{Type: "user", IDs: []string{"user_bob"}}}}}); err != nil {
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

	otherResult, _ := authService.IssueSession(context.Background(), "user_bob")
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

// stubPresenceWriter 记录 presence 写调用序列，供 presence 落库时机断言。
type stubPresenceWriter struct {
	mu     sync.Mutex
	calls  []string // 形如 "user_alice=online"
	failOn string   // 命中该 user_id 时返回错误（验证写失败不中断连接）
}

func (s *stubPresenceWriter) SetPresence(_ context.Context, userID string, online bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := "offline"
	if online {
		status = "online"
	}
	s.calls = append(s.calls, userID+"="+status)
	if s.failOn == userID {
		return fmt.Errorf("db down")
	}
	return nil
}

func (s *stubPresenceWriter) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
}

// TestRealtimePresenceWrittenOnFirstConnectAndLastDisconnect 验证 presence 落库
// 时机：首条连接 online 一次、最后一条连接断开 offline 一次；多连接共存时中间
// 建连/断开不写；写失败不影响连接生命周期。
func TestRealtimePresenceWrittenOnFirstConnectAndLastDisconnect(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	handler := newRealtimeHandlerForTest(authService)
	presence := &stubPresenceWriter{failOn: "user_alice"} // 故意写失败：不得中断连接
	handler.SetPresenceWriter(presence)
	server := httptest.NewServer(handler)
	defer server.Close()

	dial := func() *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
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
		return conn
	}

	conn1 := dial()
	if got := presence.snapshot(); len(got) != 1 || got[0] != "user_alice=online" {
		t.Fatalf("after first connect, presence calls = %v, want [user_alice=online]", got)
	}

	// 第二条连接（同 user 多端）：不重复写 online。
	conn2 := dial()
	if got := presence.snapshot(); len(got) != 1 {
		t.Fatalf("after second connect, presence calls = %v, want unchanged", got)
	}

	// 断开第一条：仍有连接存活，不写 offline。
	_ = conn1.Close()
	waitForCalls(t, presence, 1)

	// 断开最后一条：写 offline（写失败仅记日志）。
	_ = conn2.Close()
	waitForCalls(t, presence, 2)
	if got := presence.snapshot(); got[1] != "user_alice=offline" {
		t.Fatalf("after last disconnect, presence calls = %v, want second call user_alice=offline", got)
	}
}

// waitForCalls 轮询等待 presence 调用数达到 want（断连 defer 异步执行）。
func waitForCalls(t *testing.T, presence *stubPresenceWriter, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if len(presence.snapshot()) >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d presence calls, got %v", want, presence.snapshot())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// countingUserRepo 统计 GetSettings 调用次数，供 typing 节流前置断言使用。
type countingUserRepo struct {
	stubUserRepo
	getSettingsCalls int32
}

func (c *countingUserRepo) GetSettings(ctx context.Context, userID string) (bool, string, bool, error) {
	atomic.AddInt32(&c.getSettingsCalls, 1)
	return c.stubUserRepo.GetSettings(ctx, userID)
}

// TestRealtimePrivateTypingThrottleSkipsDBCheck 验证节流前置（审计 perf P2-2）：
// 3s 窗口内的后续 typing 帧直接丢弃、不再触发 DB 鉴权；首帧仍走 DB 校验。
func TestRealtimePrivateTypingThrottleSkipsDBCheck(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	repo := &countingUserRepo{stubUserRepo: stubUserRepo{
		settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{},
		relations: map[string]map[string]struct {
			isFollowing  bool
			isFollowedBy bool
			isBlocked    bool
		}{},
	}}
	handler := NewHandler(authService, repo, stubGroupRepo{members: map[string][]string{}})
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

	frame := Event{Type: "SendPrivateTyping", Payload: PrivateTypingSendPayload{TargetUserID: "user_bob", IsTyping: true}}
	if err := conn.WriteJSON(frame); err != nil {
		t.Fatalf("write first typing: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&repo.getSettingsCalls) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	before := atomic.LoadInt32(&repo.getSettingsCalls)
	if before == 0 {
		t.Fatal("首帧应触发 DB 鉴权")
	}

	// 3s 窗口内的第二帧：必须被节流（DB 调用数不再增长）。
	if err := conn.WriteJSON(frame); err != nil {
		t.Fatalf("write second typing: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	after := atomic.LoadInt32(&repo.getSettingsCalls)
	if after != before {
		t.Fatalf("被节流帧不应触发 DB 鉴权: before=%d after=%d", before, after)
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
	result, _ := authService.IssueSession(context.Background(), "user_alice")
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req2", Operation: "set", Items: []SubscriptionItem{{Type: "user", IDs: ids}}}})
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
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.RateLimits.WSSubscribePerSecond = 1
		return cfg
	})
	ratelimit.SetSource(source)
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_alice")
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "conn-1", Operation: "set", Items: []SubscriptionItem{{Type: "user", IDs: []string{"user_bob"}}}}})
	var ack Event
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	var snapshot Event
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "conn-1", Operation: "set", Items: []SubscriptionItem{{Type: "user", IDs: []string{"user_charlie"}}}}})
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
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	handler := NewHandler(
		authService,
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-hidden", Operation: "set", Items: []SubscriptionItem{{Type: "user", IDs: []string{"user_hidden"}}}}})
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

// TestRealtimeSetSubscriptionsMixedItems 验证 spec 批量结构：单请求可混合
// user/group 两类订阅项，SubscribeAck 按 type 分组回传 accepted，failed_ids
// 逐项携带 type（客户端按 type+subscription_id 回滚本地订阅）。
func TestRealtimeSetSubscriptionsMixedItems(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	handler := NewHandler(
		authService,
		stubUserRepo{settings: map[string]struct {
			presenceVisible  bool
			allowPrivateChat string
		}{"user_hidden": {presenceVisible: false}}, relations: map[string]map[string]struct {
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{
		ClientRequestID: "req-mixed",
		Operation:       "set",
		Items: []SubscriptionItem{
			{Type: "user", IDs: []string{"user_bob", "user_hidden"}},
			{Type: "group", IDs: []string{"group_1"}},
		},
	}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.Type != "SubscribeAck" || ack.Payload.Code != 20100 {
		payload, _ := json.Marshal(ack)
		t.Fatalf("unexpected ack: %s", payload)
	}
	if len(ack.Payload.Accepted) != 1 || ack.Payload.Accepted[0].Type != "user" ||
		len(ack.Payload.Accepted[0].IDs) != 1 || ack.Payload.Accepted[0].IDs[0] != "user_bob" {
		payload, _ := json.Marshal(ack)
		t.Fatalf("accepted should group user_bob only: %s", payload)
	}
	if len(ack.Payload.FailedIDs) != 2 {
		payload, _ := json.Marshal(ack)
		t.Fatalf("want 2 failed ids, got: %s", payload)
	}
	types := map[string]string{}
	for _, f := range ack.Payload.FailedIDs {
		types[f.SubscriptionID] = f.Type
	}
	if types["user_hidden"] != "user" || types["group_1"] != "group" {
		payload, _ := json.Marshal(ack)
		t.Fatalf("failed ids should carry type: %s", payload)
	}
	var snapshot Event
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	snap := snapshot.Payload.(map[string]any)
	if userIDs, ok := snap["user_ids"].([]any); !ok || len(userIDs) != 1 || userIDs[0] != "user_bob" {
		payload, _ := json.Marshal(snapshot)
		t.Fatalf("snapshot should contain only user_bob: %s", payload)
	}
}

func TestRealtimeRejectsGroupSubscriptionForNonMember(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	handler := NewHandler(
		authService,
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-group", Operation: "set", Items: []SubscriptionItem{{Type: "group", IDs: []string{"group_1"}}}}})
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
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	handler := NewHandler(
		authService,
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-unsub", Operation: "unsubscribe", Items: []SubscriptionItem{{Type: "user", IDs: []string{"user_hidden"}}}}})
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
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	handler := NewHandler(
		authService,
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
	_ = conn.WriteJSON(Event{Type: "SendPrivateTyping", Payload: PrivateTypingSendPayload{TargetUserID: "user_bob", IsTyping: true}})
	var denied struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&denied); err != nil {
		t.Fatalf("read denied event: %v", err)
	}
	if denied.Type != "Error" || denied.Payload.Code != 20100 {
		payload, _ := json.Marshal(denied)
		t.Fatalf("unexpected denied event: %s", payload)
	}
}

func TestRealtimeRejectsGroupTypingForNonMember(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	handler := NewHandler(
		authService,
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
	_ = conn.WriteJSON(Event{Type: "SendGroupTyping", Payload: GroupTypingSendPayload{GroupID: "group_1", IsTyping: true}})
	var denied struct {
		Type    string       `json:"type"`
		Payload ErrorPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&denied); err != nil {
		t.Fatalf("read denied event: %v", err)
	}
	if denied.Type != "Error" || denied.Payload.Code != 20100 {
		payload, _ := json.Marshal(denied)
		t.Fatalf("unexpected denied event: %s", payload)
	}
}

func TestRealtimeAllowsGroupSubscriptionAndTypingBeyondFirstHundredMembers(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_alice")
	members := make([]string, 0, 121)
	for i := 0; i < 120; i++ {
		members = append(members, fmt.Sprintf("user_member_%03d", i))
	}
	members = append(members, "user_alice")
	handler := NewHandler(
		authService,
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
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{ClientRequestID: "req-big", Operation: "set", Items: []SubscriptionItem{{Type: "group", IDs: []string{"group_big"}}}}})
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
	_ = conn.WriteJSON(Event{Type: "SendGroupTyping", Payload: GroupTypingSendPayload{GroupID: "group_big", IsTyping: true}})
	var typing Event
	if err := conn.ReadJSON(&typing); err != nil {
		t.Fatalf("read typing event: %v", err)
	}
	if typing.Type != "GroupTyping" {
		payload, _ := json.Marshal(typing)
		t.Fatalf("unexpected typing event: %s", payload)
	}
}

func TestSameOriginAllowsAbsentAndMatching(t *testing.T) {
	// Non-browser clients send no Origin header — allow.
	if !sameOrigin(httptest.NewRequest(http.MethodGet, "https://host.example/ws", nil)) {
		t.Fatal("absent Origin should be allowed (native client)")
	}
	// Same-origin browser request — allow.
	req := httptest.NewRequest(http.MethodGet, "https://host.example/ws", nil)
	req.Header.Set("Origin", "https://host.example")
	if !sameOrigin(req) {
		t.Fatal("same-origin should be allowed")
	}
}

func TestSameOriginAllowsNullOriginForCookieLessNativeClient(t *testing.T) {
	// HarmonyOS 等原生 WebSocket 客户端发送 Origin: null 且不带 Cookie —— 放行，
	// 鉴权由升级后的 Auth 事件令牌完成。
	req := httptest.NewRequest(http.MethodGet, "https://host.example/ws", nil)
	req.Header.Set("Origin", "null")
	if !sameOrigin(req) {
		t.Fatal("null Origin without cookies should be allowed (native client)")
	}

	// 带 Cookie 的 null Origin 视为跨站浏览器请求（CSWSH 向量）—— 拒绝。
	req = httptest.NewRequest(http.MethodGet, "https://host.example/ws", nil)
	req.Header.Set("Origin", "null")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "x"})
	if sameOrigin(req) {
		t.Fatal("null Origin with cookies must be blocked (cross-site browser)")
	}
}

func TestSameOriginAllowsHarmonyDuplicateOriginWithoutCookies(t *testing.T) {
	// 鸿蒙原生 WebSocket 实测同时发送 OS 默认的 `Origin: http://host`（丢端口）
	// 与客户端补的 `origin: http://host:port`；浏览器禁止设置 Origin，出现多个值只可能是原生客户端。
	req := httptest.NewRequest(http.MethodGet, "http://192.168.0.100:8888/ws/v1", nil)
	req.Host = "192.168.0.100:8888"
	req.Header.Add("Origin", "http://192.168.0.100")
	req.Header.Add("Origin", "http://192.168.0.100:8888")
	if !sameOrigin(req) {
		t.Fatal("duplicate same-host Origin without cookies should be allowed (native client)")
	}

	// 带 Cookie 的多 Origin 请求按跨站浏览器处理 —— 拒绝。
	req = httptest.NewRequest(http.MethodGet, "http://192.168.0.100:8888/ws/v1", nil)
	req.Host = "192.168.0.100:8888"
	req.Header.Add("Origin", "http://192.168.0.100")
	req.Header.Add("Origin", "http://192.168.0.100:8888")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "x"})
	if sameOrigin(req) {
		t.Fatal("duplicate Origin with cookies must be blocked")
	}

	// 主机名不一致的多 Origin —— 拒绝。
	req = httptest.NewRequest(http.MethodGet, "http://192.168.0.100:8888/ws/v1", nil)
	req.Host = "192.168.0.100:8888"
	req.Header.Add("Origin", "http://192.168.0.100")
	req.Header.Add("Origin", "http://evil.example:8888")
	if sameOrigin(req) {
		t.Fatal("duplicate Origin with mismatched host must be blocked")
	}
}

func TestSameOriginBlocksCrossOrigin(t *testing.T) {
	// Cross-origin browser request (CSWSH vector) — block.
	req := httptest.NewRequest(http.MethodGet, "https://host.example/ws", nil)
	req.Header.Set("Origin", "https://evil.example")
	if sameOrigin(req) {
		t.Fatal("cross-origin should be blocked")
	}
	// Unparseable origin — block.
	req = httptest.NewRequest(http.MethodGet, "https://host.example/ws", nil)
	req.Header.Set("Origin", "://bad")
	if sameOrigin(req) {
		t.Fatal("malformed origin should be blocked")
	}
}

// 回归：单条订阅消息可携带多个同类型 item，逐 item 裁剪后累加仍会突破
// 连接级上限——总量截断必须在汇总侧生效（spec：max_subscriptions 500/100）。
func TestRealtimeSubscriptionTotalLimitAcrossItems(t *testing.T) {
	SetDefaultHub(NewHub())
	ratelimit.SetSource(appconfig.NewStaticSource(config.Config{}))
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_alice")
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

	unique := func(prefix string, n int) []string {
		ids := make([]string, n)
		for i := range ids {
			ids[i] = fmt.Sprintf("%s_%d", prefix, i)
		}
		return ids
	}
	_ = conn.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{
		ClientRequestID: "req-multi",
		Operation:       "set",
		Items: []SubscriptionItem{
			{Type: "user", IDs: unique("user_a", 400)},
			{Type: "user", IDs: unique("user_b", 400)},
		},
	}})
	var ack struct {
		Type    string              `json:"type"`
		Payload SubscribeAckPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	var snapshot struct {
		Type    string                       `json:"type"`
		Payload SubscriptionsSnapshotPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if ack.Type != "SubscribeAck" || ack.Payload.Code == 0 {
		payload, _ := json.Marshal(ack)
		t.Fatalf("expected partial-failure ack, got: %s", payload)
	}
	accepted := 0
	for _, item := range ack.Payload.Accepted {
		if item.Type != "user" {
			t.Fatalf("unexpected accepted type %q", item.Type)
		}
		accepted += len(item.IDs)
	}
	if accepted != maxUserSubscriptionsPerConnection {
		t.Fatalf("expected exactly %d accepted user subscriptions, got %d", maxUserSubscriptionsPerConnection, accepted)
	}
	failed := 0
	for _, f := range ack.Payload.FailedIDs {
		if f.Reason != "limit_exceeded" {
			t.Fatalf("unexpected failure reason %q", f.Reason)
		}
		failed++
	}
	if failed != 300 {
		t.Fatalf("expected 300 limit_exceeded failures, got %d", failed)
	}
	if len(snapshot.Payload.UserIDs) != maxUserSubscriptionsPerConnection {
		t.Fatalf("expected snapshot bounded to %d users, got %d", maxUserSubscriptionsPerConnection, len(snapshot.Payload.UserIDs))
	}
}

type stubBanAuthorizer struct {
	banned  bool
	details func() (BanDetails, bool, error)
}

func (s stubBanAuthorizer) IsBanned(context.Context, string) (bool, error) { return s.banned, nil }
func (s stubBanAuthorizer) BanDetails(context.Context, string) (BanDetails, bool, error) {
	if s.details == nil {
		return BanDetails{}, false, nil
	}
	return s.details()
}

// 回归：WS 握手封禁拒绝必须按 spec 走「推送 AccountBanned 事件 + close 4001」，
// 不得使用禁用的应用层错误码（20401）或未定义的 close code（4003）。
func TestRealtimeHandshakeRejectsBannedWithSpecCloseCode(t *testing.T) {
	SetDefaultHub(NewHub())
	ratelimit.SetSource(appconfig.NewStaticSource(config.Config{}))
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, _ := authService.IssueSession(context.Background(), "user_banned")
	handler := newRealtimeHandlerForTest(authService)
	until := time.Now().UTC().Add(72 * time.Hour)
	handler.SetBanAuthorizer(stubBanAuthorizer{banned: true, details: func() (BanDetails, bool, error) {
		return BanDetails{Reason: "违规内容", DurationDays: 3, BannedAt: time.Now().UTC().Add(-time.Hour), BannedUntil: &until}, true, nil
	}})
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}})
	var banned struct {
		Type    string               `json:"type"`
		Payload AccountBannedPayload `json:"payload"`
	}
	if err := conn.ReadJSON(&banned); err != nil {
		t.Fatalf("read account banned event: %v", err)
	}
	if banned.Type != "AccountBanned" {
		t.Fatalf("expected AccountBanned event, got %q", banned.Type)
	}
	if banned.Payload.Reason != "违规内容" || banned.Payload.DurationDays != 3 || banned.Payload.BannedUntil == nil {
		t.Fatalf("unexpected payload: %#v", banned.Payload)
	}
	_, _, err = conn.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) || closeErr.Code != 4001 {
		t.Fatalf("expected close code 4001, got %v", err)
	}
}
