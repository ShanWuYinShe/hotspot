package media

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) Save(ctx context.Context, upload pendingUpload) error {
	queryable, useFallback, err := r.queryable()
	if err != nil {
		return err
	}
	if useFallback {
		return r.fallback.Save(ctx, upload)
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
	queryable, useFallback, err := r.queryable()
	if err != nil {
		return pendingUpload{}, false, err
	}
	if useFallback {
		return r.fallback.Get(ctx, uploadID)
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
	queryable, useFallback, err := r.queryable()
	if err != nil {
		return pendingUpload{}, false, err
	}
	if useFallback {
		return r.fallback.Complete(ctx, uploadID, file, idempotencyKey)
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

func (r *PostgresRepository) queryable() (persistence.QueryableDB, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, true, nil
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, false, persistence.ErrNotImplemented
	}
	return queryable, false, nil
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
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return parsed.UTC()
}

func parseRFC3339OrNow(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}
