package system

import "time"

type AnnouncementActor struct {
	UserID    string
	Nickname  string
	AvatarURL string
}

type AnnouncementRecord struct {
	AnnouncementID string
	Title          string
	Content        string
	Severity       AnnounceLevel
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
	if r.ActionURL != "" {
		item.ActionURL = r.ActionURL
	}
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
	Severity AnnounceLevel
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

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 100:
		return 100
	default:
		return limit
	}
}

type AnnouncementCreateInput struct {
	Actor     AnnouncementActor
	Title     string
	Content   string
	Severity  AnnounceLevel
	ActionURL string
	SendPush  bool
}

type AnnouncementUpdateInput struct {
	AnnouncementID string
	Title          *string
	Content        *string
	Severity       *AnnounceLevel
	ActionURL      *string
}
