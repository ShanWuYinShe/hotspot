package realtime

import (
	"context"
	"net/http/httptest"
	"strings"
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
	mu      sync.Mutex
	seq     int64
	events  []Event // 按 Record/RecordGroup 顺序，ServerSeq 已填
	states  map[string]ResumeState
	byUser  map[string][]Event
	byGroup map[string][]Event
}

func newFakeResumeStore() *fakeResumeStore {
	return &fakeResumeStore{
		states:  map[string]ResumeState{},
		byUser:  map[string][]Event{},
		byGroup: map[string][]Event{},
	}
}

func (f *fakeResumeStore) record(userID, groupID string, event Event) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	event.ServerSeq = f.seq
	f.events = append(f.events, event)
	if groupID != "" {
		f.byGroup[groupID] = append(f.byGroup[groupID], event)
	} else {
		f.byUser[userID] = append(f.byUser[userID], event)
	}
	return f.seq
}

func (f *fakeResumeStore) Record(userID string, event Event) int64 {
	return f.record(userID, "", event)
}

func (f *fakeResumeStore) RecordGroup(groupID string, event Event) int64 {
	return f.record("", groupID, event)
}

// Replay 与 RedisResumeStore 同语义：聚合 user 流 + groupIDs 各 group 流，
// 按全局 seq 升序归并。
func (f *fakeResumeStore) Replay(userID string, groupIDs []string, fromSeq int64) []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	streams := [][]Event{filterSeq(f.byUser[userID], fromSeq)}
	for _, gid := range groupIDs {
		streams = append(streams, filterSeq(f.byGroup[gid], fromSeq))
	}
	return mergeEvents(streams)
}

func filterSeq(events []Event, fromSeq int64) []Event {
	out := make([]Event, 0, len(events))
	for _, e := range events {
		if e.ServerSeq > fromSeq {
			out = append(out, e)
		}
	}
	return out
}

func (f *fakeResumeStore) LastSeq() int64 {
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
	// spec L330：server_seq 跨端为 string（JS/ArkTS 无 int64，number 有 2^53 精度上限），
	// 后端 int64 经 `,string` 序列化为 JSON 字符串。
	if seq, ok := payload["server_seq"].(string); !ok || seq == "" {
		t.Fatalf("server_seq should be a non-empty string, got %v", payload["server_seq"])
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

// newRealtimeHandlerForTestWithGroups 与 newRealtimeHandlerForTest 相同，但允许
// 指定 group 成员表（stubGroupRepo）。members 是 map 引用，测试中修改其内容会
// 反映到 handler 内部（用于模拟断线期间退出群）。
func newRealtimeHandlerForTestWithGroups(authService *platformauth.Service, members map[string][]string) *Handler {
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
		stubGroupRepo{members: members},
	)
}

// waitResumeStateSaved 轮询等待 token 的 resume state 落库（断线 defer 异步保存）。
func waitResumeStateSaved(t *testing.T, fake *fakeResumeStore, token string) ResumeState {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if state, ok := fake.LoadState(token); ok {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("resume state not saved after disconnect")
	return ResumeState{}
}

// TestResumeReplayGroupEventsOnReconnect：A 订阅群 g1 → 断开 → 断线期间群事件与
// 私聊事件交错到达 → B 带 token 重连：服务端按快照恢复订阅，聚合 user 流 + group
// 流按全局 seq 回放（spec L156：按 server_seq 续传 + 订阅由服务端恢复）。
func TestResumeReplayGroupEventsOnReconnect(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	fake := newFakeResumeStore()
	handler := newRealtimeHandlerForTestWithGroups(authService, map[string][]string{"g1": {"user_alice"}})
	handler.SetResumeStore(fake)
	DefaultHub().SetResumeStore(fake)
	server := httptest.NewServer(handler)
	defer server.Close()

	// 连接 A：订阅群 g1
	connA, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial A: %v", err)
	}
	if err := connA.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth A: %v", err)
	}
	ceA := readEventWithTimeout(t, connA)
	token1 := ceA.Payload.(map[string]any)["resume_token"].(string)
	if err := connA.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{Operation: "subscribe", Items: []SubscriptionItem{{Type: "group", IDs: []string{"g1"}}}}}); err != nil {
		t.Fatalf("write subscribe A: %v", err)
	}
	readEventWithTimeout(t, connA) // SubscribeAck
	readEventWithTimeout(t, connA) // SubscriptionsSnapshot

	// 断开 A，等 defer 保存含订阅快照的 state
	if err := connA.Close(); err != nil {
		t.Fatalf("close A: %v", err)
	}
	state := waitResumeStateSaved(t, fake, token1)
	if len(state.Subs.GroupIDs) != 1 || state.Subs.GroupIDs[0] != "g1" {
		t.Fatalf("saved state should include group subscription snapshot: %+v", state.Subs)
	}

	// 断线期间：群事件（seq 先）与私聊事件交错到达
	DefaultHub().BroadcastToGroup("g1", Event{Type: EventTypeGroupMessageNew, Payload: map[string]any{"message_id": "gm1"}})
	DefaultHub().BroadcastToUser("user_alice", Event{Type: EventTypePrivateMessageNew, Payload: map[string]any{"message_id": "pm1"}})

	// B 带 token1 重连：CE 之后按 seq 回放 gm1、pm1
	connB, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial B: %v", err)
	}
	defer connB.Close()
	if err := connB.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken, ResumeToken: token1}}); err != nil {
		t.Fatalf("write auth B: %v", err)
	}
	if ce := readEventWithTimeout(t, connB); ce.Type != EventTypeConnectionEstablished {
		t.Fatalf("want CE on reconnect, got %s", ce.Type)
	}
	ev1 := readEventWithTimeout(t, connB)
	ev2 := readEventWithTimeout(t, connB)
	if ev1.Type != EventTypeGroupMessageNew || ev1.Payload.(map[string]any)["message_id"] != "gm1" {
		t.Fatalf("replay[0] want group gm1, got %s %v", ev1.Type, ev1.Payload)
	}
	if ev2.Type != EventTypePrivateMessageNew || ev2.Payload.(map[string]any)["message_id"] != "pm1" {
		t.Fatalf("replay[1] want private pm1, got %s %v", ev2.Type, ev2.Payload)
	}
	if ev1.ServerSeq <= 0 || ev2.ServerSeq <= ev1.ServerSeq {
		t.Fatalf("replayed events should have ascending global seq: %d then %d", ev1.ServerSeq, ev2.ServerSeq)
	}

	// 服务端已按快照恢复订阅：向 g1 广播新事件应直达 B（无需 B 再发 SetSubscriptions）
	DefaultHub().BroadcastToGroup("g1", Event{Type: EventTypeGroupMessageNew, Payload: map[string]any{"message_id": "gm2"}})
	ev3 := readEventWithTimeout(t, connB)
	if ev3.Type != EventTypeGroupMessageNew || ev3.Payload.(map[string]any)["message_id"] != "gm2" {
		t.Fatalf("restored subscription should deliver gm2, got %s %v", ev3.Type, ev3.Payload)
	}
}

// TestResumeReplaySkipsDepartedGroup：断线期间用户退出群 g1，重连恢复订阅时
// isGroupMember 校验失败 → g1 不恢复订阅、不聚合回放（防越权接收）。
func TestResumeReplaySkipsDepartedGroup(t *testing.T) {
	SetDefaultHub(NewHub())
	authService := platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"})
	result, err := authService.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	fake := newFakeResumeStore()
	members := map[string][]string{"g1": {"user_alice"}}
	handler := newRealtimeHandlerForTestWithGroups(authService, members)
	handler.SetResumeStore(fake)
	DefaultHub().SetResumeStore(fake)
	server := httptest.NewServer(handler)
	defer server.Close()

	connA, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial A: %v", err)
	}
	if err := connA.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken}}); err != nil {
		t.Fatalf("write auth A: %v", err)
	}
	ceA := readEventWithTimeout(t, connA)
	token1 := ceA.Payload.(map[string]any)["resume_token"].(string)
	if err := connA.WriteJSON(Event{Type: "SetSubscriptions", Payload: SetSubscriptionsPayload{Operation: "subscribe", Items: []SubscriptionItem{{Type: "group", IDs: []string{"g1"}}}}}); err != nil {
		t.Fatalf("write subscribe A: %v", err)
	}
	readEventWithTimeout(t, connA)
	readEventWithTimeout(t, connA)
	if err := connA.Close(); err != nil {
		t.Fatalf("close A: %v", err)
	}
	waitResumeStateSaved(t, fake, token1)

	// 断线期间退出群 g1（map 引用共享，handler 内 stub 同步可见）
	members["g1"] = nil
	DefaultHub().BroadcastToGroup("g1", Event{Type: EventTypeGroupMessageNew, Payload: map[string]any{"message_id": "gm1"}})
	DefaultHub().BroadcastToUser("user_alice", Event{Type: EventTypePrivateMessageNew, Payload: map[string]any{"message_id": "pm1"}})

	// B 重连：只回放私聊 pm1；gm1 因成员校验失败被跳过
	connB, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial B: %v", err)
	}
	defer connB.Close()
	if err := connB.WriteJSON(Event{Type: "Auth", Payload: AuthPayload{ClientType: "app", AccessToken: result.AccessToken, ResumeToken: token1}}); err != nil {
		t.Fatalf("write auth B: %v", err)
	}
	if ce := readEventWithTimeout(t, connB); ce.Type != EventTypeConnectionEstablished {
		t.Fatalf("want CE on reconnect, got %s", ce.Type)
	}
	ev := readEventWithTimeout(t, connB)
	if ev.Type != EventTypePrivateMessageNew || ev.Payload.(map[string]any)["message_id"] != "pm1" {
		t.Fatalf("replay[0] want private pm1 only, got %s %v", ev.Type, ev.Payload)
	}
	// 之后不应有 g1 的回放事件
	if err := connB.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	var extra Event
	if err := connB.ReadJSON(&extra); err == nil {
		t.Fatalf("departed group should not be replayed, got %+v", extra)
	}
}

// TestServerSeqSerializedAsString：spec L330 契约——server_seq 跨端为 JSON 字符串
// （JS/ArkTS 的 number 无法无损表示 int64，安全上限 2^53），后端 int64 经
// `,string` 序列化下发。断言原始 JSON 形态，防止回退为数字序列化。
func TestServerSeqSerializedAsString(t *testing.T) {
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

	// CE：server_seq 无 omitempty，初始为 0 也下发，JSON 形态应为字符串 "0"
	if _, raw, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read CE: %v", err)
	} else if !strings.Contains(string(raw), `"server_seq":"0"`) {
		t.Fatalf("CE server_seq should serialize as JSON string \"0\", got %s", raw)
	}

	// 广播事件：server_seq 为字符串 "1"
	DefaultHub().BroadcastToUser("user_alice", Event{Type: EventTypePrivateMessageNew, Payload: map[string]any{"message_id": "m1"}})
	if _, raw, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read event: %v", err)
	} else if !strings.Contains(string(raw), `"server_seq":"1"`) {
		t.Fatalf("event server_seq should serialize as JSON string \"1\", got %s", raw)
	}
}
