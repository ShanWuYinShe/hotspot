package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

var errHandlerSentinel = errors.New("handler sentinel failure")

// payloadJSON 是单测用的标准 payload（避免手写转义串）。
const payloadJSON = `{"k":"v"}`

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
