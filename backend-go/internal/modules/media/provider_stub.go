package media

import (
	"context"
	"fmt"
	"time"
)

type StubProvider struct{}

func NewStubProvider() *StubProvider { return &StubProvider{} }

func (p *StubProvider) Presign(ctx context.Context, req PresignUploadRequest) (PresignedUpload, error) {
	_ = ctx
	uploadID := fmt.Sprintf("upload_%d", time.Now().UTC().UnixNano())
	return PresignedUpload{
		UploadID:  uploadID,
		UploadURL: "https://upload.example.com/" + uploadID,
		Headers:   map[string]string{"Content-Type": req.MimeType},
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
	}, nil
}

func (p *StubProvider) Complete(ctx context.Context, uploadID string, req PresignUploadRequest) (UploadedMedia, error) {
	_ = ctx
	file := UploadedMedia{FileID: "file_" + uploadID, FileSize: req.FileSize, MimeType: req.MimeType}
	if req.Category == CategoryMomentImage || req.Category == CategoryChatImage || req.Category == CategoryAvatar || req.Category == CategoryGroupAvatar {
		file.Width = 1080
		file.Height = 1080
		file.ThumbnailURL = "https://cdn.example.com/thumb/" + uploadID
	}
	return file, nil
}
