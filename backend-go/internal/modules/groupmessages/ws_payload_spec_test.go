package groupmessages

// ws_payload_spec_test.go — 群聊 NewMessage WS payload 的 JSON 形状契约
//（spec api_spec_websocket.md GroupMessageNewPayload）：content/share_code/
// reply_to/media 恒定 nullable，mentions 恒为数组（空数组 = 无 @）。

import (
	"encoding/json"
	"testing"

	"hotspot/backend-go/internal/platform/realtime"
)

type groupMessagePayloadSink struct{ events []realtime.Event }

func (s *groupMessagePayloadSink) Send(e realtime.Event) error {
	s.events = append(s.events, e)
	return nil
}
func (s *groupMessagePayloadSink) Close() error { return nil }

func assertGroupMessagePayloadKeys(t *testing.T, payload any, wantKeys ...string) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("payload is not a JSON object: %v (%s)", err, raw)
	}
	if len(fields) != len(wantKeys) {
		t.Fatalf("payload keys = %v, want %v (raw: %s)", payloadKeyNames(fields), wantKeys, raw)
	}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Fatalf("payload missing key %q (raw: %s)", key, raw)
		}
	}
	return fields
}

func payloadKeyNames(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	return keys
}

// TestGroupMessageNewPayloadAlwaysCarriesNullableKeys：纯文本且无 @ 的群消息
// 帧必须携带 content，且 share_code/reply_to/media 为 null、mentions 为 []
// （契约审计 B07：此前 map 条件写键导致帧缺键）。
func TestGroupMessageNewPayloadAlwaysCarriesNullableKeys(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := &stubRepo{sendResult: MessageSentResult{MessageID: "gm-1", SentAt: "2025-06-01T00:00:00Z"}}
	service := NewServiceWithRepo(repo)
	sink := &groupMessagePayloadSink{}
	hub.SetGroupSubscriptions([]string{"g1"}, sink)
	if _, err := service.Send(gmAuthCtx("u1"), SendGroupMessageRequest{GroupID: "g1", Type: "text", Content: "hi"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %#v", sink.events)
	}
	payload, ok := sink.events[0].Payload.(GroupMessageSentPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[0].Payload)
	}
	fields := assertGroupMessagePayloadKeys(t, payload,
		"message_id", "group_id", "sender", "type", "content", "share_code", "reply_to", "mentions", "media", "sent_at")
	if string(fields["content"]) != `"hi"` {
		t.Fatalf("unexpected content: %s", fields["content"])
	}
	for _, key := range []string{"share_code", "reply_to", "media"} {
		if string(fields[key]) != "null" {
			t.Fatalf("%s must be null for a text-only message, got %s", key, fields[key])
		}
	}
	if string(fields["mentions"]) != "[]" {
		t.Fatalf("mentions must be [] when no one is mentioned, got %s", fields["mentions"])
	}
}

// TestGroupMessageNewPayloadCarriesMentionsArray：有 @ 时 mentions 为 user_id 数组。
func TestGroupMessageNewPayloadCarriesMentionsArray(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := &stubRepo{sendResult: MessageSentResult{MessageID: "gm-2", SentAt: "2025-06-01T00:00:00Z"}}
	service := NewServiceWithRepo(repo)
	sink := &groupMessagePayloadSink{}
	hub.SetGroupSubscriptions([]string{"g1"}, sink)
	if _, err := service.Send(gmAuthCtx("u1"), SendGroupMessageRequest{GroupID: "g1", Type: "text", Content: "@u2 hi", Mentions: []string{"u2"}}); err != nil {
		t.Fatalf("send: %v", err)
	}
	payload, ok := sink.events[0].Payload.(GroupMessageSentPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[0].Payload)
	}
	if len(payload.Mentions) != 1 || payload.Mentions[0] != "u2" {
		t.Fatalf("unexpected mentions: %#v", payload.Mentions)
	}
}
