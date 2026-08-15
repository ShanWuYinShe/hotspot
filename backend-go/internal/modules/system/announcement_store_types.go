package system

import (
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
