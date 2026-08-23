package bootstrap

import (
	"context"
	"time"

	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/modules/moments"
)

// mediaImageResolver adapts the media service to the moments.ImageResolver
// interface, translating media.MediaAsset into moments.ImageAsset. This keeps
// the moments package free of a direct dependency on media.
type mediaImageResolver struct{ svc *media.Service }

func NewMomentImageResolver(svc *media.Service) moments.ImageResolver {
	if svc == nil {
		return nil
	}
	return &mediaImageResolver{svc: svc}
}

func (r *mediaImageResolver) ResolveMomentImages(ctx context.Context, fileIDs []string, userID string) (map[string]moments.ImageAsset, error) {
	assets, err := r.svc.ResolveMomentImages(ctx, fileIDs, userID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]moments.ImageAsset, len(assets))
	for k, v := range assets {
		out[k] = moments.ImageAsset{
			FileID:       v.FileID,
			StorageKey:   v.StorageKey,
			ThumbnailURL: v.ThumbnailURL,
			Width:        v.Width,
			Height:       v.Height,
			MimeType:     v.MimeType,
		}
	}
	return out, nil
}

func (r *mediaImageResolver) PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	return r.svc.PresignGet(ctx, storageKey, ttl)
}
