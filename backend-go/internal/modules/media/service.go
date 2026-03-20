package media

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type pendingUpload struct {
	Request   PresignUploadRequest
	Completed bool
	File      UploadedMedia
}

type Service struct {
	mu      sync.RWMutex
	uploads map[string]pendingUpload
}

func NewService() *Service {
	return &Service{uploads: make(map[string]pendingUpload)}
}

func (s *Service) Presign(ctx context.Context, req PresignUploadRequest) PresignedUpload {
	_ = ctx
	uploadID := fmt.Sprintf("upload_%d", time.Now().UTC().UnixNano())
	s.mu.Lock()
	s.uploads[uploadID] = pendingUpload{Request: req}
	s.mu.Unlock()
	return PresignedUpload{
		UploadID:  uploadID,
		UploadURL: "https://upload.example.com/" + uploadID,
		Headers:   map[string]string{"Content-Type": req.MimeType},
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
	}
}

func (s *Service) BatchPresign(ctx context.Context, req BatchPresignUploadRequest) BatchPresignedUpload {
	uploads := make([]PresignedUpload, 0, len(req.Files))
	for _, file := range req.Files {
		uploads = append(uploads, s.Presign(ctx, file))
	}
	return BatchPresignedUpload{Uploads: uploads}
}

func (s *Service) Complete(ctx context.Context, uploadID string) (UploadedMedia, bool, bool) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	upload, ok := s.uploads[uploadID]
	if !ok {
		return UploadedMedia{}, false, false
	}
	if upload.Completed {
		return upload.File, true, true
	}
	file := UploadedMedia{FileID: "file_" + uploadID, FileSize: upload.Request.FileSize, MimeType: upload.Request.MimeType}
	if upload.Request.Category == CategoryMomentImage || upload.Request.Category == CategoryChatImage || upload.Request.Category == CategoryAvatar || upload.Request.Category == CategoryGroupAvatar {
		file.Width = 1080
		file.Height = 1080
		file.ThumbnailURL = "https://cdn.example.com/thumb/" + uploadID
	}
	upload.Completed = true
	upload.File = file
	s.uploads[uploadID] = upload
	return file, true, false
}
