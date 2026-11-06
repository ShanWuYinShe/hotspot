package system

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/types"
)

type AnnouncementActor struct {
	UserID    string
	Nickname  string
	AvatarURL string
}

type AnnouncementRecord struct {
	AnnouncementID string
	Title          string
	Content        string
	Severity       AnnounceSeverity
	ActionURL      string
	CreatedAt      time.Time
	CreatedBy      AnnouncementActor
	SendPush       bool
	PushSentAt     *time.Time
	ViewCount      int
	DeletedAt      *time.Time
}

func (r AnnouncementRecord) PublicItem() SystemAnnouncementItem {
	item := SystemAnnouncementItem{
		AnnouncementID: r.AnnouncementID,
		Title:          r.Title,
		Content:        r.Content,
		Severity:       r.Severity,
		CreatedAt:      r.CreatedAt.UTC().Format(time.RFC3339),
	}
	item.ActionURL = types.StrPtr(r.ActionURL)
	return item
}

func (r AnnouncementRecord) IsDeleted() bool {
	return r.DeletedAt != nil
}

type AnnouncementCursor struct {
	CreatedAt      time.Time
	AnnouncementID string
}

type AnnouncementSearchFilter struct {
	Severity AnnounceSeverity
	Limit    int
	Cursor   *AnnouncementCursor
}

type AnnouncementPage struct {
	Items []AnnouncementRecord
	Next  *AnnouncementCursor
}

func ErrAnnouncementNotFound() error {
	return errAnnouncementNotFound
}

type AnnouncementCreateInput struct {
	Actor     AnnouncementActor
	Title     string
	Content   string
	Severity  AnnounceSeverity
	ActionURL string
	SendPush  bool
}

type AnnouncementUpdateInput struct {
	AnnouncementID string
	Title          *string
	Content        *string
	Severity       *AnnounceSeverity
	ActionURL      *string
}

// AnnouncementAdminStore 是管理端公告管理能力的最小接口：公告数据归 system 模块
// 所有，admin 模块经此接口（由 system.Service 实现）读写公告，不再自建
// system.Repository 副本（架构审计 P0-1 隐藏实例）。
type AnnouncementAdminStore interface {
	SearchAnnouncements(context.Context, AnnouncementSearchFilter) (AnnouncementPage, error)
	GetAnnouncement(context.Context, string) (AnnouncementRecord, bool, error)
	CreateAnnouncement(context.Context, AnnouncementCreateInput) (AnnouncementRecord, error)
	UpdateAnnouncement(context.Context, AnnouncementUpdateInput) (AnnouncementRecord, error)
	DeleteAnnouncement(context.Context, string) error
}
