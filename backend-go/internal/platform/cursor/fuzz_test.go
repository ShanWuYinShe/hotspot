package cursor

import (
	"encoding/base64"
	"errors"
	"testing"
)

// FuzzDecodeNeverPanics：游标来自客户端、完全不可信（任意长度/字节）。
// 契约：Decode 只能返回 nil 或 ErrInvalidCursor，绝不 panic（GCM Open/
// base64/JSON 三段都可能收到畸形输入；此前无 fuzz 覆盖）。
func FuzzDecodeNeverPanics(f *testing.F) {
	valid, err := Encode("users", map[string]any{"id": "u1", "page": 2})
	if err != nil {
		f.Fatalf("encode seed: %v", err)
	}
	f.Add("")
	f.Add("not-a-cursor")
	f.Add(base64.RawURLEncoding.EncodeToString([]byte("short")))
	f.Add(valid)
	f.Add(base64.RawURLEncoding.EncodeToString(make([]byte, 4096)))
	f.Fuzz(func(t *testing.T, value string) {
		var out map[string]any
		err := Decode("users", value, &out)
		if err != nil && !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("unexpected decode error: %v", err)
		}
	})
}

// FuzzDecodeStringNeverPanics：DecodeString 与 Decode 共用 decrypt 通道，
// 同样只需「不 panic + 错误只可能是 ErrInvalidCursor」。
func FuzzDecodeStringNeverPanics(f *testing.F) {
	codec := NewCodec("fuzz-ns")
	valid, err := codec.Encode("hello")
	if err != nil {
		f.Fatalf("encode seed: %v", err)
	}
	f.Add(valid)
	f.Add("")
	f.Add("!!!!")
	f.Fuzz(func(t *testing.T, value string) {
		if _, err := codec.DecodeString(value); err != nil && !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("unexpected decode error: %v", err)
		}
	})
}
