package reactions

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

// PersistenceRepository is the production Repository, delegating each method
// to the postgres implementation. It exists so the Service holds the
// Repository interface while the concrete wiring stays swappable.
type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) React(ctx context.Context, targetType ReactTarget, targetID, userID, emoji string) (bool, error) {
	return r.impl.React(ctx, targetType, targetID, userID, emoji)
}

func (r *PersistenceRepository) Unreact(ctx context.Context, targetType ReactTarget, targetID, userID string) (bool, error) {
	return r.impl.Unreact(ctx, targetType, targetID, userID)
}

func (r *PersistenceRepository) ListReactions(ctx context.Context, targetType ReactTarget, targetID, emoji string, limit int, cursor string) ([]ReactionItem, string, error) {
	return r.impl.ListReactions(ctx, targetType, targetID, emoji, limit, cursor)
}

func (r *PersistenceRepository) Aggregate(ctx context.Context, targetType ReactTarget, targetID, viewerID string) ([]Reaction, error) {
	return r.impl.Aggregate(ctx, targetType, targetID, viewerID)
}

func (r *PersistenceRepository) AggregateBatch(ctx context.Context, targetType ReactTarget, targetIDs []string, viewerID string) (map[string][]Reaction, error) {
	return r.impl.AggregateBatch(ctx, targetType, targetIDs, viewerID)
}

func (r *PersistenceRepository) ListReactedTargets(ctx context.Context, targetType ReactTarget, userID string, limit int, cursor string) ([]string, string, error) {
	return r.impl.ListReactedTargets(ctx, targetType, userID, limit, cursor)
}
