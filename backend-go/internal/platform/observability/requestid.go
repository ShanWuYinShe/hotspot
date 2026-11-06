package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync/atomic"
)

type requestIDKey struct{}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := clientRequestID(r.Header.Get("X-Request-Id"))
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-Id", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// clientRequestID 采用「非空即原样回显」：spec api_spec_base.md 明确服务端
// 不对客户端传入的 X-Request-Id 施加格式/熵值约束（此前 16-128 长度校验会把
// 合法的短客户端 ID 丢弃并换成服务端生成值，违反回显契约）。
func clientRequestID(value string) string {
	return value
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败极罕见；回退到单调计数器，避免全零可预测 ID。
		return fallbackRequestID()
	}
	return hex.EncodeToString(buf)
}

var fallbackRequestIDCounter atomic.Uint64

func fallbackRequestID() string {
	return "fallback-" + hex.EncodeToString([]byte{0})[:0] + u64hex(fallbackRequestIDCounter.Add(1))
}

func u64hex(v uint64) string {
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = byte(v)
		v >>= 8
	}
	return hex.EncodeToString(buf)
}
