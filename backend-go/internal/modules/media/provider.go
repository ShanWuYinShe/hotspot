package media

import "context"

type Provider interface {
	Presign(context.Context, PresignUploadRequest) (PresignedUpload, error)
	Complete(context.Context, string, PresignUploadRequest) (UploadedMedia, error)
}
