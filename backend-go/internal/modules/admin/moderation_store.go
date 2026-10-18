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
	// IsBlocked 标记该标签是否在 admin_blocked_tags 中（软屏蔽）。管理端据此
	// 渲染「屏蔽 / 解除屏蔽」中的适用动作（spec tags/search is_blocked）。
	IsBlocked bool
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
	// SetTagBan 设置标签屏蔽状态（spec api_spec_admin.md admin/tags/set_ban）：
	// active=true 屏蔽（hardDelete=true 时一并物理删除历史引用）；active=false
	// 解屏蔽（对未屏蔽标签 no-op 成功）。屏蔽不存在的标签返回 errTagNotFound。
	SetTagBan(context.Context, string, bool, bool) error
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
