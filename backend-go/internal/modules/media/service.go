package media

import (
	"context"

	"hotspot/backend-go/internal/platform/observability"
)

type Service struct {
	repo     Repository
	provider Provider
}

func NewService() *Service { return NewServiceWithRepo(NewPersistenceRepository()) }
func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithRepoAndProvider(repo, NewStubProvider())
}
func NewServiceWithRepoAndProvider(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) Presign(ctx context.Context, req PresignUploadRequest) (PresignedUpload, error) {
	upload, err := s.provider.Presign(ctx, req)
	if err != nil {
		observability.L().Error("media.presign provider failed", "error", err)
		return PresignedUpload{}, err
	}
	if err := s.repo.Save(ctx, upload.UploadID, pendingUpload{Request: req}); err != nil {
		observability.L().Error("media.presign save failed", "error", err, "upload_id", upload.UploadID)
		return PresignedUpload{}, err
	}
	return upload, nil
}

func (s *Service) BatchPresign(ctx context.Context, req BatchPresignUploadRequest) (BatchPresignedUpload, error) {
	uploads := make([]PresignedUpload, 0, len(req.Files))
	pending := make([]pendingUpload, 0, len(req.Files))
	for _, file := range req.Files {
		upload, err := s.provider.Presign(ctx, file)
		if err != nil {
			observability.L().Error("media.batch_presign provider failed", "error", err)
			return BatchPresignedUpload{}, err
		}
		uploads = append(uploads, upload)
		pending = append(pending, pendingUpload{Request: file})
	}
	for i, upload := range pending {
		if err := s.repo.Save(ctx, uploads[i].UploadID, upload); err != nil {
			observability.L().Error("media.batch_presign save failed", "error", err, "upload_id", uploads[i].UploadID)
			return BatchPresignedUpload{}, err
		}
	}
	return BatchPresignedUpload{Uploads: uploads}, nil
}

func (s *Service) Complete(ctx context.Context, uploadID string) (UploadedMedia, bool, error) {
	upload, ok, err := s.repo.Get(ctx, uploadID)
	if err != nil {
		observability.L().Error("media.complete get failed", "error", err, "upload_id", uploadID)
		return UploadedMedia{}, false, err
	}
	if !ok {
		return UploadedMedia{}, false, nil
	}
	if upload.Completed {
		return upload.File, true, nil
	}
	file, err := s.provider.Complete(ctx, uploadID, upload.Request)
	if err != nil {
		observability.L().Error("media.complete provider failed", "error", err, "upload_id", uploadID)
		return UploadedMedia{}, false, err
	}
	completed, ok, err := s.repo.Complete(ctx, uploadID, file)
	if err != nil {
		observability.L().Error("media.complete save failed", "error", err, "upload_id", uploadID)
		return UploadedMedia{}, false, err
	}
	if !ok {
		return UploadedMedia{}, false, nil
	}
	return completed.File, true, nil
}
