package admin

import (
	"encoding/json"
	"testing"
)

func decodeReportWSPayload(t *testing.T, payload any) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("payload is not an object: %v (%s)", err, raw)
	}
	return fields
}

func assertStringField(t *testing.T, fields map[string]json.RawMessage, key, want string) {
	t.Helper()
	var got string
	if err := json.Unmarshal(fields[key], &got); err != nil {
		t.Fatalf("field %q is not a string: %s", key, fields[key])
	}
	if got != want {
		t.Fatalf("field %q = %q, want %q", key, got, want)
	}
}

// resolved 事件：note 为字符串时原样下发；键集合与 spec 一致（6 键）。
func TestBuildReportHandledWSPayloadResolvedWithNote(t *testing.T) {
	fields := decodeReportWSPayload(t, BuildReportHandledWSPayload(
		"n1", "r1", "moment", "m1", "违规内容已处理", true, "2026-11-28T00:00:00Z",
	))
	if len(fields) != 6 {
		t.Fatalf("payload keys = %d, want 6", len(fields))
	}
	assertStringField(t, fields, "notify_id", "n1")
	assertStringField(t, fields, "report_id", "r1")
	assertStringField(t, fields, "target_type", "moment")
	assertStringField(t, fields, "target_id", "m1")
	assertStringField(t, fields, "resolved_at", "2026-11-28T00:00:00Z")
	assertStringField(t, fields, "note", "违规内容已处理")
}

// dismissed 事件且无说明：note 必须是显式 null（恒定 nullable，不能缺键）。
func TestBuildReportHandledWSPayloadDismissedWithoutNote(t *testing.T) {
	fields := decodeReportWSPayload(t, BuildReportHandledWSPayload(
		"n2", "r2", "comment", "c1", "", false, "2026-11-28T01:00:00Z",
	))
	if len(fields) != 6 {
		t.Fatalf("payload keys = %d, want 6", len(fields))
	}
	if _, ok := fields["resolved_at"]; ok {
		t.Fatal("dismissed payload must not contain resolved_at")
	}
	assertStringField(t, fields, "dismissed_at", "2026-11-28T01:00:00Z")
	if string(fields["note"]) != "null" {
		t.Fatalf("note = %s, want null", fields["note"])
	}
}
