package auth

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	memory := NewInMemoryRepository()
	if factory == nil || factory.Config.PostgresDSN == "" {
		return &PersistenceRepository{impl: memory}
	}
	return &PersistenceRepository{impl: NewPostgresRepository(factory, nil)}
}

func (r *PersistenceRepository) Register(ctx context.Context, params RegisterParams) (Account, error) {
	return r.impl.Register(ctx, params)
}

func (r *PersistenceRepository) CreateGuest(ctx context.Context, params GuestParams) (Account, error) {
	return r.impl.CreateGuest(ctx, params)
}

func (r *PersistenceRepository) Authenticate(ctx context.Context, loginID, password string) (Account, error) {
	return r.impl.Authenticate(ctx, loginID, password)
}

func (r *PersistenceRepository) CreatePasswordReset(ctx context.Context, loginID string, expiresAt time.Time) error {
	return r.impl.CreatePasswordReset(ctx, loginID, expiresAt)
}

func (r *PersistenceRepository) ResetPassword(ctx context.Context, loginID, verificationCode, newPassword string) (Account, error) {
	return r.impl.ResetPassword(ctx, loginID, verificationCode, newPassword)
}

func (r *PersistenceRepository) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	return r.impl.ChangePassword(ctx, userID, oldPassword, newPassword)
}
