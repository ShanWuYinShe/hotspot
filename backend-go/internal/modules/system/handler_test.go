package system

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

func TestGetMetaRejectsNonEmptyBody(t *testing.T) {
	handler := NewHandler(NewService(appconfig.NewStaticSource(config.Load())))
	req := httptest.NewRequest(http.MethodPost, "/get_meta", strings.NewReader(`{"foo":"bar"}`))
	rr := httptest.NewRecorder()

	handler.GetMeta(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || int(code) != 10000 {
		t.Fatalf("unexpected code: %#v", resp["code"])
	}
}

func TestGetClientConfigReturnsSpecShape(t *testing.T) {
	cfg := config.Load()
	cfg.SupportEmail = "support@example.com"
	cfg.OfficialWebsiteURL = "https://geotalk.app"
	handler := NewHandler(NewService(appconfig.NewStaticSource(cfg)))
	req := httptest.NewRequest(http.MethodPost, "/get_client_config", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	handler.GetClientConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Contact struct {
				SupportEmail    string `json:"support_email"`
				OfficialWebsite string `json:"official_website"`
			} `json:"contact"`
			Defaults struct {
				DefaultLocation struct {
					LocationName string `json:"location_name"`
				} `json:"default_location"`
			} `json:"defaults"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected code: %d", resp.Code)
	}
	if resp.Data.Contact.SupportEmail != "support@example.com" {
		t.Fatalf("unexpected support email: %s", resp.Data.Contact.SupportEmail)
	}
	if resp.Data.Contact.OfficialWebsite != "https://geotalk.app" {
		t.Fatalf("unexpected official website: %s", resp.Data.Contact.OfficialWebsite)
	}
	if resp.Data.Defaults.DefaultLocation.LocationName == "" {
		t.Fatal("expected default location name")
	}
}

type staticRepo struct {
	items []AnnouncementRecord
	err   error
}

func (r *staticRepo) GetMeta(context.Context) ApiVersionMeta {
	return ApiVersionMeta{Version: "v1", Status: ApiVersionStatusActive}
}
func (r *staticRepo) Health(context.Context) (int, string, *HealthStatus) {
	return 0, "", &HealthStatus{Status: SystemHealthHealthy, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}
func (r *staticRepo) ListAnnouncements(context.Context, int) ([]AnnouncementRecord, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]AnnouncementRecord(nil), r.items...), nil
}
func (r *staticRepo) SearchAnnouncements(context.Context, AnnouncementSearchFilter) (AnnouncementPage, error) {
	return AnnouncementPage{}, nil
}
func (r *staticRepo) CreateAnnouncement(context.Context, AnnouncementCreateInput) (AnnouncementRecord, error) {
	return AnnouncementRecord{}, nil
}
func (r *staticRepo) UpdateAnnouncement(context.Context, AnnouncementUpdateInput) (AnnouncementRecord, error) {
	return AnnouncementRecord{}, nil
}
func (r *staticRepo) DeleteAnnouncement(context.Context, string) error { return nil }

type fixedProvider struct{ count int }

func (p fixedProvider) GetClientConfig(context.Context) ClientConfig    { return ClientConfig{} }
func (p fixedProvider) GetSystemAnnouncementsCount(context.Context) int { return p.count }

func TestListAnnouncementsReturnsRepoData(t *testing.T) {
	repo := &staticRepo{items: []AnnouncementRecord{{
		AnnouncementID: "ann_1",
		Title:          "系统维护",
		Content:        "今晚维护",
		Severity:       AnnouncementSeverityWarning,
		ActionURL:      "https://example.com/maintenance",
		CreatedAt:      time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC),
	}}}
	handler := NewHandler(NewServiceWithDeps(repo, fixedProvider{count: 5}))
	req := httptest.NewRequest(http.MethodPost, "/list_announcements", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	handler.ListAnnouncements(rr, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []SystemAnnouncementItem `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.Data.Items[0].AnnouncementID != "ann_1" || resp.Data.Items[0].ActionURL != "https://example.com/maintenance" {
		t.Fatalf("unexpected announcement item: %#v", resp.Data.Items[0])
	}
}

func TestListAnnouncementsReturnsUnavailableOnRepoError(t *testing.T) {
	handler := NewHandler(NewServiceWithDeps(&staticRepo{err: errors.New("db down")}, fixedProvider{count: 5}))
	req := httptest.NewRequest(http.MethodPost, "/list_announcements", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	handler.ListAnnouncements(rr, req)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10600 || resp.Message != "服务暂时不可用" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
