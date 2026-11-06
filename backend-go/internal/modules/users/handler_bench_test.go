package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// BenchmarkGetMyProfileHandler 覆盖高频轻读 users/account 的 envelope 编码与
// 画像→AccountData 投影（stub repo，隔离 DB）。
func BenchmarkGetMyProfileHandler(b *testing.B) {
	repo := &stubUserRepo{
		profile:      MyProfile{UserID: "u1", Nickname: "Alice", UserType: "normal", Stats: UserStats{}},
		profileFound: true,
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1", SessionID: "s1"})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/account", strings.NewReader(`{}`)).WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.GetMyProfile(rr, req)
	}
}
