package system

import (
	"context"
	"strings"
	"time"
)

// AnnouncementNotifyPrefix 是系统公告虚拟通知的 notify_id 前缀：
// notify_id = prefix + announcement_id。notify 模块按该前缀把 mark_read 的
// 通知 ID 路由回公告已读表，客户端把它当作普通 notify_id 透传/去重即可。
const AnnouncementNotifyPrefix = "sysann_"

// AnnouncementNotifyID 返回公告对应的稳定虚拟 notify_id。
func AnnouncementNotifyID(announcementID string) string {
	return AnnouncementNotifyPrefix + announcementID
}

// ParseAnnouncementNotifyID 判断 notify_id 是否来自系统公告虚拟通知，并返回
// announcement_id。空 ID 或非前缀返回 ok=false。
func ParseAnnouncementNotifyID(notifyID string) (announcementID string, ok bool) {
	if !strings.HasPrefix(notifyID, AnnouncementNotifyPrefix) {
		return "", false
	}
	announcementID = strings.TrimPrefix(notifyID, AnnouncementNotifyPrefix)
	if announcementID == "" {
		return "", false
	}
	return announcementID, true
}

// AnnouncementNotification 是系统公告在通知中心里的虚拟行（不落
// notifications 表）：notify 模块把它转换成 NotifyItem。
type AnnouncementNotification struct {
	AnnouncementID string
	Title          string
	Severity       AnnounceSeverity
	CreatedAt      time.Time
	IsRead         bool
}

// AnnouncementNotifyCursor 是公告虚拟行的 keyset 游标，与通知列表共用
// (created_at DESC, notify_id DESC) 全序；NotifyID 为 AnnouncementNotifyID。
type AnnouncementNotifyCursor struct {
	CreatedAt time.Time
	NotifyID  string
}

// AnnouncementNotificationStore 是 system 模块向 notify 模块暴露的公告通知
// 读能力（所有者接口，避免 notify 直查 system_announcements）。
type AnnouncementNotificationStore interface {
	// ListAnnouncementNotifications 返回当前用户可见的公告虚拟行，按
	// (created_at DESC, notify_id DESC) 排序，最多 limit 条；isRead=nil 表示
	// 不限已读状态。cursor 为零值时从头开始。
	ListAnnouncementNotifications(ctx context.Context, userID string, cursor AnnouncementNotifyCursor, isRead *bool, limit int) ([]AnnouncementNotification, error)
	// UnreadAnnouncementCount 返回当前用户可见且未读的公告数。
	UnreadAnnouncementCount(ctx context.Context, userID string) (int, error)
	// MarkAnnouncementRead 把指定公告标记为当前用户已读（幂等，不存在的公告
	// 跳过）。
	MarkAnnouncementRead(ctx context.Context, userID string, announcementIDs []string) error
	// MarkAllAnnouncementsRead 把当前用户可见的全部公告标记为已读（幂等）。
	MarkAllAnnouncementsRead(ctx context.Context, userID string) error
}
