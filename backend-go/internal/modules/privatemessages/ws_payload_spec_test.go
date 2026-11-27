package privatemessages

// ws_payload_spec_test.go — 私聊 WS payload 的 JSON 形状契约（spec
// api_spec_websocket.md）。content/share_code/reply_to/media 为恒定 nullable，
// 无值时必须显式下发 null；PrivateMessageRead 不携带冗余 target_user_id/reader_id。

import (
	"context"
	"encoding/json"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

func assertPrivateMessagePayloadKeys(t *testing.T, payload any, wantKeys ...string) map[string]json.RawMessage {
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
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		t.Fatalf("payload keys = %v, want %v (raw: %s)", keys, wantKeys, raw)
	}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Fatalf("payload missing key %q (raw: %s)", key, raw)
		}
	}
	return fields
}

// TestPrivateMessageNewPayloadAlwaysCarriesNullableKeys：纯文本消息帧必须
// 携带 content 字符串，且 share_code/reply_to/media 三个键恒在、值为 null
// （契约审计 B07：此前 omitempty 让帧缺键）。
func TestPrivateMessageNewPayloadAlwaysCarriesNullableKeys(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	}
	service := NewServiceWithDeps(NewInMemoryRepository(), userStub)
	sink := &eventSink{}
	hub.RegisterUser("user_bob", "sess_bob", sink)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	if _, err := service.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "hello"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %#v", sink.events)
	}
	payload, ok := sink.events[0].Payload.(PrivateMessageSentPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[0].Payload)
	}
	fields := assertPrivateMessagePayloadKeys(t, payload,
		"message_id", "target_user_id", "sender", "type", "content", "share_code", "reply_to", "media", "sent_at")
	if string(fields["content"]) != `"hello"` {
		t.Fatalf("unexpected content: %s", fields["content"])
	}
	for _, key := range []string{"share_code", "reply_to", "media"} {
		if string(fields[key]) != "null" {
			t.Fatalf("%s must be null for a text-only message, got %s", key, fields[key])
		}
	}
}

// TestPrivateMessageReadPayloadJSONShape：spec 明确私聊已读回执无需冗余
// target_user_id/reader_id（接收方天然知道对端身份）。
func TestPrivateMessageReadPayloadJSONShape(t *testing.T) {
	fields := assertPrivateMessagePayloadKeys(t, PrivateMessageReadPayload{
		UpToMessageID: "msg_1",
		ReadAt:        "2026-11-28T00:00:00Z",
	}, "up_to_message_id", "read_at")
	if string(fields["up_to_message_id"]) != `"msg_1"` {
		t.Fatalf("unexpected up_to_message_id: %s", fields["up_to_message_id"])
	}
}
