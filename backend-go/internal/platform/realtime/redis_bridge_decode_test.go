package realtime

import (
	"strings"
	"testing"
)

// decodeEnvelope 契约：常规 Publish（无 "s"）原样透传；组合脚本信封（顶层 "s"）
// 回填 ServerSeq；非法 JSON 报错。新旧格式永久并存（NoRecord 事件走常规格式）。
func TestDecodeEnvelope(t *testing.T) {
	// 组合脚本格式（Lua 手工拼接）：o/s/e 三字段
	combined := `{"o":"hs-a","s":42,"e":{"type":"private_message_new","payload":{}}}`
	envelope, err := decodeEnvelope([]byte(combined))
	if err != nil {
		t.Fatalf("decode combined: %v", err)
	}
	if envelope.Origin != "hs-a" || envelope.Seq != 42 {
		t.Fatalf("envelope: %#v", envelope)
	}
	if envelope.Event.Type != "private_message_new" || envelope.Event.ServerSeq != 42 {
		t.Fatalf("event: %#v", envelope.Event)
	}

	// 常规 Publish 格式（无 "s"）——Event 原样，ServerSeq 不被改写
	legacy := `{"o":"hs-b","e":{"type":"private_typing"}}`
	envelope, err = decodeEnvelope([]byte(legacy))
	if err != nil {
		t.Fatalf("decode legacy: %v", err)
	}
	if envelope.Seq != 0 || envelope.Event.ServerSeq != 0 || envelope.Event.Type != "private_typing" {
		t.Fatalf("legacy envelope must pass through untouched: %#v", envelope)
	}

	// 非法 JSON
	if _, err := decodeEnvelope([]byte("not-json")); err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if _, err := decodeEnvelope([]byte(strings.Repeat("x", 8))); err == nil {
		t.Fatal("expected error for garbage payload")
	}
}
