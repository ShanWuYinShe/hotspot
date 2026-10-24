package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisResumeStore 的边界/降级语义（正常路径见 resume_store_test.go）。
// resume 是断线续传的正确性底座：seq 必须全局单调、回放窗口/顺序正确、
// 损坏数据与 Redis 故障必须降级而不是 panic/阻塞。
func TestRedisResumeStoreLastSeqAndReplayWindow(t *testing.T) {
	store, _ := newMiniredisStore(t)
	if got := store.LastSeq(); got != 0 {
		t.Fatalf("empty store LastSeq = %d, want 0", got)
	}
	if events := store.Replay("u1", []string{"g1"}, 0); events != nil {
		t.Fatalf("empty replay = %#v, want nil", events)
	}

	// user 流与 group 流交错写入，seq 必须共享同一全局计数器。
	seqUser1 := store.Record("u1", Event{Type: EventTypePrivateMessageNew})
	seqGroup := store.RecordGroup("g1", Event{Type: EventTypeGroupMessageNew})
	seqUser2 := store.Record("u1", Event{Type: EventTypeNewFollower})
	seqOtherUser := store.Record("u2", Event{Type: EventTypeUserOnline})
	if !(seqUser1 < seqGroup && seqGroup < seqUser2 && seqUser2 < seqOtherUser) {
		t.Fatalf("seq not globally monotonic: %d %d %d %d", seqUser1, seqGroup, seqUser2, seqOtherUser)
	}
	if got := store.LastSeq(); got != seqOtherUser {
		t.Fatalf("LastSeq = %d, want %d", got, seqOtherUser)
	}

	// fromSeq=0：user+group 全量事件按全局 seq 升序归并；其它用户的事件不得泄漏。
	all := store.Replay("u1", []string{"g1"}, 0)
	if len(all) != 3 {
		t.Fatalf("replay all = %d events, want 3: %#v", len(all), all)
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ServerSeq >= all[i].ServerSeq {
			t.Fatalf("replay not strictly increasing: %d then %d", all[i-1].ServerSeq, all[i].ServerSeq)
		}
	}
	if all[0].ServerSeq != seqUser1 || all[1].ServerSeq != seqGroup || all[2].ServerSeq != seqUser2 {
		t.Fatalf("replay order = %v", []int64{all[0].ServerSeq, all[1].ServerSeq, all[2].ServerSeq})
	}
	// 只回放 > fromSeq 的事件（续传语义）；fromSeq 为最新 → nil。
	if events := store.Replay("u1", []string{"g1"}, seqUser1); len(events) != 2 {
		t.Fatalf("replay after seq1 = %d events, want 2", len(events))
	}
	if events := store.Replay("u1", []string{"g1"}, store.LastSeq()); events != nil {
		t.Fatalf("replay at head = %#v, want nil", events)
	}
	// 负 fromSeq 视同从头回放（防御性容错）。
	if events := store.Replay("u1", []string{"g1"}, -5); len(events) != 3 {
		t.Fatalf("negative fromSeq replay = %d events, want 3", len(events))
	}
	// 重复 groupID 不得重复回放（实现按订阅上限去重前不应放大事件量）。
	if events := store.Replay("u1", []string{"g1", "g1"}, 0); len(events) != 3 {
		t.Fatalf("duplicate group replay = %d events, want 3", len(events))
	}
}

func TestRedisResumeStoreReplaySkipsCorruptEntries(t *testing.T) {
	store, mr := newMiniredisStore(t)
	rdb := store.rdb
	ctx := context.Background()
	key := store.streamKey("u1")
	// 手工写入三类脏 entry：缺 "e" 字段、非法 JSON、合法 JSON。
	if err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: key, ID: "1", Values: map[string]any{"bad": "x"}}).Err(); err != nil {
		t.Fatalf("xadd missing e: %v", err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: key, ID: "2", Values: map[string]any{"e": "{not-json"}}).Err(); err != nil {
		t.Fatalf("xadd invalid json: %v", err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: key, ID: "3", Values: map[string]any{"e": `{"Type":"NewFollower"}`}}).Err(); err != nil {
		t.Fatalf("xadd valid: %v", err)
	}
	events := store.Replay("u1", nil, 0)
	if len(events) != 1 || events[0].Type != EventTypeNewFollower || events[0].ServerSeq != 3 {
		t.Fatalf("corrupt stream replay = %#v", events)
	}
	_ = mr
}

func TestRedisResumeStoreCorruptStateTreatedAsMissing(t *testing.T) {
	store, mr := newMiniredisStore(t)
	// 损坏 JSON：LoadState 必须返回 ok=false（回退全量），而不是 panic。
	mr.Set(store.stateKey("tok-corrupt"), "{not-json")
	if _, ok := store.LoadState("tok-corrupt"); ok {
		t.Fatal("corrupt state must be treated as missing")
	}
	if _, ok := store.LoadState(""); ok {
		t.Fatal("empty token must miss")
	}
	if _, ok := store.LoadState("tok-missing"); ok {
		t.Fatal("missing token must miss")
	}
	// 正常 roundtrip 对照。
	state := ResumeState{UserID: "u1", LastSeq: 7}
	store.SaveState("tok-ok", state)
	got, ok := store.LoadState("tok-ok")
	if !ok || got.UserID != "u1" || got.LastSeq != 7 {
		t.Fatalf("state roundtrip = %#v ok=%v", got, ok)
	}
}

func TestRedisResumeStoreBackendDownDegrades(t *testing.T) {
	store, mr := newMiniredisStore(t)
	mr.Close() // Redis 不可用

	// 所有写路径返回 0/false（不阻塞广播），读路径返回空，不 panic。
	if seq := store.Record("u1", Event{Type: EventTypeNewFollower}); seq != 0 {
		t.Fatalf("record while down = %d, want 0", seq)
	}
	if seq := store.RecordGroup("g1", Event{Type: EventTypeGroupMessageNew}); seq != 0 {
		t.Fatalf("record group while down = %d, want 0", seq)
	}
	if seq, ok := store.RecordUserAndPublish("u1", "user:u1", Event{Type: EventTypeNewFollower}); seq != 0 || ok {
		t.Fatalf("record+publish while down = %d/%v, want 0/false", seq, ok)
	}
	if seq, ok := store.RecordGroupAndPublish("g1", "group:g1", Event{Type: EventTypeGroupMessageNew}); seq != 0 || ok {
		t.Fatalf("group record+publish while down = %d/%v, want 0/false", seq, ok)
	}
	if events := store.Replay("u1", []string{"g1"}, 0); events != nil {
		t.Fatalf("replay while down = %#v, want nil", events)
	}
	if got := store.LastSeq(); got != 0 {
		t.Fatalf("LastSeq while down = %d, want 0", got)
	}
	store.SaveState("tok", ResumeState{UserID: "u1"})
	if _, ok := store.LoadState("tok"); ok {
		t.Fatal("LoadState while down must miss")
	}
}

func TestResumeStoreCloseDoesNotCloseSharedClient(t *testing.T) {
	store, _ := newMiniredisStore(t)
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// 共享 *redis.Client 生命周期归 bootstrap：Close 后仍应可用。
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := store.rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("shared redis client must stay usable after Close: %v", err)
	}
}

func TestRecordAndPublishRequiresOrigin(t *testing.T) {
	store, mr := newMiniredisStoreWithOrigin(t, "")
	if seq, ok := store.RecordUserAndPublish("u1", "user:u1", Event{Type: EventTypeNewFollower}); seq != 0 || ok {
		t.Fatalf("empty origin must not publish: %d/%v", seq, ok)
	}
	if seq, ok := store.RecordGroupAndPublish("g1", "group:g1", Event{Type: EventTypeGroupMessageNew}); seq != 0 || ok {
		t.Fatalf("empty origin group must not publish: %d/%v", seq, ok)
	}
	// 但普通 Record 仍工作（不依赖 origin）。
	if seq := store.Record("u1", Event{Type: EventTypeNewFollower}); seq == 0 {
		t.Fatal("plain Record must work without origin")
	}
	_ = mr
}
