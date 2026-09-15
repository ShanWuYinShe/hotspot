package moments

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

const (
	queryInsertMomentImage = `
INSERT INTO moment_images (moment_id, file_id, position, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (moment_id, file_id) DO NOTHING`

	queryDeleteMomentImagesByMomentID = `
DELETE FROM moment_images
WHERE moment_id = $1`

	// queryListMomentImagesByMomentIDs returns ordered image rows joined with
	// the completed media upload sessions. Rows are ordered by position so the
	// caller can append them directly to MomentItem.Images in publish order.
	// JOIN 条件必须带 status='completed'：045 的 idx_media_upload_sessions_file_id
	// 是 partial index，谓词不匹配则退化为对 media_upload_sessions 的全表扫描。
	queryListMomentImagesByMomentIDs = `
SELECT mi.moment_id,
       mus.file_id,
       mus.storage_key,
       COALESCE(mus.thumbnail_url, ''),
       COALESCE(mus.width, 0),
       COALESCE(mus.height, 0),
       COALESCE(mus.final_mime_type, '')
FROM moment_images mi
JOIN media_upload_sessions mus ON mus.file_id = mi.file_id AND mus.status = 'completed'
WHERE mi.moment_id = ANY($1)
ORDER BY mi.moment_id, mi.position ASC`
)

// insertMomentImages attaches the given file_ids to a moment in order. It is
// the caller's responsibility to have already validated ownership/category.
func insertMomentImages(ctx context.Context, queryable persistence.Queryable, momentID string, fileIDs []string, createdAt time.Time) error {
	for position, fileID := range fileIDs {
		if err := queryable.Exec(ctx, queryInsertMomentImage, momentID, fileID, position, createdAt); err != nil {
			return err
		}
	}
	return nil
}

// replaceMomentImages implements the spec's replace semantics for image_file_ids:
// delete all existing rows then re-insert the new set. An empty slice clears
// the moment's images; a nil slice is a no-op (caller should not call).
func replaceMomentImages(ctx context.Context, queryable persistence.Queryable, momentID string, fileIDs []string, createdAt time.Time) error {
	if err := queryable.Exec(ctx, queryDeleteMomentImagesByMomentID, momentID); err != nil {
		return err
	}
	if len(fileIDs) == 0 {
		return nil
	}
	return insertMomentImages(ctx, queryable, momentID, fileIDs, createdAt)
}

// momentImageRow is the scanned shape of queryListMomentImagesByMomentIDs.
type momentImageRow struct {
	MomentID     string
	FileID       string
	StorageKey   string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
}

// loadMomentImageRows fetches ordered image rows for the given moment IDs in a
// single query. Returns a map keyed by moment_id; each value is ordered by
// position. The caller presigns the storage_key/thumbnail_url into URLs.
func loadMomentImageRows(ctx context.Context, queryable persistence.Queryable, momentIDs []string) (map[string][]momentImageRow, error) {
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

// FetchMomentImageRows satisfies MomentImageFetcher for the PostgresRepository,
// resolving the active queryable (transaction context or pool).
func (r *PostgresRepository) FetchMomentImageRows(ctx context.Context, momentIDs []string) (map[string][]momentImageRow, error) {
	q, ok := queryable(ctx, r.factory)
	if !ok {
		return map[string][]momentImageRow{}, nil
	}
	return loadMomentImageRows(ctx, q, momentIDs)
}
