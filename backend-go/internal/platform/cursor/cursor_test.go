package cursor

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	type payload struct {
		ID   string `json:"id"`
		Page int    `json:"page"`
	}
	in := payload{ID: "u1", Page: 2}
	encoded, err := Encode("users", in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var out payload
	if err := Decode("users", encoded, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out != in {
		t.Fatalf("expected %#v, got %#v", in, out)
	}
}

func TestDecodeRejectsWrongNamespace(t *testing.T) {
	encoded, err := Encode("users", map[string]any{"id": "u1"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var out map[string]any
	if err := Decode("groups", encoded, &out); err != ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestDecodeEmptyValueNoop(t *testing.T) {
	out := map[string]any{"keep": "value"}
	if err := Decode("users", "", &out); err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if out["keep"] != "value" {
		t.Fatalf("expected original map preserved, got %#v", out)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	var out map[string]any
	if err := Decode("users", "not-a-valid-cursor", &out); err != ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}
