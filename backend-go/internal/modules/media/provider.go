package media

import "context"

type Provider interface {
	Presign(context.Context, PresignUploadRequest, string) (PresignedUpload, error)
	Head(context.Context, string) (ObjectMetadata, error)
}
