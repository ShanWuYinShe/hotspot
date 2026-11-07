package realtime

import "testing"

// TestRedisResumeStoreReplayBatch：单次回放最多 limit 条、hasMore 正确，按
// fromSeq 推进入多批可完整回放全部事件且 seq 连续。
func TestRedisResumeStoreReplayBatch(t *testing.T) {
	store, _ := newMiniredisStore(t)
	const total = 120
	for i := 0; i < total; i++ {
		store.Record("u_batch", Event{Type: EventTypePrivateMessageNew})
	}
	const limit = 50
	fromSeq := int64(0)
	seen := 0
	for {
		events, hasMore := store.ReplayBatch("u_batch", nil, fromSeq, limit)
		if len(events) > limit {
			t.Fatalf("batch returned %d events, limit %d", len(events), limit)
		}
		for _, ev := range events {
			seen++
			if ev.ServerSeq != int64(seen) {
				t.Fatalf("event seq = %d, want %d", ev.ServerSeq, seen)
			}
		}
		if len(events) == 0 {
			t.Fatalf("empty batch before drain: seen=%d hasMore=%v", seen, hasMore)
		}
		fromSeq = events[len(events)-1].ServerSeq
		if !hasMore {
			break
		}
	}
	if seen != total {
		t.Fatalf("drained %d events, want %d", seen, total)
	}
}
