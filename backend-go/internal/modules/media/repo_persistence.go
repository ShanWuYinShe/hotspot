package media

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) Save(ctx context.Context, upload pendingUpload) error {
	return r.impl.Save(ctx, upload)
}

func (r *PersistenceRepository) Get(ctx context.Context, uploadID string) (pendingUpload, bool, error) {
	return r.impl.Get(ctx, uploadID)
}

func (r *PersistenceRepository) Complete(ctx context.Context, uploadID string, file UploadedMedia, idempotencyKey string) (pendingUpload, bool, error) {
	return r.impl.Complete(ctx, uploadID, file, idempotencyKey)
}
