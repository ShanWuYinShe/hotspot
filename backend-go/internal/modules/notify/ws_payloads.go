package notify

import (
	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/realtime"
)

// ws_payloads.go — notify 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md SystemAnnouncementPayload 镜像）。注意
// NotifyItem.Payload（落库 JSONB）保持 map——那是持久化 DTO，不在本批
// 「WS 出站」范围。

// SystemAnnouncementItemArg 复用 systemmodule.SystemAnnouncementItem 字段，
// 但 action_url 沿用原 map 的「非空才写键」行为，故独立成 struct。
type SystemAnnouncementPayload struct {
	NotifyID string `json:"notify_id"`
	// Announcement 嵌套公告体（action_url 条件键见 AnnouncementBody）。
	Announcement AnnouncementBody `json:"announcement"`
}

type AnnouncementBody struct {
	AnnouncementID string `json:"announcement_id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Severity       string `json:"severity"`
	CreatedAt      string `json:"created_at"`
	ActionURL      string `json:"action_url,omitempty"` // 原 map：非空才写键
}

// SystemAnnouncementRealtimeEvent 构造系统公告 WS 事件：直推路径
// (PushSystemAnnouncement) 与 outbox dispatcher 共用，避免两处 payload 漂移。
func SystemAnnouncementRealtimeEvent(item systemmodule.SystemAnnouncementItem, notifyID string) realtime.Event {
	announcement := AnnouncementBody{
		AnnouncementID: item.AnnouncementID,
		Title:          item.Title,
		Content:        item.Content,
		Severity:       string(item.Severity),
		CreatedAt:      item.CreatedAt,
	}
	if item.ActionURL != nil && *item.ActionURL != "" {
		announcement.ActionURL = *item.ActionURL
	}
	return realtime.Event{Type: realtime.EventTypeSystemAnnouncement, Payload: SystemAnnouncementPayload{NotifyID: notifyID, Announcement: announcement}}
}
