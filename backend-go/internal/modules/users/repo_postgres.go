package users

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) PutProfile(ctx context.Context, profile MyProfile) {
	r.fallback.PutProfile(ctx, profile)
}
func (r *PostgresRepository) DeleteProfile(ctx context.Context, userID string) {
	r.fallback.DeleteProfile(ctx, userID)
}
func (r *PostgresRepository) PutSettings(ctx context.Context, userID string, settings UserSettings) {
	r.fallback.PutSettings(ctx, userID, settings)
}
func (r *PostgresRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) []MyProfile {
	return r.fallback.SearchProfiles(ctx, req)
}
func (r *PostgresRepository) Follow(ctx context.Context, userID, targetUserID string) {
	r.fallback.Follow(ctx, userID, targetUserID)
}
func (r *PostgresRepository) Unfollow(ctx context.Context, userID, targetUserID string) {
	r.fallback.Unfollow(ctx, userID, targetUserID)
}
func (r *PostgresRepository) Block(ctx context.Context, userID, targetUserID string) {
	r.fallback.Block(ctx, userID, targetUserID)
}
func (r *PostgresRepository) Unblock(ctx context.Context, userID, targetUserID string) {
	r.fallback.Unblock(ctx, userID, targetUserID)
}
func (r *PostgresRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) {
	r.fallback.SetRemark(ctx, userID, targetUserID, remark)
}
func (r *PostgresRepository) ListBlocked(ctx context.Context, userID string) []BlockedUserItem {
	return r.fallback.ListBlocked(ctx, userID)
}
func (r *PostgresRepository) Relationship(ctx context.Context, userID, targetUserID string) Relationship {
	return r.fallback.Relationship(ctx, userID, targetUserID)
}
