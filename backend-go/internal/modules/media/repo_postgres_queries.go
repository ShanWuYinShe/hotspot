package media

const (
	queryInsertUploadSession = `
INSERT INTO media_upload_sessions (
    upload_id,
    user_id,
    storage_key,
    file_name,
    declared_file_size,
    declared_mime_type,
    category,
    md5,
    status,
    file_id,
    final_file_size,
    final_mime_type,
    thumbnail_url,
    duration,
    width,
    height,
    completed_idempotency_key,
    created_at,
    completed_at
) VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
)
ON CONFLICT (upload_id) DO UPDATE
SET user_id = EXCLUDED.user_id,
    storage_key = EXCLUDED.storage_key,
    file_name = EXCLUDED.file_name,
    declared_file_size = EXCLUDED.declared_file_size,
    declared_mime_type = EXCLUDED.declared_mime_type,
    category = EXCLUDED.category,
    md5 = EXCLUDED.md5,
    status = EXCLUDED.status,
    created_at = EXCLUDED.created_at,
    completed_at = EXCLUDED.completed_at`

	queryGetUploadSession = `
SELECT upload_id,
       user_id,
       storage_key,
       file_name,
       declared_file_size,
       declared_mime_type,
       category,
       COALESCE(md5, ''),
       status,
       COALESCE(file_id, ''),
       COALESCE(final_file_size, 0),
       COALESCE(final_mime_type, ''),
       COALESCE(thumbnail_url, ''),
       COALESCE(duration, 0),
       COALESCE(width, 0),
       COALESCE(height, 0),
       COALESCE(completed_idempotency_key, ''),
       created_at,
       completed_at
FROM media_upload_sessions
WHERE upload_id = $1`

	queryCompleteUploadSession = `
UPDATE media_upload_sessions
SET status = $2,
    file_id = $3,
    final_file_size = $4,
    final_mime_type = $5,
    thumbnail_url = $6,
    duration = $7,
    width = $8,
    height = $9,
    completed_idempotency_key = $10,
    completed_at = $11
WHERE upload_id = $1`

	queryResolveByFileIDs = `
SELECT file_id,
       storage_key,
       COALESCE(thumbnail_url, ''),
       COALESCE(width, 0),
       COALESCE(height, 0),
       COALESCE(final_mime_type, '')
FROM media_upload_sessions
WHERE file_id = ANY($1)
  AND user_id = $2
  AND category = $3
  AND status = 'completed'`
)
