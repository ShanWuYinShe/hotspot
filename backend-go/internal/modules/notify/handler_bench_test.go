package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// BenchmarkNotifyListHandler 覆盖 notify/fetch 列表 envelope 编码（空列表 stub）。
func BenchmarkNotifyListHandler(b *testing.B) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1", SessionID: "s1"})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notify/fetch", strings.NewReader(`{}`)).WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.List(rr, req)
	}
}
