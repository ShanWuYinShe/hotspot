package httpx

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzDecodeStrictJSON：请求体完全不可信。契约：任意字节输入下 DecodeStrictJSON
// 不 panic，失败必须是 10000（含「请求体过大」分支不误判为格式错误）。
func FuzzDecodeStrictJSON(f *testing.F) {
	f.Add(`{"name":"ok"}`)
	f.Add("")
	f.Add("not-json")
	f.Add(`{"a":1,"a":2}`)
	f.Add(`{"trailing":1} extra`)
	f.Add(strings.Repeat("x", 4096))
	f.Fuzz(func(t *testing.T, body string) {
		var dest struct {
			Name string `json:"name"`
		}
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		apiErr := DecodeStrictJSON(req, &dest)
		if apiErr != nil && apiErr.Code != CodeInvalidRequest {
			t.Fatalf("unexpected error code: %v", apiErr)
		}
	})
}

// FuzzRequireEmptyObject：空对象端点只容忍未知字段，必须拒绝非对象/坏 JSON，
// 且任意输入不 panic。
func FuzzRequireEmptyObject(f *testing.F) {
	f.Add("{}")
	f.Add(`{"unknown_field":"ignored"}`)
	f.Add("")
	f.Add("[]")
	f.Add("not-json")
	f.Fuzz(func(t *testing.T, body string) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		apiErr := RequireEmptyObject(req)
		if apiErr != nil && apiErr.Code != CodeInvalidRequest {
			t.Fatalf("unexpected error code: %v", apiErr)
		}
	})
}

var _ = errors.Is // keep errors import intentional if helpers change
