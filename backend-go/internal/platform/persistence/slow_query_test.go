package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestSlowQueryTracer_LogsOnlyAboveThreshold(t *testing.T) {
	type report struct {
		sql      string
		duration time.Duration
		err      error
	}
	var got []report
	tr := NewSlowQueryTracer(30*time.Millisecond, func(sql string, d time.Duration, err error) {
		got = append(got, report{sql, d, err})
	})
	if tr == nil {
		t.Fatal("tracer should not be nil for positive threshold")
	}

	// 快查询：不记录。
	fast := tr.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	tr.TraceQueryEnd(fast, nil, pgx.TraceQueryEndData{Err: nil})
	// 慢查询：记录（等待超过阈值）。
	slow := tr.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "SELECT pg_sleep(1)"})
	time.Sleep(40 * time.Millisecond)
	boom := errors.New("boom")
	tr.TraceQueryEnd(slow, nil, pgx.TraceQueryEndData{Err: boom})

	if len(got) != 1 {
		t.Fatalf("reports = %d, want 1 (only the slow query)", len(got))
	}
	if !strings.Contains(got[0].sql, "pg_sleep") {
		t.Fatalf("sql = %q, want it to contain pg_sleep", got[0].sql)
	}
	if got[0].duration < 30*time.Millisecond {
		t.Fatalf("duration = %v, want >= 30ms", got[0].duration)
	}
	if !errors.Is(got[0].err, boom) {
		t.Fatalf("err = %v, want boom", got[0].err)
	}
}

func TestNewSlowQueryTracer_Disabled(t *testing.T) {
	if tr := NewSlowQueryTracer(0, func(string, time.Duration, error) {}); tr != nil {
		t.Fatal("threshold <= 0 should disable tracer")
	}
	if tr := NewSlowQueryTracer(time.Second, nil); tr != nil {
		t.Fatal("nil logf should disable tracer")
	}
}
