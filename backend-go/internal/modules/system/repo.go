package system

import "context"

type Repository interface {
	GetMeta(context.Context) ApiVersionMeta
	Health(context.Context) (int, string, *HealthStatus)
	ListAnnouncements(context.Context, int) ([]AnnouncementRecord, error)
	SearchAnnouncements(context.Context, AnnouncementSearchFilter) (AnnouncementPage, error)
	CreateAnnouncement(context.Context, AnnouncementCreateInput) (AnnouncementRecord, error)
	UpdateAnnouncement(context.Context, AnnouncementUpdateInput) (AnnouncementRecord, error)
	DeleteAnnouncement(context.Context, string) error
}
