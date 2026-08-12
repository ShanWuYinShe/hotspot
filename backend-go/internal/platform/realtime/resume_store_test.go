package realtime

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/alicebob/miniredis/v2"
)

// newMiniredisStore 启动一个内存 Redis（miniredis）并返回基于它的 RedisResumeStore。
// miniredis 提供真实 Redis 语义（Streams/Lua/TTL），使 ResumeStore 的 Redis 交互
// 可在单测内验证（项目此前 realtime 的 redis 层零覆盖，本测试顺带补齐基础）。
func newMiniredisStore(t *testing.T) (*RedisResumeStore, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRedisResumeStore(rdb, 0), s // ttl=0 → DefaultResumeTTL(60s)
}

func TestRedisResumeStoreRecordReplaySeqMonotonic(t *testing.T) {
	store, _ := newMiniredisStore(t)
	seq1 := store.Record("u1", Event{Type: EventTypePrivateMessageNew})
	seq2 := store.Record("u1", Event{Type: EventTypePrivateMessageEdited})
	seq3 := store.Record("u1", Event{Type: EventTypePrivateMessageRecalled})
	if seq1 != 1 || seq2 != 2 || seq3 != 3 {
		t.Fatalf("seq not monotonic from 1: got %d,%d,%d", seq1, seq2, seq3)
	}
	// Replay(seq1) → seq>1 的事件（seq2/seq3），按序，ServerSeq 由 entry ID 反推补回。
	got := store.Replay("u1", seq1)
	if len(got) != 2 {
		t.Fatalf("replay want 2 events, got %d", len(got))
	}
	if got[0].ServerSeq != 2 || got[0].Type != EventTypePrivateMessageEdited {
		t.Fatalf("replay[0] = seq=%d type=%s, want seq=2 Edited", got[0].ServerSeq, got[0].Type)
	}
	if got[1].ServerSeq != 3 || got[1].Type != EventTypePrivateMessageRecalled {
		t.Fatalf("replay[1] = seq=%d type=%s, want seq=3 Recalled", got[1].ServerSeq, got[1].Type)
	}
	if seq := store.LastSeq("u1"); seq != 3 {
		t.Fatalf("LastSeq = %d, want 3", seq)
	}
}

func TestRedisResumeStorePerUserIsolation(t *testing.T) {
	store, _ := newMiniredisStore(t)
	store.Record("u1", Event{Type: "A"})
	store.Record("u2", Event{Type: "B"})
	store.Record("u1", Event{Type: "C"})
	if got := store.Replay("u1", 0); len(got) != 2 {
		t.Fatalf("u1 want 2 events, got %d", len(got))
	}
	if got := store.Replay("u2", 0); len(got) != 1 {
		t.Fatalf("u2 want 1 event, got %d", len(got))
	}
}

func TestRedisResumeStoreStateRoundTrip(t *testing.T) {
	store, _ := newMiniredisStore(t)
	state := ResumeState{UserID: "u1", LastSeq: 42, Subs: SubscriptionsSnapshotPayload{UserIDs: []string{"a"}}}
	store.SaveState("tok-1", state)
	got, ok := store.LoadState("tok-1")
	if !ok {
		t.Fatal("LoadState ok=false after SaveState")
	}
	if got.UserID != "u1" || got.LastSeq != 42 || len(got.Subs.UserIDs) != 1 {
		t.Fatalf("LoadState round-trip mismatch: %+v", got)
	}
	if _, ok := store.LoadState("nope"); ok {
		t.Fatal("LoadState unknown token should return ok=false")
	}
}

func TestRedisResumeStoreStateTTLExpiry(t *testing.T) {
	store, s := newMiniredisStore(t)
	store.SaveState("tok-1", ResumeState{UserID: "u1"})
	s.FastForward(2 * time.Minute) // 超过默认 60s TTL
	if _, ok := store.LoadState("tok-1"); ok {
		t.Fatal("state should expire after TTL")
	}
}

func TestRedisResumeStoreStreamTTLExpiry(t *testing.T) {
	store, s := newMiniredisStore(t)
	store.Record("u1", Event{Type: "A"})
	s.FastForward(2 * time.Minute) // Stream 与 seq key 均被 PEXPIRE 续期 60s
	if got := store.Replay("u1", 0); len(got) != 0 {
		t.Fatalf("stream should expire after TTL, got %d events", len(got))
	}
}

func TestRedisResumeStoreEmptyUserID(t *testing.T) {
	store, _ := newMiniredisStore(t)
	if seq := store.Record("", Event{Type: "A"}); seq != 0 {
		t.Fatalf("Record('') want 0, got %d", seq)
	}
	if got := store.Replay("", 0); got != nil {
		t.Fatalf("Replay('') want nil, got %v", got)
	}
}

func TestNopResumeStoreDegrades(t *testing.T) {
	var s NopResumeStore
	if seq := s.Record("u1", Event{Type: "A"}); seq != 0 {
		t.Fatalf("Nop Record want 0, got %d", seq)
	}
	if got := s.Replay("u1", 0); got != nil {
		t.Fatalf("Nop Replay want nil, got %v", got)
	}
	if _, ok := s.LoadState("tok"); ok {
		t.Fatal("Nop LoadState want ok=false")
	}
}

// TestRedisResumeStoreCrossInstance 验证 spec L159 核心硬前提：WS 多实例部署时，
// 用户重连可能落到与断线时不同的实例——两个实例共享同一 Redis，A 实例 Record 的
// 事件能被 B 实例 Replay 读到、A 存的 state 能被 B 加载。单实例内存方案必然失败。
func TestRedisResumeStoreCrossInstance(t *testing.T) {
	s := miniredis.RunT(t)
	rdbA := redis.NewClient(&redis.Options{Addr: s.Addr()})
	rdbB := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdbA.Close(); _ = rdbB.Close() })
	storeA := NewRedisResumeStore(rdbA, 0)
	storeB := NewRedisResumeStore(rdbB, 0)

	// 实例 A 上 user 产生事件（写入共享 Stream）
	seq1 := storeA.Record("u1", Event{Type: EventTypePrivateMessageNew})
	seq2 := storeA.Record("u1", Event{Type: EventTypePrivateMessageEdited})

	// 客户端重连落到实例 B，从共享 Stream 回放
	got := storeB.Replay("u1", 0)
	if len(got) != 2 {
		t.Fatalf("cross-instance replay want 2 events, got %d", len(got))
	}
	if got[0].ServerSeq != seq1 || got[1].ServerSeq != seq2 {
		t.Fatalf("cross-instance seq mismatch: got %d,%d want %d,%d", got[0].ServerSeq, got[1].ServerSeq, seq1, seq2)
	}
	// A 存的 resume state，B 能读到（token 跨实例有效）
	storeA.SaveState("tok", ResumeState{UserID: "u1", LastSeq: seq2})
	state, ok := storeB.LoadState("tok")
	if !ok || state.LastSeq != seq2 {
		t.Fatalf("cross-instance state read failed: %+v ok=%v", state, ok)
	}
}
