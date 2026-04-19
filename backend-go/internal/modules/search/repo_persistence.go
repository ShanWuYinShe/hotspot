package search

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) Suggest(ctx context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	return r.impl.Suggest(ctx, req)
}
