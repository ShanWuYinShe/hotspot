package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
)

var errHandlerSentinel = errors.New("handler sentinel failure")

// payloadJSON 是单测用的标准 payload（避免手写转义串）。
const payloadJSON = `{"k":"v"}`

// TestDispatcherRetentionNoopWithoutFactory 覆盖保留期清理在无数据库/无 dispatcher
// 时的降级语义：不 panic、不报错（dispatcher 可能因 DB 未配置而未真正启动）。
func TestDispatcherRetentionNoopWithoutFactory(t *testing.T) {
	var nilDispatcher *Dispatcher
	nilDispatcher.pruneRetentionOnce(context.Background())

	dispatcher := NewDispatcher(nil, nil)
	dispatcher.pruneRetentionOnce(context.Background())
	if _, err := dispatcher.deleteProcessedBatch(context.Background(), time.Now(), 10); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("deleteProcessedBatch err = %v, want ErrNotImplemented", err)
	}
	if _, err := dispatcher.deleteDeadLetterBatch(context.Background(), time.Now(), 10); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("deleteDeadLetterBatch err = %v, want ErrNotImplemented", err)
	}
	if _, err := dispatcher.countDeadLetters(context.Background()); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("countDeadLetters err = %v, want ErrNotImplemented", err)
	}
	// ctx 已取消时快速返回，不发 SQL。
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	dispatcher.pruneRetentionOnce(cancelled)
}

// TestPruneBatches 锁定分批删除循环：未满批即停、满批持续时受单轮批数上限
// 约束、出错时返回已删除数并停止。
func TestPruneBatches(t *testing.T) {
	dispatcher := NewDispatcher(nil, nil)
	cutoff := time.Now().Add(-time.Hour)

	var calls int
	deleted, err := dispatcher.pruneBatches(context.Background(), cutoff, func(context.Context, time.Time, int) (int64, error) {
		calls++
		return 3, nil // 未满批：一轮即停
	})
	if err != nil || deleted != 3 || calls != 1 {
		t.Fatalf("未满批：deleted=%d calls=%d err=%v, want 3/1/nil", deleted, calls, err)
	}

	calls = 0
	deleted, err = dispatcher.pruneBatches(context.Background(), cutoff, func(context.Context, time.Time, int) (int64, error) {
		calls++
		return retentionBatchSize, nil // 始终满批：受单轮批数上限约束
	})
	if err != nil || deleted != int64(retentionBatchSize*retentionMaxBatches) || calls != retentionMaxBatches {
		t.Fatalf("持续满批：deleted=%d calls=%d err=%v, want %d/%d/nil",
			deleted, calls, err, retentionBatchSize*retentionMaxBatches, retentionMaxBatches)
	}

	calls = 0
	boom := errors.New("delete boom")
	deleted, err = dispatcher.pruneBatches(context.Background(), cutoff, func(context.Context, time.Time, int) (int64, error) {
		calls++
		if calls == 2 {
			return 0, boom
		}
		return retentionBatchSize, nil
	})
	if !errors.Is(err, boom) || deleted != int64(retentionBatchSize) || calls != 2 {
		t.Fatalf("出错：deleted=%d calls=%d err=%v, want %d/2/boom", deleted, calls, err, retentionBatchSize)
	}
}

// TestDeadLettersCollectorExposesGauge：死信总数以 Gauge（hotspot_outbox_dead_letters）
// 暴露；nil/未启动的 dispatcher 暴露 0 保持指标序列连续，清扫刷新后的取值直接
// 反映在 /metrics 输出。
func TestDeadLettersCollectorExposesGauge(t *testing.T) {
	render := func(d *Dispatcher) string {
		registry := prometheus.NewRegistry()
		registry.MustRegister(NewDeadLettersCollector(d))
		rec := httptest.NewRecorder()
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		return rec.Body.String()
	}
	if body := render(nil); !strings.Contains(body, "hotspot_outbox_dead_letters 0") {
		t.Fatalf("nil dispatcher 应暴露 0，got:\n%s", body)
	}
	dispatcher := NewDispatcher(nil, nil)
	dispatcher.deadLetters.Store(3)
	if body := render(dispatcher); !strings.Contains(body, "hotspot_outbox_dead_letters 3") {
		t.Fatalf("死信 Gauge 未反映最新取值，got:\n%s", body)
	}
}

// deadLetterStubQueryable 是 Gauge 刷新测试用的 Queryable 桩：只让死信计数查询
// 返回预设值/错误。
type deadLetterStubQueryable struct {
	count int64
	err   error
}

func (s *deadLetterStubQueryable) Exec(context.Context, string, ...any) error {
	return errors.New("unexpected Exec in dead letter count test")
}

func (s *deadLetterStubQueryable) QueryRow(context.Context, string, ...any) persistence.Row {
	return deadLetterStubRow{count: s.count, err: s.err}
}

func (s *deadLetterStubQueryable) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return nil, errors.New("unexpected Query in dead letter count test")
}

type deadLetterStubRow struct {
	count int64
	err   error
}

func (r deadLetterStubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 1 {
		return fmt.Errorf("unexpected scan dest count: %d", len(dest))
	}
	p, ok := dest[0].(*int64)
	if !ok {
		return fmt.Errorf("unexpected scan dest type: %T", dest[0])
	}
	*p = r.count
	return nil
}

// TestRefreshDeadLetterGauge 锁定 Gauge 刷新口径：查询成功即更新；查询失败保留
// 上一次取值（把「查不到」写成 0 会把真实堆积伪装成「没有死信」，让告警静默失效）。
func TestRefreshDeadLetterGauge(t *testing.T) {
	logger, err := observability.NewLogger("test", "error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	dispatcher := NewDispatcher(nil, logger)
	stub := &deadLetterStubQueryable{count: 7, err: errors.New("db down")}
	ctx := persistence.WithTx(context.Background(), stub, nil)

	dispatcher.refreshDeadLetterGauge(ctx)
	if got := dispatcher.deadLetters.Load(); got != 0 {
		t.Fatalf("首次查询失败时不应写入取值，Gauge = %d, want 0", got)
	}

	stub.err = nil
	dispatcher.refreshDeadLetterGauge(ctx)
	if got := dispatcher.deadLetters.Load(); got != 7 {
		t.Fatalf("查询成功后 Gauge = %d, want 7", got)
	}

	// 之后的查询失败不得把已取到的值覆盖成 0。
	stub.err = errors.New("db down again")
	dispatcher.refreshDeadLetterGauge(ctx)
	if got := dispatcher.deadLetters.Load(); got != 7 {
		t.Fatalf("查询失败后 Gauge = %d, want 保留 7", got)
	}
}

// TestDispatcherInvokeHandlerRecoversPanic 锁定 handler panic 的转换语义：
// 必须返回错误（而不是让 panic 逃逸掀掉领取事务），错误信息要带上 panic 内容，
// 便于 dead-letter 的 last_error 定位毒事件。真实库口径（attempts 递增、事务
// 不回滚、10 次进死信、后续事件不停投递）见 outbox_it_test.go。
func TestDispatcherInvokeHandlerRecoversPanic(t *testing.T) {
	dispatcher := NewDispatcher(nil, nil)
	row := outboxRow{eventID: 7, topic: "test.panic", payload: json.RawMessage(payloadJSON), attempts: 3}

	t.Run("正常返回保持 nil", func(t *testing.T) {
		got := dispatcher.invokeHandler(context.Background(), func(_ context.Context, payload json.RawMessage) error {
			if string(payload) != payloadJSON {
				t.Fatalf("payload = %s, want %s", payload, payloadJSON)
			}
			return nil
		}, row)
		if got != nil {
			t.Fatalf("err = %v, want nil", got)
		}
	})

	t.Run("普通 error 原样透传", func(t *testing.T) {
		got := dispatcher.invokeHandler(context.Background(), func(context.Context, json.RawMessage) error {
			return errHandlerSentinel
		}, row)
		if !errors.Is(got, errHandlerSentinel) {
			t.Fatalf("err = %v, want sentinel 透传", got)
		}
	})

	t.Run("字符串 panic 转 error", func(t *testing.T) {
		got := dispatcher.invokeHandler(context.Background(), func(context.Context, json.RawMessage) error {
			panic("boom")
		}, row)
		if got == nil {
			t.Fatal("handler panic 必须转成 error，不能返回 nil")
		}
		if !strings.Contains(got.Error(), "boom") || !strings.Contains(got.Error(), "test.panic") {
			t.Fatalf("err = %q, 应同时包含 panic 值与 topic", got.Error())
		}
	})

	t.Run("error 值 panic 转 error", func(t *testing.T) {
		got := dispatcher.invokeHandler(context.Background(), func(context.Context, json.RawMessage) error {
			panic(errHandlerSentinel)
		}, row)
		if got == nil || !strings.Contains(got.Error(), errHandlerSentinel.Error()) {
			t.Fatalf("err = %v, 应包含 panic error 信息", got)
		}
	})
}
