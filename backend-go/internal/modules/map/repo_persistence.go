package mapmodule

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory, nil)}
}

func (r *PersistenceRepository) Heatmap(ctx context.Context, query canvasQuery) (MapHeatCanvas, error) {
	return r.impl.Heatmap(ctx, query)
}

func (r *PersistenceRepository) Hotspots(ctx context.Context, query canvasQuery) (MapHotspotsCanvas, error) {
	return r.impl.Hotspots(ctx, query)
}
