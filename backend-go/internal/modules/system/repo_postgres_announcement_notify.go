package system

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

const queryListAnnouncementNotifications = `SELECT sa.announcement_id, sa.title, sa.severity, sa.created_at,
       (ar.user_id IS NOT NULL) AS is_read
FROM system_announcements sa
JOIN users viewer ON viewer.user_id = $1
LEFT JOIN announcement_reads ar
  ON ar.announcement_id = sa.announcement_id AND ar.user_id = $1
WHERE sa.deleted_at IS NULL
  AND sa.status = 'published'
  AND sa.created_at >= viewer.created_at
  AND ($2::boolean IS NULL OR (ar.user_id IS NOT NULL) = $2)
  AND ($3::timestamptz IS NULL OR sa.created_at < $3
       OR (sa.created_at = $3 AND ('sysann_' || sa.announcement_id) < $4))
ORDER BY sa.created_at DESC, ('sysann_' || sa.announcement_id) DESC
LIMIT $5`

const queryUnreadAnnouncementCount = `SELECT COUNT(*)::int
FROM system_announcements sa
JOIN users viewer ON viewer.user_id = $1
LEFT JOIN announcement_reads ar
  ON ar.announcement_id = sa.announcement_id AND ar.user_id = $1
WHERE sa.deleted_at IS NULL
  AND sa.status = 'published'
  AND sa.created_at >= viewer.created_at
  AND ar.user_id IS NULL`

const queryMarkAnnouncementRead = `INSERT INTO announcement_reads (announcement_id, user_id, read_at)
SELECT sa.announcement_id, $2, now()
FROM system_announcements sa
JOIN users viewer ON viewer.user_id = $2
WHERE sa.announcement_id = ANY($1::text[])
  AND sa.deleted_at IS NULL
  AND sa.status = 'published'
  AND sa.created_at >= viewer.created_at
ON CONFLICT (announcement_id, user_id) DO NOTHING`

const queryMarkAllAnnouncementsRead = `INSERT INTO announcement_reads (announcement_id, user_id, read_at)
SELECT sa.announcement_id, $1, now()
FROM system_announcements sa
JOIN users viewer ON viewer.user_id = $1
WHERE sa.deleted_at IS NULL
  AND sa.status = 'published'
  AND sa.created_at >= viewer.created_at
ON CONFLICT (announcement_id, user_id) DO NOTHING`

// ListAnnouncementNotifications 见 AnnouncementNotificationStore。
func (r *PostgresRepository) ListAnnouncementNotifications(ctx context.Context, userID string, cursor AnnouncementNotifyCursor, isRead *bool, limit int) ([]AnnouncementNotification, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	if limit <= 0 {
		limit = 20
	}
	var cursorAt any
	var cursorID string
	if !cursor.CreatedAt.IsZero() {
		cursorAt = cursor.CreatedAt.UTC()
		cursorID = cursor.NotifyID
	}
	rows, err := queryable.Query(ctx, queryListAnnouncementNotifications, userID, isRead, cursorAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AnnouncementNotification, 0, limit)
	for rows.Next() {
		var item AnnouncementNotification
		if err := rows.Scan(&item.AnnouncementID, &item.Title, &item.Severity, &item.CreatedAt, &item.IsRead); err != nil {
			return nil, err
		}
		item.CreatedAt = item.CreatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// UnreadAnnouncementCount 见 AnnouncementNotificationStore。
func (r *PostgresRepository) UnreadAnnouncementCount(ctx context.Context, userID string) (int, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var count int
	if err := queryable.QueryRow(ctx, queryUnreadAnnouncementCount, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// MarkAnnouncementRead 见 AnnouncementNotificationStore。
func (r *PostgresRepository) MarkAnnouncementRead(ctx context.Context, userID string, announcementIDs []string) error {
	if len(announcementIDs) == 0 {
		return nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAnnouncementRead, announcementIDs, userID)
}

// MarkAllAnnouncementsRead 见 AnnouncementNotificationStore。
func (r *PostgresRepository) MarkAllAnnouncementsRead(ctx context.Context, userID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAllAnnouncementsRead, userID)
}
