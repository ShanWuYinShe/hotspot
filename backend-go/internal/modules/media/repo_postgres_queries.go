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

	// queryCompleteUploadSession 条件更新（已 completed 的行不再改写的幂等保障）
	// 并用 RETURNING 判断是否命中（审查 P2-19：此前 UPDATE 后再 Get 一次拿结果，
	// 每文件 2 次往返；调用方本已持有待写入的 file 元数据）。同一条 UPDATE 把
	// storage_key 改写为 copy-on-complete 的发布键（不可写键，发布后内容不可变）。
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
    completed_at = $11,
    storage_key = $12
WHERE upload_id = $1 AND status <> 'completed'
RETURNING upload_id`

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

	// queryPendingUploadUsage 预签名配额的用户级聚合：只统计未完成会话
	// （status <> 'completed'）。user_id 命中 idx_media_upload_sessions_user_created_at
	// 的前缀，单用户行数很少（配额的 purpose 正是把它压在个位数）。COUNT 返回
	// int8、SUM(integer) 返回 int8，统一 cast ::bigint 避免驱动层整数宽度差异；
	// 空集由 COALESCE 保证返回 0（SUM 空集为 NULL）。
	queryPendingUploadUsage = `
SELECT COUNT(*)::bigint,
       COALESCE(SUM(declared_file_size), 0)::bigint
FROM media_upload_sessions
WHERE user_id = $1 AND status <> 'completed'`
)

// 账户注销级联（user_purge.go）：Service 层在删除会话行之前先取出该用户全部
// 上传会话的存储键——含 completed 的不可写发布键与未完成的临时上传键——行删除
// 完成后逐条删除对应对象，避免已注销账号的媒体对象永久残留（第 81 轮审计）。
const queryListUserStorageKeys = `
SELECT storage_key
FROM media_upload_sessions
WHERE user_id = $1`

// 保留期 GC（retention.go）：超期未完成会话 + 其孤儿存储对象。
const (
	queryListStaleUploads = `
SELECT upload_id, storage_key, user_id
FROM media_upload_sessions
WHERE status <> 'completed' AND created_at < $1
ORDER BY created_at ASC
LIMIT $2`

	queryDeleteUploadRow = `
DELETE FROM media_upload_sessions WHERE upload_id = $1 AND status <> 'completed'`
)
