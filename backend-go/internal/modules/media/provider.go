package media

import (
	"context"
	"time"
)

type Provider interface {
	Presign(context.Context, PresignUploadRequest, string) (PresignedUpload, error)
	PresignGet(context.Context, string, time.Duration) (string, error)
	Head(context.Context, string) (ObjectMetadata, error)
}
