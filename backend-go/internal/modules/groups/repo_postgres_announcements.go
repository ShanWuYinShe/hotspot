package groups

import (
	"context"

	"time"

	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) CreateAnnouncement(ctx context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return GroupAnnouncementItem{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupAnnouncementItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	announcementID, err := idgen.WithPrefix("announce_")
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}
	if err := queryable.Exec(ctx, queryInsertGroupAnnouncement, announcementID, req.GroupID, req.Title, req.Content, creatorID, isPinned, now); err != nil {
		return GroupAnnouncementItem{}, err
	}
	return GroupAnnouncementItem{
		AnnouncementID: announcementID,
		GroupID:        req.GroupID,
		Title:          req.Title,
		Content:        req.Content,
		CreatedBy:      UserInfo{UserID: creatorID, Nickname: creatorID},
		CreatedAt:      now.Format(time.RFC3339),
		IsPinned:       isPinned,
	}, nil
}

func (r *PostgresRepository) ListAnnouncements(ctx context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListGroupAnnouncements, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]GroupAnnouncementItem, 0)
	for rows.Next() {
		var item GroupAnnouncementItem
		var createdByUserID string
		var createdAt time.Time
		var isPinned bool
		if err := rows.Scan(&item.AnnouncementID, &item.GroupID, &item.Title, &item.Content, &createdByUserID, &isPinned, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedBy = UserInfo{UserID: createdByUserID, Nickname: createdByUserID}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.IsPinned = isPinned
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) UpdateAnnouncement(ctx context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	return queryable.Exec(ctx, queryUpdateGroupAnnouncement, req.AnnouncementID, nilOrString(derefStringPtr(req.Title)), nilOrString(derefStringPtr(req.Content)), req.IsPinned, now)
}

func (r *PostgresRepository) DeleteAnnouncement(ctx context.Context, groupID string, announcementID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupAnnouncement, announcementID, time.Now().UTC())
}
