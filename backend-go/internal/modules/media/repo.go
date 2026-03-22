package media

import "context"

type pendingUpload struct {
	Request   PresignUploadRequest
	Completed bool
	File      UploadedMedia
}

type Repository interface {
	Save(context.Context, string, pendingUpload) error
	Get(context.Context, string) (pendingUpload, bool, error)
	Complete(context.Context, string, UploadedMedia) (pendingUpload, bool, error)
}
