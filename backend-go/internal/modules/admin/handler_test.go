package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
)

func TestGetMeReturnsPermissions(t *testing.T) {
	handler := NewHandler(NewService(appconfig.NewStaticSource(config.Load())))
	req := httptest.NewRequest(http.MethodPost, "/rbac/get_me", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "admin", SessionID: "sess_admin", IsAdmin: true})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetMe(rr, req)

	var resp struct {
		Code int              `json:"code"`
		Data AdminPermissions `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Permissions) == 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
