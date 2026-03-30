package media

import "context"

type pendingUpload struct {
	UploadID                string
	UserID                  string
	StorageKey              string
	FileName                string
	DeclaredFileSize        int
	DeclaredMimeType        string
	Category                MediaCategory
	MD5                     string
	Completed               bool
	File                    UploadedMedia
	CompletedIdempotencyKey string
	CreatedAt               string
	CompletedAt             string
}

type Repository interface {
	Save(context.Context, pendingUpload) error
	Get(context.Context, string) (pendingUpload, bool, error)
	Complete(context.Context, string, UploadedMedia, string) (pendingUpload, bool, error)
}
