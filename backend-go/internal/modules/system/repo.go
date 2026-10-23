package system

import "context"

type Repository interface {
	GetMeta(context.Context) ApiVersionMeta
	Health(context.Context) (int, string, *HealthStatus)
	// AnnouncementsVersion 返回公告域的变更指纹：任何公告的创建/修改/
	// 删除都必须使其变化（PG 取 MAX(updated_at)，内存实现为递增计数）。
	// 供 bootstrap_version 组装缓存失效信号（spec api_spec_system.md
	// 缓存契约：发布新公告时递增 bootstrap_version）。
	AnnouncementsVersion(context.Context) (string, error)
	ListAnnouncements(context.Context, int) ([]AnnouncementRecord, error)
	SearchAnnouncements(context.Context, AnnouncementSearchFilter) (AnnouncementPage, error)
	GetAnnouncement(context.Context, string) (AnnouncementRecord, bool, error)
	CreateAnnouncement(context.Context, AnnouncementCreateInput) (AnnouncementRecord, error)
	UpdateAnnouncement(context.Context, AnnouncementUpdateInput) (AnnouncementRecord, error)
	DeleteAnnouncement(context.Context, string) error
}
