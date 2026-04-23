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
	"hotspot/backend-go/internal/platform/httpx"
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

func TestGetMetaReturnsActiveV1(t *testing.T) {
	repo := &stubRepo{meta: ApiVersionMeta{Version: "v1", Status: ApiVersionStatusActive}}
	service := NewServiceWithDeps(repo, &stubProvider{})
	handler := NewHandler(service)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/get_meta", strings.NewReader(`{}`))
	handler.GetMeta(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var resp httpx.ApiResponse[ApiVersionMeta]
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Fatalf("code=%d", resp.Code)
	}
	if resp.Data.Version != "v1" {
		t.Fatalf("version=%s", resp.Data.Version)
	}
	if resp.Data.Status != ApiVersionStatusActive {
		t.Fatalf("status=%s", resp.Data.Status)
	}
}

func TestHealthReturnsHealthyStatus(t *testing.T) {
	ts := time.Now().UTC().Format(time.RFC3339)
	health := &HealthStatus{Status: SystemHealthHealthy, Timestamp: ts}
	service := NewServiceWithDeps(&stubRepo{healthCode: 0, health: health}, &stubProvider{})
	handler := NewHandler(service)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/health", strings.NewReader(`{}`))
	handler.Health(rr, req)
	var resp httpx.ApiResponse[HealthStatus]
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Data.Status != SystemHealthHealthy {
		t.Fatalf("expected healthy, got %s", resp.Data.Status)
	}
}

func TestHealthReturnsUnhealthyCode(t *testing.T) {
	service := NewServiceWithDeps(&stubRepo{healthCode: 10600, healthMsg: "db down", health: nil}, &stubProvider{})
	handler := NewHandler(service)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/health", strings.NewReader(`{}`))
	handler.Health(rr, req)
	var resp httpx.ApiResponse[HealthStatus]
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != 10600 {
		t.Fatalf("expected code 10600, got %d", resp.Code)
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
func (r *staticRepo) GetAnnouncement(context.Context, string) (AnnouncementRecord, bool, error) {
	return AnnouncementRecord{}, false, nil
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

func TestGetClientConfigUsesSharedAppConfigSource(t *testing.T) {
	cfg := config.Load()
	source := appconfig.NewStaticSource(cfg)
	_, err := source.Update(context.Background(), func(current appconfig.AppConfig) appconfig.AppConfig {
		current.Features.EnableNearbyPeople = false
		current.Contact.SupportEmail = "shared@example.com"
		current.Defaults.DefaultLocation.LocationName = "共享配置位置"
		return current
	})
	if err != nil {
		t.Fatalf("update shared source: %v", err)
	}

	handler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(source)))
	req := httptest.NewRequest(http.MethodPost, "/get_client_config", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	handler.GetClientConfig(rr, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Features struct {
				EnableNearbyPeople bool `json:"enable_nearby_people"`
			} `json:"features"`
			Contact struct {
				SupportEmail string `json:"support_email"`
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
	if resp.Data.Features.EnableNearbyPeople {
		t.Fatalf("expected shared feature switch update to be visible: %#v", resp.Data.Features)
	}
	if resp.Data.Contact.SupportEmail != "shared@example.com" {
		t.Fatalf("unexpected support email: %#v", resp.Data.Contact)
	}
	if resp.Data.Defaults.DefaultLocation.LocationName != "共享配置位置" {
		t.Fatalf("unexpected default location: %#v", resp.Data.Defaults.DefaultLocation)
	}
}
