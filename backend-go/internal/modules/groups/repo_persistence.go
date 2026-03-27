package groups

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	memory := NewInMemoryRepository()
	if factory == nil || factory.Config.PostgresDSN == "" {
		return &PersistenceRepository{impl: memory}
	}
	return &PersistenceRepository{impl: NewPostgresRepository(factory, memory)}
}

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateGroupRequest) (GroupItem, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	return r.impl.Get(ctx, groupID)
}

func (r *PersistenceRepository) Search(ctx context.Context, keyword string) (PaginatedGroups, error) {
	return r.impl.Search(ctx, keyword)
}

func (r *PersistenceRepository) Update(ctx context.Context, userID string, req UpdateGroupRequest) error {
	return r.impl.Update(ctx, userID, req)
}

func (r *PersistenceRepository) Dismiss(ctx context.Context, groupID string) error {
	return r.impl.Dismiss(ctx, groupID)
}

func (r *PersistenceRepository) ListMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	return r.impl.ListMembers(ctx, groupID, role, limit, cursor)
}

func (r *PersistenceRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	return r.impl.SearchMembers(ctx, groupID, keyword, role, limit, cursor)
}

func (r *PersistenceRepository) ListMyGroups(ctx context.Context, userID string, limit int, cursor string) ([]GroupItem, string, error) {
	return r.impl.ListMyGroups(ctx, userID, limit, cursor)
}

func (r *PersistenceRepository) JoinGroup(ctx context.Context, userID string, groupID string, password *string, message *string) (JoinGroupResult, error) {
	return r.impl.JoinGroup(ctx, userID, groupID, password, message)
}

func (r *PersistenceRepository) CreateAnnouncement(ctx context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	return r.impl.CreateAnnouncement(ctx, creatorID, req)
}

func (r *PersistenceRepository) ListAnnouncements(ctx context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	return r.impl.ListAnnouncements(ctx, groupID)
}

func (r *PersistenceRepository) UpdateAnnouncement(ctx context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	return r.impl.UpdateAnnouncement(ctx, groupID, req)
}

func (r *PersistenceRepository) DeleteAnnouncement(ctx context.Context, groupID string, announcementID string) error {
	return r.impl.DeleteAnnouncement(ctx, groupID, announcementID)
}
