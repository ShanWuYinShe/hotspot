package media

import "context"

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository() *PersistenceRepository {
	return &PersistenceRepository{impl: NewInMemoryRepository()}
}

func (r *PersistenceRepository) Save(ctx context.Context, uploadID string, upload pendingUpload) error {
	return r.impl.Save(ctx, uploadID, upload)
}

func (r *PersistenceRepository) Get(ctx context.Context, uploadID string) (pendingUpload, bool, error) {
	return r.impl.Get(ctx, uploadID)
}

func (r *PersistenceRepository) Complete(ctx context.Context, uploadID string, file UploadedMedia) (pendingUpload, bool, error) {
	return r.impl.Complete(ctx, uploadID, file)
}
