package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 审计 perf P1-4：组合脚本单次往返完成「seq 分配 + Stream 写入 + 跨实例发布」。
// miniredis 提供真实 Lua/PubSub 语义，订阅端捕获发布载荷验证信封契约：
// {"o":origin,"s":seq,"e":<event>}，且 s 能经 decodeEnvelope 回填 ServerSeq。
// waitMessage 收订阅消息：跳过 Subscription 确认帧，返回首条真实消息。
func waitMessage(t *testing.T, sub *redis.PubSub) *redis.Message {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		msg, err := sub.ReceiveTimeout(context.Background(), time.Second)
		if err != nil {
			t.Fatalf("receive: %v", err)
		}
		if m, ok := msg.(*redis.Message); ok {
			return m
		}
	}
	t.Fatal("no pubsub message within 3s")
	return nil
}

func TestRecordUserAndPublishSingleRoundTrip(t *testing.T) {
	store, mr := newMiniredisStoreWithOrigin(t, "hs-a")

	pub := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = pub.Close() })
	sub := pub.Subscribe(context.Background(), "hotspot:user:u1")
	defer func() { _ = sub.Close() }()

	seq, published := store.RecordUserAndPublish("u1", "user:u1", Event{Type: EventTypePrivateMessageNew})
	if !published {
		t.Fatal("combined record+publish should report published=true")
	}
	if seq <= 0 {
		t.Fatalf("seq must be positive, got %d", seq)
	}

	// Stream 落库验证（resume 语义不变）
	got := store.Replay("u1", nil, 0)
	if len(got) != 1 || got[0].Type != EventTypePrivateMessageNew || got[0].ServerSeq != seq {
		t.Fatalf("stream replay mismatch: %#v", got)
	}

	// 发布载荷验证（跨实例信封契约）
	payload := waitMessage(t, sub)
	envelope, err := decodeEnvelope([]byte(payload.Payload))
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Origin != "hs-a" {
		t.Fatalf("origin: got %q, want hs-a", envelope.Origin)
	}
	if envelope.Seq != seq {
		t.Fatalf("envelope seq: got %d, want %d", envelope.Seq, seq)
	}
	if envelope.Event.Type != EventTypePrivateMessageNew {
		t.Fatalf("envelope event type: got %q", envelope.Event.Type)
	}
	if envelope.Event.ServerSeq != seq {
		t.Fatalf("decodeEnvelope should backfill ServerSeq, got %d", envelope.Event.ServerSeq)
	}
	// miniredis 的 Lua 执行由其内置解释器完成，此处再以手拼 JSON 做一次
	// 契约形态自证（防止未来信封字段漂移而 miniredis 恰好兼容）。
	var probe struct {
		Origin string          `json:"o"`
		Seq    int64           `json:"s"`
		Event  json.RawMessage `json:"e"`
	}
	if err := json.Unmarshal([]byte(payload.Payload), &probe); err != nil {
		t.Fatalf("envelope is not the documented {o,s,e} shape: %v", err)
	}
	if probe.Seq != seq || probe.Origin != "hs-a" || len(probe.Event) == 0 {
		t.Fatalf("envelope fields mismatch: %+v", probe)
	}
}

func TestRecordGroupAndPublishSingleRoundTrip(t *testing.T) {
	store, mr := newMiniredisStoreWithOrigin(t, "hs-a")

	pub := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = pub.Close() })
	sub := pub.Subscribe(context.Background(), "hotspot:group:g1")
	defer func() { _ = sub.Close() }()

	seq, published := store.RecordGroupAndPublish("g1", "group:g1", Event{Type: EventTypeGroupMessageNew})
	if !published || seq <= 0 {
		t.Fatalf("combined group path: published=%v seq=%d", published, seq)
	}
	// group 流落库（XRANGE 直查，与 user 流同一 seq 空间）
	vals, err := store.rdb.XRange(context.Background(), "resume:group:g1", "1", "+").Result()
	if err != nil || len(vals) != 1 {
		t.Fatalf("group stream should hold 1 entry, got %d entries err=%v", len(vals), err)
	}
	if seqFromStreamID(vals[0].ID) != seq {
		t.Fatalf("stream entry id %q should equal seq %d", vals[0].ID, seq)
	}
	msg := waitMessage(t, sub)
	envelope, err := decodeEnvelope([]byte(msg.Payload))
	if err != nil || envelope.Seq != seq || envelope.Event.Type != EventTypeGroupMessageNew {
		t.Fatalf("group envelope mismatch: env=%#v err=%v", envelope, err)
	}
}

// 空 origin（未配置）与脚本失败必须回退两步路径：published=false，seq=0。
func TestRecordAndPublishDegradedPaths(t *testing.T) {
	noOrigin, _ := newMiniredisStoreWithOrigin(t, "")
	if seq, published := noOrigin.RecordUserAndPublish("u1", "user:u1", Event{Type: "x"}); published || seq != 0 {
		t.Fatalf("empty origin should not publish: seq=%d published=%v", seq, published)
	}

	// 脚本失败（关闭 miniredis 使连接拒绝）
	store, mr := newMiniredisStoreWithOrigin(t, "hs-a")
	mr.Close()
	if seq, published := store.RecordUserAndPublish("u1", "user:u1", Event{Type: "x"}); published || seq != 0 {
		t.Fatalf("failed script should degrade: seq=%d published=%v", seq, published)
	}
	if _, published := store.RecordGroupAndPublish("g1", "group:g1", Event{Type: "x"}); published {
		t.Fatal("failed script should degrade (group path)")
	}
}
