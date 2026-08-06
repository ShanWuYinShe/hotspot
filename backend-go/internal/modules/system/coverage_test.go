package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
)

// fakeBootstrapResolver 提供确定性的 profile/unread 摘要，供 Bootstrap
// service/handler 测试使用。
type fakeBootstrapResolver struct {
	profile any
	found   bool
	summary UnreadSummary
}

func (f fakeBootstrapResolver) ResolveProfile(context.Context) (any, bool) {
	return f.profile, f.found
}

func (f fakeBootstrapResolver) ResolveUnreadSummary(context.Context) UnreadSummary {
	return f.summary
}

func TestServiceBootstrapWithoutResolver(t *testing.T) {
	svc := NewService(appconfig.NewStaticSource(config.Load()))
	bd := svc.Bootstrap(context.Background(), nil)
	if bd.BootstrapVersion == "" || bd.Meta.Version == "" {
		t.Fatalf("expected bootstrap version and meta, got %#v", bd)
	}
	if bd.Profile != nil || bd.UnreadSummary != nil {
		t.Fatalf("without resolver profile/unread must be absent, got %#v", bd)
	}
}

func TestServiceBootstrapWithResolver(t *testing.T) {
	svc := NewService(appconfig.NewStaticSource(config.Load()))
	resolver := fakeBootstrapResolver{
		profile: map[string]any{"user_id": "u1"},
		found:   true,
		summary: UnreadSummary{NotificationTotal: 3, TotalBadge: 5},
	}
	bd := svc.Bootstrap(context.Background(), resolver)
	if bd.Profile == nil {
		t.Fatal("expected profile from resolver")
	}
	if bd.UnreadSummary == nil || bd.UnreadSummary.NotificationTotal != 3 || bd.UnreadSummary.TotalBadge != 5 {
		t.Fatalf("expected resolver unread summary, got %#v", bd.UnreadSummary)
	}
}

func TestServiceBootstrapResolverMissKeepsProfileAbsent(t *testing.T) {
	svc := NewService(appconfig.NewStaticSource(config.Load()))
	svc.SetBootstrapResolver(fakeBootstrapResolver{found: false})
	if svc.GetBootstrapResolver() == nil {
		t.Fatal("expected resolver to be set")
	}
	bd := svc.Bootstrap(context.Background(), svc.GetBootstrapResolver())
	if bd.Profile != nil {
		t.Fatalf("resolver miss must not set profile, got %#v", bd.Profile)
	}
	if bd.UnreadSummary == nil {
		t.Fatal("unread summary is always resolved when resolver present")
	}
}

func TestBootstrapHandlerReturnsSpecShape(t *testing.T) {
	handler := NewHandler(NewService(appconfig.NewStaticSource(config.Load())))
	req := httptest.NewRequest(http.MethodPost, "/bootstrap", strings.NewReader(`{"client_type":"app"}`))
	rr := httptest.NewRecorder()
	handler.Bootstrap(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			BootstrapVersion string `json:"bootstrap_version"`
			Meta             struct {
				Version string `json:"version"`
			} `json:"meta"`
			ClientConfig json.RawMessage `json:"client_config"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode bootstrap response: %v", err)
	}
	if resp.Code != 0 || resp.Data.BootstrapVersion == "" || resp.Data.Meta.Version == "" || len(resp.Data.ClientConfig) == 0 {
		t.Fatalf("unexpected bootstrap response: %#v", resp)
	}
}

func TestBootstrapHandlerRejectsMalformedBody(t *testing.T) {
	handler := NewHandler(NewService(appconfig.NewStaticSource(config.Load())))
	req := httptest.NewRequest(http.MethodPost, "/bootstrap", strings.NewReader(`not-json`))
	rr := httptest.NewRecorder()
	handler.Bootstrap(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest for malformed body, got %d", resp.Code)
	}
}

func TestBootstrapHandlerCarriesResolverData(t *testing.T) {
	svc := NewService(appconfig.NewStaticSource(config.Load()))
	svc.SetBootstrapResolver(fakeBootstrapResolver{
		profile: map[string]any{"user_id": "u1", "nickname": "Alice"},
		found:   true,
		summary: UnreadSummary{NotificationTotal: 1},
	})
	handler := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/bootstrap", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.Bootstrap(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Profile       map[string]any `json:"profile"`
			UnreadSummary *UnreadSummary `json:"unread_summary"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Profile["user_id"] != "u1" || resp.Data.UnreadSummary == nil || resp.Data.UnreadSummary.NotificationTotal != 1 {
		t.Fatalf("expected resolver data in bootstrap, got %#v", resp.Data)
	}
}

func TestGetClientConfigRejectsNonEmptyBody(t *testing.T) {
	handler := NewHandler(NewService(appconfig.NewStaticSource(config.Load())))
	req := httptest.NewRequest(http.MethodPost, "/get_client_config", strings.NewReader(`[1,2]`))
	rr := httptest.NewRecorder()
	handler.GetClientConfig(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest for non-empty body, got %d", resp.Code)
	}
}
