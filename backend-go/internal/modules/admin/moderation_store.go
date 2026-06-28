package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
)

type TagRecord struct {
	Name       string
	UsageCount int
	CreatedAt  time.Time
}

type TagPage struct {
	Items []TagRecord
	Next  *tagCursor
}

type TagSearchFilter struct {
	Keyword string
	Limit   int
	Cursor  *tagCursor
}

type ModerationStore interface {
	SearchTags(context.Context, TagSearchFilter) (TagPage, error)
	GetTagDeleteState(context.Context, string) (bool, bool, error)
	DeleteTag(context.Context, string, bool) error
	SearchAnnouncements(context.Context, systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error)
	GetAnnouncement(context.Context, string) (systemmodule.AnnouncementRecord, bool, error)
	CreateAnnouncement(context.Context, systemmodule.AnnouncementCreateInput) (systemmodule.AnnouncementRecord, error)
	UpdateAnnouncement(context.Context, systemmodule.AnnouncementUpdateInput) (systemmodule.AnnouncementRecord, error)
	DeleteAnnouncement(context.Context, string) error
}

var errTagNotFound = errors.New("tag not found")

func normalizeTagName(name string) string {
	return strings.TrimSpace(name)
}
