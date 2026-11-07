package notify

import (
	"context"

	systemmodule "hotspot/backend-go/internal/modules/system"
)

// AnnouncementNotificationSource 是 system 模块提供的公告虚拟通知读能力
// （消费者侧接口）。notify 不再直查 system_announcements，公告本体/已读状态
// 的读写全部经该接口，保持模块数据所有权。
type AnnouncementNotificationSource interface {
	ListAnnouncementNotifications(ctx context.Context, userID string, cursor systemmodule.AnnouncementNotifyCursor, isRead *bool, limit int) ([]systemmodule.AnnouncementNotification, error)
	UnreadAnnouncementCount(ctx context.Context, userID string) (int, error)
	MarkAnnouncementRead(ctx context.Context, userID string, announcementIDs []string) error
	MarkAllAnnouncementsRead(ctx context.Context, userID string) error
}
