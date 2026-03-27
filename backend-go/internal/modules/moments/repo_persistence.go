package moments

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	memory := NewInMemoryRepository()
	if factory == nil || factory.Config.PostgresDSN == "" {
		return &PersistenceRepository{impl: memory}
	}
	return &PersistenceRepository{impl: NewPostgresRepository(factory, memory)}
}

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	return r.impl.Get(ctx, momentID)
}

func (r *PersistenceRepository) ListMine(ctx context.Context, userID string) (PaginatedMoments, error) {
	return r.impl.ListMine(ctx, userID)
}

func (r *PersistenceRepository) Exists(ctx context.Context, momentID string) (bool, error) {
	return r.impl.Exists(ctx, momentID)
}

func (r *PersistenceRepository) IncrementCommentCount(ctx context.Context, momentID string) error {
	return r.impl.IncrementCommentCount(ctx, momentID)
}

func (r *PersistenceRepository) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	return r.impl.Search(ctx, req)
}

func (r *PersistenceRepository) Update(ctx context.Context, userID string, req EditMomentRequest) error {
	return r.impl.Update(ctx, userID, req)
}

func (r *PersistenceRepository) Delete(ctx context.Context, momentID string) error {
	return r.impl.Delete(ctx, momentID)
}

func (r *PersistenceRepository) Like(ctx context.Context, userID string, momentID string) error {
	return r.impl.Like(ctx, userID, momentID)
}

func (r *PersistenceRepository) Unlike(ctx context.Context, userID string, momentID string) error {
	return r.impl.Unlike(ctx, userID, momentID)
}

func (r *PersistenceRepository) Favorite(ctx context.Context, userID string, momentID string) error {
	return r.impl.Favorite(ctx, userID, momentID)
}

func (r *PersistenceRepository) Unfavorite(ctx context.Context, userID string, momentID string) error {
	return r.impl.Unfavorite(ctx, userID, momentID)
}

func (r *PersistenceRepository) ListLikes(ctx context.Context, momentID string, limit int, cursor string) ([]LikeItem, string, error) {
	return r.impl.ListLikes(ctx, momentID, limit, cursor)
}

func (r *PersistenceRepository) ListFavorites(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	return r.impl.ListFavorites(ctx, userID, limit, cursor)
}
