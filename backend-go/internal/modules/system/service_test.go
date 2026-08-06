package system

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/httpx"
	"testing"
	"time"
)

type stubRepo struct {
	meta        ApiVersionMeta
	healthCode  httpx.Code
	healthMsg   string
	health      *HealthStatus
	announces   []AnnouncementRecord
	announceErr error
}

func (r *stubRepo) GetMeta(_ context.Context) ApiVersionMeta { return r.meta }
func (r *stubRepo) Health(_ context.Context) (int, string, *HealthStatus) {
	return int(r.healthCode), r.healthMsg, r.health
}
func (r *stubRepo) ListAnnouncements(_ context.Context, _ int) ([]AnnouncementRecord, error) {
	return r.announces, r.announceErr
}
func (r *stubRepo) SearchAnnouncements(_ context.Context, _ AnnouncementSearchFilter) (AnnouncementPage, error) {
	return AnnouncementPage{}, nil
}
func (r *stubRepo) GetAnnouncement(_ context.Context, _ string) (AnnouncementRecord, bool, error) {
	return AnnouncementRecord{}, false, nil
}
func (r *stubRepo) CreateAnnouncement(_ context.Context, _ AnnouncementCreateInput) (AnnouncementRecord, error) {
	return AnnouncementRecord{}, nil
}
func (r *stubRepo) UpdateAnnouncement(_ context.Context, _ AnnouncementUpdateInput) (AnnouncementRecord, error) {
	return AnnouncementRecord{}, nil
}
func (r *stubRepo) DeleteAnnouncement(_ context.Context, _ string) error { return nil }

type stubProvider struct {
	config        ClientConfig
	announceCount int
}

func (p *stubProvider) GetClientConfig(_ context.Context) ClientConfig { return p.config }
func (p *stubProvider) GetSystemAnnouncementsCount(_ context.Context) int {
	if p.announceCount <= 0 {
		return 10
	}
	return p.announceCount
}

func TestServiceGetMetaDelegates(t *testing.T) {
	expected := ApiVersionMeta{Version: "v2", Status: ApiVersionStatusDeprecated}
	service := NewServiceWithDeps(&stubRepo{meta: expected}, &stubProvider{})
	result := service.GetMeta(context.Background())
	if result.Version != "v2" || result.Status != ApiVersionStatusDeprecated {
		t.Fatalf("unexpected meta: %#v", result)
	}
}

func TestServiceHealthDelegates(t *testing.T) {
	health := &HealthStatus{Status: SystemHealthHealthy, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	service := NewServiceWithDeps(&stubRepo{healthCode: 0, healthMsg: "", health: health}, &stubProvider{})
	code, msg, h := service.Health(context.Background())
	if code != 0 || msg != "" || h.Status != SystemHealthHealthy {
		t.Fatalf("unexpected health: code=%d msg=%q health=%#v", code, msg, h)
	}
}

func TestServiceListAnnouncementsAppliesCountLimit(t *testing.T) {
	now := time.Now().UTC()
	records := []AnnouncementRecord{
		{AnnouncementID: "a1", Title: "First", Content: "c1", CreatedAt: now},
		{AnnouncementID: "a2", Title: "Second", Content: "c2", CreatedAt: now},
	}
	repo := &stubRepo{announces: records}
	provider := &stubProvider{announceCount: 5}
	service := NewServiceWithDeps(repo, provider)
	result, err := service.ListAnnouncements(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].AnnouncementID != "a1" {
		t.Fatalf("expected a1, got %s", result.Items[0].AnnouncementID)
	}
}

func TestServiceListAnnouncementsPropagatesError(t *testing.T) {
	service := NewServiceWithDeps(&stubRepo{announceErr: errors.New("boom")}, &stubProvider{})
	_, err := service.ListAnnouncements(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceGetClientConfigDelegates(t *testing.T) {
	expected := ClientConfig{Features: FeatureSwitches{EnableMoments: true}}
	service := NewServiceWithDeps(&stubRepo{}, &stubProvider{config: expected})
	result := service.GetClientConfig(context.Background())
	if !result.Features.EnableMoments {
		t.Fatal("expected EnableMoments=true")
	}
}

func TestServiceGetClientConfigReturnsConfig(t *testing.T) {
	expected := ClientConfig{
		Features: FeatureSwitches{EnableNearbyPeople: true, EnableMoments: false},
		Limits:   AppLimits{MaxGroupMembers: 200, CommentMaxLength: 500},
		Contact:  ContactInfo{SupportEmail: "test@example.com"},
	}
	service := NewServiceWithDeps(&stubRepo{}, &stubProvider{config: expected})
	result := service.GetClientConfig(context.Background())
	if !result.Features.EnableNearbyPeople {
		t.Fatal("expected EnableNearbyPeople=true")
	}
	if result.Features.EnableMoments {
		t.Fatal("expected EnableMoments=false")
	}
	if result.Limits.MaxGroupMembers != 200 {
		t.Fatalf("expected MaxGroupMembers=200, got %d", result.Limits.MaxGroupMembers)
	}
	if result.Contact.SupportEmail != "test@example.com" {
		t.Fatalf("expected support email test@example.com, got %s", result.Contact.SupportEmail)
	}
}

func TestServiceListAnnouncementsEmptyList(t *testing.T) {
	repo := &stubRepo{announces: []AnnouncementRecord{}}
	provider := &stubProvider{announceCount: 5}
	service := NewServiceWithDeps(repo, provider)
	result, err := service.ListAnnouncements(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Items == nil {
		t.Fatal("expected non-nil items slice")
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Items))
	}
}

func TestServiceListAnnouncementsZeroCount(t *testing.T) {
	repo := &stubRepo{announces: []AnnouncementRecord{
		{AnnouncementID: "a1", Title: "T", Content: "C", CreatedAt: time.Now().UTC()},
	}}
	provider := &stubProvider{announceCount: 0}
	service := NewServiceWithDeps(repo, provider)
	result, err := service.ListAnnouncements(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].AnnouncementID != "a1" {
		t.Fatalf("expected a1, got %s", result.Items[0].AnnouncementID)
	}
}
