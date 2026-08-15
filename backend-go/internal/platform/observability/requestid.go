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
		requestID := r.Header.Get("X-Request-Id")
		if !validRequestID(requestID) {
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

func validRequestID(value string) bool {
	return len(value) >= 16 && len(value) <= 128
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
