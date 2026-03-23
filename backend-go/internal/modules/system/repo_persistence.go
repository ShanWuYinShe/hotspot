package system

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

func (r *PersistenceRepository) GetMeta(ctx context.Context) ApiVersionMeta {
	return r.impl.GetMeta(ctx)
}

func (r *PersistenceRepository) Health(ctx context.Context) (int, string, *HealthStatus) {
	return r.impl.Health(ctx)
}

func (r *PersistenceRepository) ListAnnouncements(ctx context.Context, limit int) ([]AnnouncementRecord, error) {
	return r.impl.ListAnnouncements(ctx, limit)
}

func (r *PersistenceRepository) SearchAnnouncements(ctx context.Context, filter AnnouncementSearchFilter) (AnnouncementPage, error) {
	return r.impl.SearchAnnouncements(ctx, filter)
}

func (r *PersistenceRepository) CreateAnnouncement(ctx context.Context, input AnnouncementCreateInput) (AnnouncementRecord, error) {
	return r.impl.CreateAnnouncement(ctx, input)
}

func (r *PersistenceRepository) UpdateAnnouncement(ctx context.Context, input AnnouncementUpdateInput) (AnnouncementRecord, error) {
	return r.impl.UpdateAnnouncement(ctx, input)
}

func (r *PersistenceRepository) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	return r.impl.DeleteAnnouncement(ctx, announcementID)
}
