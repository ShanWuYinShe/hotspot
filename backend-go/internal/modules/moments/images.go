package moments

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/observability"
)

// momentReadTTL is how long a presigned GET URL remains valid. Moments are
// long-lived content; the frontend re-fetches on each page load so expiry does
// not cause blank images — it only forces a refresh on the next visit.
const momentReadTTL = 7 * 24 * time.Hour

// ImageAsset mirrors the media module's resolved-asset view without importing
// it (avoids a moments -> media dependency at the type level).
type ImageAsset struct {
	FileID       string
	StorageKey   string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
}

// ImageResolver resolves uploaded media file_ids to stored assets and signs
// short-term client-facing read URLs. Implemented by the media service.
type ImageResolver interface {
	// ResolveMomentImages returns assets keyed by file_id. It enforces that
	// every id belongs to userID, has category 'moment_image' and is completed;
	// if any id is missing it returns an error (all-or-nothing).
	ResolveMomentImages(ctx context.Context, fileIDs []string, userID string) (map[string]ImageAsset, error)
	// PresignGet signs a short-term client-facing absolute read URL.
	PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error)
}

// ErrImagesNotOwned is returned when one or more image_file_ids cannot be
// resolved for the acting user (not owned, wrong category, or not completed).
var ErrImagesNotOwned = errors.New("moments: image file_ids not owned or not ready")

// MomentImageFetcher is the narrow repo capability needed by the hydration
// helper. Both the moments and feed postgres repos satisfy it via
// loadMomentImageRows.
type MomentImageFetcher interface {
	FetchMomentImageRows(ctx context.Context, momentIDs []string) (map[string][]momentImageRow, error)
}

// HydrateMomentImages fills item.Images for every item in place, presigning
// read URLs via resolver. It performs a single batched query. Items with no
// images keep a nil Images slice (omitted from JSON).
func HydrateMomentImages(ctx context.Context, fetcher MomentImageFetcher, resolver ImageResolver, items []MomentItem) {
	if len(items) == 0 {
		return
	}
	momentIDs := make([]string, 0, len(items))
	for _, it := range items {
		if it.MomentID != "" {
			momentIDs = append(momentIDs, it.MomentID)
		}
	}
	rowsByMoment, err := fetcher.FetchMomentImageRows(ctx, momentIDs)
	if err != nil {
		observability.L().Error("moments.hydrate_images fetch failed", "error", err)
		return
	}
	if resolver == nil {
		return
	}
	for i := range items {
		rows := rowsByMoment[items[i].MomentID]
		if len(rows) == 0 {
			continue
		}
		images := make([]MomentImage, 0, len(rows))
		for _, r := range rows {
			url, err := resolver.PresignGet(ctx, r.StorageKey, momentReadTTL)
			if err != nil {
				observability.L().Error("moments.hydrate_images presign failed", "error", err, "storage_key", r.StorageKey)
				continue
			}
			img := MomentImage{URL: url}
			if r.Width != 0 {
				width := r.Width
				img.Width = &width
			}
			if r.Height != 0 {
				height := r.Height
				img.Height = &height
			}
			if r.MimeType != "" {
				mimeType := r.MimeType
				img.MimeType = &mimeType
			}
			// thumbnail_url currently stored as a storage key path (or empty);
			// presign it when present, otherwise fall back to the full URL.
			if r.ThumbnailURL != "" {
				if thumb, err := resolver.PresignGet(ctx, r.ThumbnailURL, momentReadTTL); err == nil {
					img.ThumbnailURL = &thumb
				} else {
					img.ThumbnailURL = &url
				}
			} else {
				img.ThumbnailURL = &url
			}
			images = append(images, img)
		}
		if len(images) > 0 {
			items[i].Images = images
		}
	}
}

// HydrateSingleMoment fills a single moment's images in place.
func HydrateSingleMoment(ctx context.Context, fetcher MomentImageFetcher, resolver ImageResolver, item *MomentItem) {
	if item == nil {
		return
	}
	wrapped := []MomentItem{*item}
	HydrateMomentImages(ctx, fetcher, resolver, wrapped)
	item.Images = wrapped[0].Images
}
