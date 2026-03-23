package system

const (
	queryListPublicAnnouncements = `
SELECT sa.announcement_id,
       sa.title,
       sa.content,
       sa.severity,
       COALESCE(sa.action_url, ''),
       sa.created_at,
       sa.created_by_user_id,
       COALESCE(u.nickname, sa.created_by_user_id),
       COALESCE(u.avatar_url, ''),
       sa.send_push,
       sa.push_sent_at,
       sa.view_count
FROM system_announcements sa
LEFT JOIN users u ON u.user_id = sa.created_by_user_id
WHERE sa.deleted_at IS NULL AND sa.status = 'published'
ORDER BY sa.created_at DESC, sa.announcement_id DESC
LIMIT $1`

	querySearchAnnouncements = `
SELECT sa.announcement_id,
       sa.title,
       sa.content,
       sa.severity,
       COALESCE(sa.action_url, ''),
       sa.created_at,
       sa.created_by_user_id,
       COALESCE(u.nickname, sa.created_by_user_id),
       COALESCE(u.avatar_url, ''),
       sa.send_push,
       sa.push_sent_at,
       sa.view_count
FROM system_announcements sa
LEFT JOIN users u ON u.user_id = sa.created_by_user_id
WHERE sa.deleted_at IS NULL
  AND ($1 = '' OR sa.severity = $1)
  AND ($2::timestamptz IS NULL OR sa.created_at < $2 OR (sa.created_at = $2 AND sa.announcement_id < $3))
ORDER BY sa.created_at DESC, sa.announcement_id DESC
LIMIT $4`

	queryInsertAnnouncement = `
INSERT INTO system_announcements (
  announcement_id, title, content, severity, action_url, status,
  send_push, push_sent_at, view_count, created_by_user_id, created_at, updated_at, deleted_at
)
VALUES ($1,$2,$3,$4,$5,'published',$6,$7,$8,$9,$10,$10,NULL)`

	queryAnnouncementByID = `
SELECT sa.announcement_id,
       sa.title,
       sa.content,
       sa.severity,
       COALESCE(sa.action_url, ''),
       sa.created_at,
       sa.created_by_user_id,
       COALESCE(u.nickname, sa.created_by_user_id),
       COALESCE(u.avatar_url, ''),
       sa.send_push,
       sa.push_sent_at,
       sa.view_count,
       sa.deleted_at
FROM system_announcements sa
LEFT JOIN users u ON u.user_id = sa.created_by_user_id
WHERE sa.announcement_id = $1`

	queryUpdateAnnouncement = `
UPDATE system_announcements
SET title = $2,
    content = $3,
    severity = $4,
    action_url = $5,
    updated_at = $6
WHERE announcement_id = $1 AND deleted_at IS NULL`

	queryDeleteAnnouncement = `
UPDATE system_announcements
SET deleted_at = $2,
    updated_at = $2
WHERE announcement_id = $1 AND deleted_at IS NULL`
)
