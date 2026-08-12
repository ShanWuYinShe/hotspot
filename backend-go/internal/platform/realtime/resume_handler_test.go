package realtime

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"

	"github.com/gorilla/websocket"
)

// fakeResumeStore 是 ResumeStore 的内存实现，用于 handler resume 流程的端到端测试
// （验证回放、token 下发、降级、越权 token 拒绝等业务逻辑，不依赖 Redis）。
type fakeResumeStore struct {
	mu     sync.Mutex
	seq    int64
	events []Event
	states map[string]ResumeState
}

func newFakeResumeStore() *fakeResumeStore {
	return &fakeResumeStore{states: map[string]ResumeState{}}
}

func (f *fakeResumeStore) Record(_ string, event Event) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	event.ServerSeq = f.seq
	f.events = append(f.events, event)
	return f.seq
}

func (f *fakeResumeStore) Replay(_ string, fromSeq int64) []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Event, 0)
	for _, e := range f.events {
		if e.ServerSeq > fromSeq {
			out = append(out, e)
		}
	}
	return out
}

func (f *fakeResumeStore) LastSeq(string) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seq
}

func (f *fakeResumeStore) SaveState(token string, state ResumeState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[token] = state
}

func (f *fakeResumeStore) LoadState(token string) (ResumeState, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.states[token]
	return s, ok
}

func (f *fakeResumeStore) Close() error { return nil }

// readEventWithTimeout 读取一个 WS 事件，2s 超时则 t.Fatal（用于断言「应有事件」）。
func readEventWithTimeout(t *testing.T, conn *websocket.Conn) Event {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	var ev Event
	if err := conn.ReadJSON(&ev); err != nil {
		t.Fatalf("read event: %v", err)
	}
	return ev
}

// TestConnectionEstablishedHasResumeToken：store 配置时，CE 下发非空 resume_token
// 与 server_seq（spec L133：ConnectionEstablished 含 resume_token 与 server_seq）。
func TestConnectionEstablishedHasResumeToken(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	fake := newFakeResumeStore()
	handler := newRealtimeHandlerForTest(authService)
	handler.SetResumeStore(fake)
	DefaultHub().SetResumeStore(fake)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	ce := readEventWithTimeout(t, conn)
	if ce.Type != EventTypeConnectionEstablished {
		t.Fatalf("want ConnectionEstablished, got %s", ce.Type)
	}
	payload := ce.Payload.(map[string]any)
	if token, _ := payload["resume_token"].(string); token == "" {
		t.Fatal("resume_token should be non-empty when store is configured")
	}
	if seq, ok := payload["server_seq"].(float64); !ok || seq < 0 {
		t.Fatalf("server_seq missing or negative: %v", payload["server_seq"])
	}
}

// TestResumeReplayOnReconnect：A 连接收事件→断开→断线期间新事件→B 带 token 重连，
// 收到断线期间事件的有序回放（spec L156：按 server_seq 续传）。
func TestResumeReplayOnReconnect(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	fake := newFakeResumeStore()
	handler := newRealtimeHandlerForTest(authService)
	handler.SetResumeStore(fake)
	DefaultHub().SetResumeStore(fake)
	server := httptest.NewServer(handler)
	defer server.Close()

	// 连接 A（首次，不带 token）
	connA, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial A: %v", err)
	}
	if err := connA.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth A: %v", err)
	}
	ceA := readEventWithTimeout(t, connA)
	token1 := ceA.Payload.(map[string]any)["resume_token"].(string)

	// A 在线期间收到 2 条直达事件
	DefaultHub().BroadcastToUser("user_alice", Event{Type: EventTypePrivateMessageNew, Payload: map[string]any{"message_id": "m1"}})
	DefaultHub().BroadcastToUser("user_alice", Event{Type: EventTypePrivateMessageNew, Payload: map[string]any{"message_id": "m2"}})
	readEventWithTimeout(t, connA) // m1
	readEventWithTimeout(t, connA) // m2

	// 断开 A，等 defer 存 resume state（UnregisterUser + SaveState）
	if err := connA.Close(); err != nil {
		t.Fatalf("close A: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := fake.LoadState(token1); ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := fake.LoadState(token1); !ok {
		t.Fatal("resume state not saved after disconnect")
	}

	// A 断线期间新事件到达（Record 入流，无在线 sink 不直达）
	DefaultHub().BroadcastToUser("user_alice", Event{Type: EventTypePrivateMessageNew, Payload: map[string]any{"message_id": "m3"}})
	DefaultHub().BroadcastToUser("user_alice", Event{Type: EventTypePrivateMessageNew, Payload: map[string]any{"message_id": "m4"}})

	// B 带 token1 重连，应收回放
	connB, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial B: %v", err)
	}
	defer connB.Close()
	if err := connB.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken, ResumeToken: token1}}); err != nil {
		t.Fatalf("write auth B: %v", err)
	}
	ceB := readEventWithTimeout(t, connB) // ConnectionEstablished
	if ceB.Type != EventTypeConnectionEstablished {
		t.Fatalf("want CE on reconnect, got %s", ceB.Type)
	}
	ev1 := readEventWithTimeout(t, connB) // 回放 m3
	ev2 := readEventWithTimeout(t, connB) // 回放 m4
	m1 := ev1.Payload.(map[string]any)["message_id"].(string)
	m2 := ev2.Payload.(map[string]any)["message_id"].(string)
	if m1 != "m3" || m2 != "m4" {
		t.Fatalf("replay want m3,m4; got %s,%s", m1, m2)
	}
	if ev1.ServerSeq <= 0 || ev2.ServerSeq <= ev1.ServerSeq {
		t.Fatalf("replayed events should have ascending seq: %d then %d", ev1.ServerSeq, ev2.ServerSeq)
	}
}

// TestResumeDegradesWithoutStore：store 未注入（Redis 不可用），token 仍下发但
// 重连不回放（spec L161 降级保证：resume 故障→客户端全量重放仍可用）。
func TestResumeDegradesWithoutStore(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	handler := newRealtimeHandlerForTest(authService) // 不注入 store
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	ce := readEventWithTimeout(t, conn)
	token := ce.Payload.(map[string]any)["resume_token"].(string)
	if token == "" {
		t.Fatal("token should still be issued in degraded mode")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // 等断开处理

	// 重连带 token：store nil，无回放
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial 2: %v", err)
	}
	defer conn2.Close()
	if err := conn2.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken, ResumeToken: token}}); err != nil {
		t.Fatalf("write auth 2: %v", err)
	}
	readEventWithTimeout(t, conn2) // CE
	// 之后应无回放事件（超时读失败为期望）
	if err := conn2.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	var extra Event
	if err := conn2.ReadJSON(&extra); err == nil {
		t.Fatalf("expected no replay in degraded mode, got %+v", extra)
	}
}

// TestResumeIgnoresForeignToken：token 有效但不属于当前 user → 不回放（防越权续传他人事件）。
func TestResumeIgnoresForeignToken(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	bobResult, err := authService.IssueSession(context.Background(), "user_bob")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	fake := newFakeResumeStore()
	fake.states["foreign-tok"] = ResumeState{UserID: "user_charlie", LastSeq: 10} // 他人 token
	handler := newRealtimeHandlerForTest(authService)
	handler.SetResumeStore(fake)
	DefaultHub().SetResumeStore(fake)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: bobResult.AccessToken, ResumeToken: "foreign-tok"}}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	ce := readEventWithTimeout(t, conn)
	if ce.Type != EventTypeConnectionEstablished {
		t.Fatalf("want CE, got %s", ce.Type)
	}
	if err := conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	var extra Event
	if err := conn.ReadJSON(&extra); err == nil {
		t.Fatalf("foreign token should not trigger replay, got %+v", extra)
	}
}
