package media

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Save(ctx context.Context, upload pendingUpload) error {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return err
	}
	return queryable.Exec(ctx, queryInsertUploadSession,
		upload.UploadID,
		upload.UserID,
		upload.StorageKey,
		upload.FileName,
		upload.DeclaredFileSize,
		upload.DeclaredMimeType,
		upload.Category,
		nullableString(upload.MD5),
		statusFromCompleted(upload.Completed),
		nullableString(upload.File.FileID),
		nullableInt(upload.File.FileSize),
		nullableString(upload.File.MimeType),
		nullableString(upload.File.ThumbnailURL),
		nullableInt(upload.File.Duration),
		nullableInt(upload.File.Width),
		nullableInt(upload.File.Height),
		nullableString(upload.CompletedIdempotencyKey),
		parseRFC3339OrNow(upload.CreatedAt),
		nullableTime(upload.CompletedAt),
	)
}

func (r *PostgresRepository) Get(ctx context.Context, uploadID string) (pendingUpload, bool, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return pendingUpload{}, false, err
	}
	var (
		result        pendingUpload
		status        string
		fileID        string
		finalFileSize int
		finalMimeType string
		thumbnailURL  string
		duration      int
		width         int
		height        int
		completedKey  string
		createdAt     time.Time
		completedAt   *time.Time
	)
	if err := queryable.QueryRow(ctx, queryGetUploadSession, uploadID).Scan(
		&result.UploadID,
		&result.UserID,
		&result.StorageKey,
		&result.FileName,
		&result.DeclaredFileSize,
		&result.DeclaredMimeType,
		&result.Category,
		&result.MD5,
		&status,
		&fileID,
		&finalFileSize,
		&finalMimeType,
		&thumbnailURL,
		&duration,
		&width,
		&height,
		&completedKey,
		&createdAt,
		&completedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pendingUpload{}, false, nil
		}
		return pendingUpload{}, false, err
	}
	result.Completed = status == "completed"
	result.File = UploadedMedia{FileID: fileID, FileSize: finalFileSize, MimeType: finalMimeType, ThumbnailURL: thumbnailURL, Duration: duration, Width: width, Height: height}
	result.CompletedIdempotencyKey = completedKey
	result.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if completedAt != nil {
		result.CompletedAt = completedAt.UTC().Format(time.RFC3339)
	}
	return result, true, nil
}

func (r *PostgresRepository) Complete(ctx context.Context, uploadID string, file UploadedMedia, idempotencyKey string) (pendingUpload, bool, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return pendingUpload{}, false, err
	}
	completedAt := time.Now().UTC()
	if err := queryable.Exec(ctx, queryCompleteUploadSession,
		uploadID,
		"completed",
		file.FileID,
		file.FileSize,
		file.MimeType,
		nullableString(file.ThumbnailURL),
		nullableInt(file.Duration),
		nullableInt(file.Width),
		nullableInt(file.Height),
		idempotencyKey,
		completedAt,
	); err != nil {
		return pendingUpload{}, false, err
	}
	return r.Get(ctx, uploadID)
}

func (r *PostgresRepository) ResolveByFileIDs(ctx context.Context, fileIDs []string, userID string, category MediaCategory) (map[string]MediaAsset, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return nil, err
	}
	if len(fileIDs) == 0 {
		return map[string]MediaAsset{}, nil
	}
	rows, err := queryable.Query(ctx, queryResolveByFileIDs, fileIDs, userID, string(category))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assets := make(map[string]MediaAsset, len(fileIDs))
	for rows.Next() {
		var asset MediaAsset
		if err := rows.Scan(&asset.FileID, &asset.StorageKey, &asset.ThumbnailURL, &asset.Width, &asset.Height, &asset.MimeType); err != nil {
			return nil, err
		}
		assets[asset.FileID] = asset
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return assets, nil
}

func (r *PostgresRepository) queryable(ctx context.Context) (persistence.Queryable, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	return queryable, nil
}

func statusFromCompleted(completed bool) string {
	if completed {
		return "completed"
	}
	return "pending"
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableTime(value string) any {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return parsed.UTC()
}

func parseRFC3339OrNow(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}

// CompletedFileMetaByID 按 file_id 读取完成态文件的展示元数据（不限定
// owner/category——举报快照需要文件真值而非所有权范围内读取）。
func (r *PostgresRepository) CompletedFileMetaByID(ctx context.Context, fileID string) (FileMeta, bool, error) {
	q, err := r.queryable(ctx)
	if err != nil {
		return FileMeta{}, false, err
	}
	const q_ = `
SELECT category,
       storage_key,
       COALESCE(thumbnail_url, ''),
       COALESCE(width, 0),
       COALESCE(height, 0),
       COALESCE(final_mime_type, declared_mime_type, ''),
       COALESCE(file_name, ''),
       COALESCE(final_file_size, declared_file_size, 0),
       COALESCE(duration, 0)
FROM media_upload_sessions
WHERE file_id = $1 AND status = 'completed'`
	var meta FileMeta
	if err := q.QueryRow(ctx, q_, fileID).Scan(
		&meta.Category,
		&meta.StorageKey,
		&meta.ThumbnailURL,
		&meta.Width,
		&meta.Height,
		&meta.MimeType,
		&meta.FileName,
		&meta.FileSize,
		&meta.Duration,
	); err != nil {
		return FileMeta{}, false, err
	}
	return meta, true, nil
}

// ListStaleUploads 返回超过保留期仍未完成的会话（按 created_at 升序，
// 旧行优先处理），供保留期 GC 逐条删对象+删行。
func (r *PostgresRepository) ListStaleUploads(ctx context.Context, cutoff time.Time, limit int) ([]pendingUpload, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := queryable.Query(ctx, queryListStaleUploads, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]pendingUpload, 0)
	for rows.Next() {
		var item pendingUpload
		if err := rows.Scan(&item.UploadID, &item.StorageKey, &item.UserID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// DeleteUploadRow 按 upload_id 物理删除会话行（GC 第二步）。
func (r *PostgresRepository) DeleteUploadRow(ctx context.Context, uploadID string) error {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return err
	}
	return queryable.Exec(ctx, queryDeleteUploadRow, uploadID)
}
