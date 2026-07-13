package media

import (
	"context"
	"sync"
)

type InMemoryRepository struct {
	mu      sync.RWMutex
	uploads map[string]pendingUpload
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{uploads: make(map[string]pendingUpload)}
}

func (r *InMemoryRepository) Save(ctx context.Context, upload pendingUpload) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uploads[upload.UploadID] = upload
	return nil
}

func (r *InMemoryRepository) Get(ctx context.Context, uploadID string) (pendingUpload, bool, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	upload, ok := r.uploads[uploadID]
	return upload, ok, nil
}

func (r *InMemoryRepository) Complete(ctx context.Context, uploadID string, file UploadedMedia, idempotencyKey string) (pendingUpload, bool, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	upload, ok := r.uploads[uploadID]
	if !ok {
		return pendingUpload{}, false, nil
	}
	upload.Completed = true
	upload.File = file
	upload.CompletedIdempotencyKey = idempotencyKey
	r.uploads[uploadID] = upload
	return upload, true, nil
}

func (r *InMemoryRepository) ResolveByFileIDs(_ context.Context, fileIDs []string, userID string, category MediaCategory) (map[string]MediaAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	want := make(map[string]struct{}, len(fileIDs))
	for _, id := range fileIDs {
		want[id] = struct{}{}
	}
	assets := make(map[string]MediaAsset)
	for _, upload := range r.uploads {
		if !upload.Completed || upload.UserID != userID || upload.Category != category {
			continue
		}
		if upload.File.FileID == "" {
			continue
		}
		if _, ok := want[upload.File.FileID]; !ok {
			continue
		}
		assets[upload.File.FileID] = MediaAsset{
			FileID:       upload.File.FileID,
			StorageKey:   upload.StorageKey,
			ThumbnailURL: upload.File.ThumbnailURL,
			Width:        upload.File.Width,
			Height:       upload.File.Height,
			MimeType:     upload.File.MimeType,
		}
	}
	return assets, nil
}
