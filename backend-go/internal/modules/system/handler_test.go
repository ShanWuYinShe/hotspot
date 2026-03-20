package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
