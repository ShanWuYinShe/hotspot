package feed

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
)

// momentReadTTL mirrors the moments package's read URL TTL. Kept in sync so
// feed-served and moments-served moments have identical URL lifetimes.
const momentReadTTL = 7 * 24 * time.Hour

// ImageAsset is the feed-side view of a resolved media asset, mirroring
// moments.ImageAsset. The feed package keeps its own copy to avoid importing
// the moments package.
type ImageAsset struct {
	FileID       string
	StorageKey   string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
}

// ImageResolver resolves uploaded media file_ids to stored assets and signs
// short-term client-facing read URLs. Implemented by the bootstrap adapter
// over the media service.
type ImageResolver interface {
	ResolveMomentImages(ctx context.Context, fileIDs []string, userID string) (map[string]ImageAsset, error)
	PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error)
}

// momentImageRow is the scanned shape of the moment_images join query.
type momentImageRow struct {
	MomentID     string
	FileID       string
	StorageKey   string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
}

const queryListMomentImagesByMomentIDs = `
SELECT mi.moment_id,
       mus.file_id,
       mus.storage_key,
       COALESCE(mus.thumbnail_url, ''),
       COALESCE(mus.width, 0),
       COALESCE(mus.height, 0),
       COALESCE(mus.final_mime_type, '')
FROM moment_images mi
JOIN media_upload_sessions mus ON mus.file_id = mi.file_id
WHERE mi.moment_id = ANY($1)
ORDER BY mi.moment_id, mi.position ASC`

// FetchMomentImageRows fetches ordered image rows for the given moment IDs in
// a single query. Used by the service-layer hydration helper.
func FetchMomentImageRows(ctx context.Context, queryable persistence.Queryable, momentIDs []string) (map[string][]momentImageRow, error) {
	if len(momentIDs) == 0 {
		return map[string][]momentImageRow{}, nil
	}
	rows, err := queryable.Query(ctx, queryListMomentImagesByMomentIDs, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byMoment := make(map[string][]momentImageRow, len(momentIDs))
	for rows.Next() {
		var r momentImageRow
		if err := rows.Scan(&r.MomentID, &r.FileID, &r.StorageKey, &r.ThumbnailURL, &r.Width, &r.Height, &r.MimeType); err != nil {
			return nil, err
		}
		byMoment[r.MomentID] = append(byMoment[r.MomentID], r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return byMoment, nil
}

// hydrateMomentImages fills item.Images for every item in place, presigning
// read URLs via resolver. Items with no images keep a nil Images slice.
func hydrateMomentImages(ctx context.Context, resolver ImageResolver, rowsByMoment map[string][]momentImageRow, items []MomentItem) {
	if resolver == nil || len(items) == 0 {
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
				observability.L().Error("feed.hydrate_images presign failed", "error", err, "storage_key", r.StorageKey)
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
