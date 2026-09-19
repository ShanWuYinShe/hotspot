package realtime

import (
	"testing"
)

// TestResumeStreamTrimmedToMaxLen 回归：持续写入的活跃流必须被裁剪到上限，
// 否则 PEXPIRE 每次续期使热流永不过期、无界增长。写入上限+100 条后回放，
// 断言仅保留最新的 resumeStreamMaxLen 条（seq 连续、无空洞）。
func TestResumeStreamTrimmedToMaxLen(t *testing.T) {
	store, _ := newMiniredisStore(t)
	const extra = 100
	total := resumeStreamMaxLen + extra
	for i := 0; i < total; i++ {
		if seq := store.Record("u_trim", Event{Type: "t"}); seq <= 0 {
			t.Fatalf("record %d: seq=%d", i, seq)
		}
	}
	events := store.Replay("u_trim", nil, 0)
	if len(events) != resumeStreamMaxLen {
		t.Fatalf("expected %d retained events, got %d", resumeStreamMaxLen, len(events))
	}
	first := int64(total - resumeStreamMaxLen + 1)
	for i, ev := range events {
		if want := first + int64(i); ev.ServerSeq != want {
			t.Fatalf("event %d: expected seq %d, got %d", i, want, ev.ServerSeq)
		}
	}
}
