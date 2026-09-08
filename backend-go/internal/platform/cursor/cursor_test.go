package cursor

import (
	"encoding/base64"
	"testing"
)

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

func TestDecodeStringRoundTrip(t *testing.T) {
	codec := NewCodec("test-ns")
	rawMsg := "hello-world-cursor-123"
	encoded, err := codec.Encode(rawMsg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := codec.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode string failed: %v", err)
	}
	if decoded != `"`+rawMsg+`"` {
		t.Fatalf("unexpected decoded string: got %q, want %q", decoded, `"`+rawMsg+`"`)
	}

	// 测试空字符串
	emptyDecoded, err := codec.DecodeString("")
	if err != nil || emptyDecoded != "" {
		t.Fatalf("decode empty string failed: got %q, err=%v", emptyDecoded, err)
	}

	// 测试非法 base64
	_, err = codec.DecodeString("!@#not-valid-base64-url#@!")
	if err != ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor for bad base64, got %v", err)
	}

	// 测试长度小于 nonceSize
	shortBlob := base64.RawURLEncoding.EncodeToString([]byte("short"))
	_, err = codec.DecodeString(shortBlob)
	if err != ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor for short payload, got %v", err)
	}
}

func TestDecodeRejectsTamperedCiphertext(t *testing.T) {
	codec := NewCodec("tamper-test")
	payload := map[string]string{"secret": "sensitive-data"}
	encoded, err := codec.Encode(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// 解码原始字节并篡改其中一个字节
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // 翻转最后一位

	tampered := base64.RawURLEncoding.EncodeToString(raw)
	var out map[string]string
	if err := codec.Decode(tampered, &out); err != ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor on tampered ciphertext, got %v", err)
	}
}

func TestDecodeTypeMismatch(t *testing.T) {
	codec := NewCodec("mismatch-ns")
	encoded, err := codec.Encode("just-a-plain-string")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var num int
	if err := codec.Decode(encoded, &num); err != ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor when unmarshaling string to int, got %v", err)
	}
}

func TestEncodeUnserializablePayload(t *testing.T) {
	codec := NewCodec("err-ns")
	// channel 无法进行 json.Marshal
	ch := make(chan int)
	_, err := codec.Encode(ch)
	if err == nil {
		t.Fatal("expected error encoding channel, got nil")
	}
}
