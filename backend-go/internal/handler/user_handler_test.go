package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"backend-go/internal/api"
	"backend-go/internal/auth"
	"backend-go/internal/service"
	"backend-go/internal/types"
)

// fakeUserService 实现 UserService，用于 Handler 层测试。
type fakeUserService struct {
	profile        *types.MyProfile
	settings       *types.UserSettings
	updateProfile  int
	updateLocation int
	upgradeCalled  int
	updateSettings int
	deleteCalled   int
}

func (f *fakeUserService) GetMyProfile(ctx context.Context) (*types.MyProfile, error) {
	return f.profile, nil
}

func (f *fakeUserService) UpdateMyProfile(ctx context.Context, req types.UpdateProfileRequest) error {
	f.updateProfile++
	return nil
}

func (f *fakeUserService) UpdateMyLocation(ctx context.Context, req types.UpdateLocationRequest) error {
	f.updateLocation++
	return nil
}

func (f *fakeUserService) UpgradeAccount(ctx context.Context, req types.UpgradeAccountRequest) error {
	f.upgradeCalled++
	return nil
}

func (f *fakeUserService) GetMySettings(ctx context.Context) (*types.UserSettings, error) {
	return f.settings, nil
}

func (f *fakeUserService) UpdateMySettings(ctx context.Context, req types.UpdateSettingsRequest) error {
	f.updateSettings++
	return nil
}

func (f *fakeUserService) DeleteAccount(ctx context.Context, req types.DeleteAccountRequest) error {
	f.deleteCalled++
	return nil
}

var _ service.UserService = (*fakeUserService)(nil)

func newUserRouter(svc service.UserService) *chi.Mux {
	r := chi.NewRouter()
	h := NewUserHandler(svc)
	RegisterUserRoutes(r, h)
	return r
}

func TestUserHandler_GetMyProfile_ReturnsProfile(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	svc := &fakeUserService{
		profile: &types.MyProfile{
			UserID:         "user-1",
			Nickname:       "Alice",
			UserType:       "normal",
			Location:       types.Location{Point: types.GeoPoint{Lat: 0, Lng: 0}, LocationName: ""},
			PresenceStatus: "offline",
			LastOnline:     now,
			CreatedAt:      now,
			Stats:          types.UserStats{},
		},
	}
	r := newUserRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/profile/get", nil)
	req = req.WithContext(auth.WithAuthInfo(req.Context(), &auth.AuthInfo{UserID: "user-1"}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp api.Response[types.MyProfile]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code=0, got %d", resp.Code)
	}
	if resp.Data == nil || resp.Data.UserID != "user-1" {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
}

func TestUserHandler_UpdateMyProfile_AcceptsBodyAndReturnsSuccess(t *testing.T) {
	svc := &fakeUserService{}
	r := newUserRouter(svc)

	body := types.UpdateProfileRequest{Nickname: func(s string) *string { return &s }("Bob")}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/profile/update", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithAuthInfo(req.Context(), &auth.AuthInfo{UserID: "user-1"}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if svc.updateProfile != 1 {
		t.Fatalf("expected UpdateMyProfile to be called once, got %d", svc.updateProfile)
	}
}

func TestUserHandler_GetMySettings_ReturnsSettings(t *testing.T) {
	svc := &fakeUserService{
		settings: &types.UserSettings{PrivacyLevel: "public", Searchable: true},
	}
	r := newUserRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/settings/get", nil)
	req = req.WithContext(auth.WithAuthInfo(req.Context(), &auth.AuthInfo{UserID: "user-1"}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp api.Response[types.UserSettings]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data == nil || resp.Data.PrivacyLevel != "public" {
		t.Fatalf("unexpected settings response: %+v", resp)
	}
}
