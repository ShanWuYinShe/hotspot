package media

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type StubProvider struct{}

func NewStubProvider() *StubProvider { return &StubProvider{} }

func (p *StubProvider) Presign(ctx context.Context, req PresignUploadRequest, ownerUserID string) (PresignedUpload, error) {
	_ = ctx
	uploadID := fmt.Sprintf("upload_%d", time.Now().UTC().UnixNano())
	return PresignedUpload{
		UploadID:   uploadID,
		UploadURL:  "https://upload.example.com/" + uploadID,
		Headers:    map[string]string{"Content-Type": req.MimeType},
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		StorageKey: storageKeyFor(ownerUserID, uploadID, req.FileName),
	}, nil
}

func (p *StubProvider) Head(ctx context.Context, storageKey string) (ObjectMetadata, error) {
	_ = ctx
	_ = storageKey
	return ObjectMetadata{}, errors.New("stub provider does not support head")
}
