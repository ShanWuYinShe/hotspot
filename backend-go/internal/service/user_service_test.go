package service

import (
	"context"
	"testing"
	"time"

	"backend-go/internal/auth"
	"backend-go/internal/domain"
	"backend-go/internal/types"
)

// fakeUserRepoForProfile 复用已有 fakeUserRepository，但单独定义以便与 AuthService 测试解耦。
type fakeUserRepoForProfile struct {
	usersByID map[string]*domain.User
}

func newFakeUserRepoForProfile() *fakeUserRepoForProfile {
	return &fakeUserRepoForProfile{usersByID: make(map[string]*domain.User)}
}

func (r *fakeUserRepoForProfile) CreateUser(_ context.Context, u *domain.User) error {
	copy := *u
	r.usersByID[u.UserID] = &copy
	return nil
}

func (r *fakeUserRepoForProfile) FindByLoginID(_ context.Context, loginID string) (*domain.User, error) {
	for _, u := range r.usersByID {
		if u.LoginID != nil && *u.LoginID == loginID {
			copy := *u
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepoForProfile) FindByID(_ context.Context, userID string) (*domain.User, error) {
	if u, ok := r.usersByID[userID]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, nil
}

func (r *fakeUserRepoForProfile) UpdatePasswordAndType(_ context.Context, userID string, passwordHash *string, userType string) error {
	u, ok := r.usersByID[userID]
	if !ok {
		return nil
	}
	u.Password = passwordHash
	u.UserType = userType
	return nil
}

// fakeSessionRepoForProfile 仅满足构造函数依赖，本轮测试不关心其具体行为。
type fakeSessionRepoForProfile struct{}

func (r *fakeSessionRepoForProfile) CreateSession(_ context.Context, s *domain.Session) error {
	return nil
}
func (r *fakeSessionRepoForProfile) ListSessionsByUser(_ context.Context, userID string) ([]domain.Session, error) {
	return nil, nil
}
func (r *fakeSessionRepoForProfile) RevokeSession(_ context.Context, userID, sessionID string) error {
	return nil
}
func (r *fakeSessionRepoForProfile) RevokeAllExcept(_ context.Context, userID, currentSessionID string) (int64, error) {
	return 0, nil
}

func TestUserService_GetMyProfile_BasicFieldsAndDefaults(t *testing.T) {
	repo := newFakeUserRepoForProfile()
	sessions := &fakeSessionRepoForProfile{}
	svc := NewUserService(repo, sessions)

	now := time.Now().UTC()
	u := &domain.User{
		UserID:    "user-1",
		Nickname:  "Alice",
		UserType:  "normal",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = repo.CreateUser(context.Background(), u)

	ctx := auth.WithAuthInfo(context.Background(), &auth.AuthInfo{UserID: "user-1"})
	profile, err := svc.GetMyProfile(ctx)
	if err != nil {
		t.Fatalf("GetMyProfile returned error: %v", err)
	}
	if profile.UserID != "user-1" || profile.Nickname != "Alice" || profile.UserType != "normal" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestUserService_GetMySettings_ReturnsDefaults(t *testing.T) {
	repo := newFakeUserRepoForProfile()
	sessions := &fakeSessionRepoForProfile{}
	svc := NewUserService(repo, sessions)

	repo.CreateUser(context.Background(), &domain.User{UserID: "user-1", Nickname: "Alice", UserType: "normal", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
	ctx := auth.WithAuthInfo(context.Background(), &auth.AuthInfo{UserID: "user-1"})

	settings, err := svc.GetMySettings(ctx)
	if err != nil {
		t.Fatalf("GetMySettings returned error: %v", err)
	}
	if settings == nil || !settings.Searchable || !settings.AllowFollow {
		t.Fatalf("unexpected default settings: %+v", settings)
	}
}

func TestUserService_DeleteAccount_InvalidConfirmationReturnsError(t *testing.T) {
	repo := newFakeUserRepoForProfile()
	sessions := &fakeSessionRepoForProfile{}
	svc := NewUserService(repo, sessions)

	repo.CreateUser(context.Background(), &domain.User{UserID: "user-1", Nickname: "Alice", UserType: "normal", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
	ctx := auth.WithAuthInfo(context.Background(), &auth.AuthInfo{UserID: "user-1"})

	err := svc.DeleteAccount(ctx, types.DeleteAccountRequest{Confirmation: "WRONG"})
	if err == nil {
		t.Fatalf("expected error for invalid confirmation, got nil")
	}
}
