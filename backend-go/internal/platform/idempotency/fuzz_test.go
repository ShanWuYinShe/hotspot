package idempotency

import (
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// FuzzRequireKey：Idempotency-Key 来自客户端。契约：任意 header 值不 panic；
// 缺失/格式非法返回 10000（缺少）或 22002（格式非法），合法键写入 context。
func FuzzRequireKey(f *testing.F) {
	f.Add("")
	f.Add("short")
	f.Add("12345678901234567890123456789012")
	f.Add("A_-9abcdefghijklmnopqrstuvwxyz012345")
	f.Fuzz(func(t *testing.T, key string) {
		req := httptest.NewRequest("POST", "/", nil)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		apiErr := RequireKey(req)
		if apiErr == nil {
			if got := KeyFromContext(req.Context()); got != key {
				t.Fatalf("context key mismatch: %q", got)
			}
			return
		}
		switch apiErr.Code {
		case httpx.CodeInvalidRequest, httpx.CodeIdempotencyConflict:
		default:
			t.Fatalf("unexpected error code: %v", apiErr)
		}
	})
}
