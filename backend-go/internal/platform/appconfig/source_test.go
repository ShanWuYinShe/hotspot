package appconfig

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

func TestNewStaticSourceUsesRuntimeDefaults(t *testing.T) {
	src := NewStaticSource(config.Config{
		SupportEmail:       "support@example.com",
		PrivacyPolicyURL:   "https://example.com/privacy",
		TermsOfServiceURL:  "https://example.com/tos",
		OfficialWebsiteURL: "https://example.com",
		DefaultUserAvatar:  "user.png",
		DefaultGroupAvatar: "group.png",
	})
	cfg := src.Get(context.Background())
	if cfg.Contact.SupportEmail != "support@example.com" {
		t.Fatalf("unexpected support email: %s", cfg.Contact.SupportEmail)
	}
	if cfg.Defaults.DefaultUserAvatar != "user.png" {
		t.Fatalf("unexpected default user avatar: %s", cfg.Defaults.DefaultUserAvatar)
	}
	if cfg.Defaults.DefaultGroupAvatar != "group.png" {
		t.Fatalf("unexpected default group avatar: %s", cfg.Defaults.DefaultGroupAvatar)
	}
}

func TestStaticSourceUpdateAppliesFunction(t *testing.T) {
	src := NewStaticSource(config.Config{})
	updated, err := src.Update(context.Background(), func(cfg AppConfig) AppConfig {
		cfg.Features.EnableMoments = false
		cfg.Contact.SupportEmail = "new@example.com"
		return cfg
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Features.EnableMoments {
		t.Fatal("expected moments disabled")
	}
	if updated.Contact.SupportEmail != "new@example.com" {
		t.Fatalf("unexpected support email: %s", updated.Contact.SupportEmail)
	}
	got := src.Get(context.Background())
	if got.Contact.SupportEmail != "new@example.com" {
		t.Fatalf("expected persisted update, got %s", got.Contact.SupportEmail)
	}
}

func TestStaticSourceGetReturnsClone(t *testing.T) {
	src := NewStaticSource(config.Config{})
	cfg := src.Get(context.Background())
	cfg.Upload.AllowedImageTypes[0] = "mutated/type"
	again := src.Get(context.Background())
	if again.Upload.AllowedImageTypes[0] == "mutated/type" {
		t.Fatal("expected clone isolation for slices")
	}
}

func TestCloneConfigCopiesSlices(t *testing.T) {
	cfg := AppConfig{Upload: UploadConfig{AllowedImageTypes: []string{"a", "b"}, AllowedVideoTypes: []string{"v1"}}}
	cloned := cloneConfig(cfg)
	cloned.Upload.AllowedImageTypes[0] = "changed"
	cloned.Upload.AllowedVideoTypes[0] = "changed-video"
	if cfg.Upload.AllowedImageTypes[0] != "a" {
		t.Fatalf("original image slice mutated: %#v", cfg.Upload.AllowedImageTypes)
	}
	if cfg.Upload.AllowedVideoTypes[0] != "v1" {
		t.Fatalf("original video slice mutated: %#v", cfg.Upload.AllowedVideoTypes)
	}
}

func TestNullableString(t *testing.T) {
	if got := nullableString(""); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := nullableString("x"); got != "x" {
		t.Fatalf("expected x, got %#v", got)
	}
}

func TestParseTimeOrNow(t *testing.T) {
	parsed := parseTimeOrNow("2025-01-02T03:04:05Z")
	if parsed.UTC().Format(time.RFC3339) != "2025-01-02T03:04:05Z" {
		t.Fatalf("unexpected parsed time: %s", parsed.UTC().Format(time.RFC3339))
	}
	fallback := parseTimeOrNow("not-a-time")
	if fallback.IsZero() {
		t.Fatal("expected fallback current time")
	}
}
