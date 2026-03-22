package mapmodule

import "context"

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository() *PersistenceRepository {
	return &PersistenceRepository{impl: NewInMemoryRepository()}
}

func (r *PersistenceRepository) Heatmap(ctx context.Context, query canvasQuery) (MapHeatCanvas, error) {
	return r.impl.Heatmap(ctx, query)
}

func (r *PersistenceRepository) Hotspots(ctx context.Context, query canvasQuery) (MapHotspotsCanvas, error) {
	return r.impl.Hotspots(ctx, query)
}
