package seed

import "time"

// specs.go — 开发种子数据的「数据规格」层：纯静态描述（不含 SQL、不含 IO）。
//
// 与 .workwork/seed-spec.md §3 的条目一一对应；数据的事实口径以数据库迁移
// 与各模块 repo 的读写查询为准（规格文档与代码冲突处以代码为准，冲突记录
// 见 backend-go/AGENTS.md「开发种子数据」与交付说明）。
//
// 时间一律用相对基准时刻的偏移（Ago）表达：真实写入时基准时刻取执行时刻向
// 下取整到小时，因此同一小时内重复 --reset 得到同一组时间线，且数据永远
// 看起来是「最近几天」的。

const (
	oneHour = time.Hour
	oneDay  = 24 * time.Hour
)

// seedAnchor* 是种子数据的默认浏览锚点（徐州云龙湖），与 user_seed_lumos 位置一致。
// app_configs 的 defaults.default_location 与 default_group_settings.location 都指向它，
// 保证全新环境开箱即用时「附近/地图/建群默认位置」都落在种子数据附近——否则服务端默认
// 锚点（南天门 35.15,112.68）与徐州数据相距 400+km，附近页全空（2026-10-29 联调实测）。
const (
	seedAnchorLocationName = "云龙湖"
	seedAnchorLatitude     = 34.2604
	seedAnchorLongitude    = 117.1855
)

// settingsSpec 对应 user_settings 行（迁移 001 + 033）。字段零值不做缺省
// 填充：静态规格里显式给出完整值，避免「忘了写 → 落库为 false」。
type settingsSpec struct {
	PrivacyLevel          string
	LocationVisible       bool
	PresenceVisible       bool
	SocialListsVisible    bool
	GroupsVisible         bool
	ReactedMomentsVisible bool
	StatsVisible          bool
	AllowFollow           bool
	AllowPrivateChat      string
	Searchable            bool
	PhoneSearchable       bool
	NicknameSearchable    bool
	LocationSearchable    bool
	NearbyMomentNotify    bool
}

// defaultSettings 返回普通用户的默认设置（与迁移 001 的列默认值一致，
// allow_private_chat=everyone）；lumos 额外开启附近瞬间通知。
func defaultSettings() settingsSpec {
	return settingsSpec{
		PrivacyLevel:          "public",
		LocationVisible:       true,
		PresenceVisible:       true,
		SocialListsVisible:    true,
		GroupsVisible:         true,
		ReactedMomentsVisible: true,
		StatsVisible:          true,
		AllowFollow:           true,
		AllowPrivateChat:      "everyone",
		Searchable:            true,
		PhoneSearchable:       false,
		NicknameSearchable:    true,
		LocationSearchable:    true,
		NearbyMomentNotify:    false,
	}
}

// userSpec 对应 users 行（迁移 001 + 038）。
type userSpec struct {
	UserID        string
	Nickname      string
	AvatarAsset   string // "" = 无头像（avatar_url 为 NULL，前端走首字母兜底）
	Bio           string
	LocationName  string
	Latitude      float64
	Longitude     float64
	Presence      string // online / offline
	Status        string // normal / banned
	BannedReason  string
	BannedDays    int
	BannedAgo     time.Duration
	Tags          []string
	CreatedAgo    time.Duration
	LastOnlineAgo time.Duration
	Settings      settingsSpec
}

// relationSpec 对应 user_relations 行（迁移 001）：单向边。
// 互关 = 两条 Following 边；拉黑 = is_blocked=true 的边（与关注正交）。
type relationSpec struct {
	From        string
	To          string
	Following   bool
	Blocked     bool
	FollowedAgo time.Duration
	BlockedAgo  time.Duration
	BlockReason string
}

// accountSpec 对应 auth_accounts 行（迁移 007）。Password 明文只存在于进程
// 内存，写入前用 bcrypt 哈希；不会被打印到日志。
type accountSpec struct {
	LoginID    string
	UserID     string
	Password   string
	CreatedAgo time.Duration
}

// reactionSpec 对应 reactions 行（迁移 031，target 由调用方决定）。
type reactionSpec struct {
	UserID string
	Emoji  string
}

// commentSpec 对应 comments 行（迁移 003 + 011）。
// ParentID 非空表示二级回复（父评论的 replies_count 由构建层统计）。
type commentSpec struct {
	CommentID  string
	UserID     string
	ParentID   string
	Content    string
	Likes      []string // comment_likes 行
	CreatedAgo time.Duration
}

// momentSpec 对应 moments / moment_images / moment_tags / moment_mentions /
// moment_likes / moment_favorites / comments / reactions 的多表组合。
type momentSpec struct {
	MomentID     string
	AuthorID     string
	Description  string
	LocationName string
	Latitude     float64
	Longitude    float64
	CreatedAgo   time.Duration
	PrivacyLevel string // public / private
	IsDeleted    bool   // 软删（迁移 037：is_deleted + description 清空）
	ImageCount   int    // 瞬间图数量（素材由 mediaplan 轮换分配）
	Tags         []string
	Mentions     []string // moment_mentions.mentioned_user_id
	Reactions    []reactionSpec
	Comments     []commentSpec
}

// groupAnnouncementSpec 对应 group_announcements 行（迁移 027）。
type groupAnnouncementSpec struct {
	AnnouncementID string
	Title          string
	Content        string
	CreatorID      string
	Pinned         bool
	CreatedAgo     time.Duration
}

// groupSpec 对应 groups 行（迁移 004 + 036/041/042/043）。
type groupSpec struct {
	GroupID           string
	Name              string
	Description       string
	AvatarAsset       string
	OwnerID           string
	AdminIDs          []string
	MemberIDs         []string // 不含 owner/admin
	JoinMethod        string   // direct / password / invite_only
	RequiresApproval  bool
	Password          string // 非空 = password 群，写 bcrypt 到 password_hash
	IsSearchable      bool
	MembersSearchable bool
	MaxMembers        int
	Tags              []string
	LocationName      string
	Latitude          float64
	Longitude         float64
	NextActivityAt    string // groups.next_activity_at（TEXT，空 = NULL）
	CreatedAgo        time.Duration
	LastActivityAgo   time.Duration
	Announcements     []groupAnnouncementSpec
}

// groupMessageSpec 对应 group_messages 行（迁移 004 + 044 + 054）。
// Type='system' 时 SenderID 为空（sender_user_id NULL）。
type groupMessageSpec struct {
	MessageID   string
	GroupID     string
	SenderID    string
	Type        string // text / image / share / system
	Content     string
	ShareCode   string
	MediaKey    string // chat 媒体查找键（mediaplan 生成）
	ReplyTo     string
	Mentions    []string
	SentAgo     time.Duration
	Recalled    bool
	RecalledAgo time.Duration
	RecalledBy  string
	ContentPrev string // 撤回/编辑前的内容
	EditedAgo   time.Duration
}

// groupReadStateSpec 对应 group_message_read_state 行（迁移 018）。
type groupReadStateSpec struct {
	GroupID           string
	UserID            string
	LastReadMessageID string
}

// chatMessageSpec 对应 private_messages 行（迁移 004 / 016）。
type chatMessageSpec struct {
	MessageID   string
	SenderID    string
	Type        string // text / image / share
	Content     string
	MediaKey    string
	ShareCode   string
	ReplyTo     string
	SentAgo     time.Duration
	Recalled    bool
	RecalledAgo time.Duration
	RecalledBy  string
	ContentPrev string
	EditedAgo   time.Duration
}

// chatUserStateSpec 对应 private_chat_user_state 行（迁移 019）。
type chatUserStateSpec struct {
	UserID string
	Pinned bool
	Muted  bool
}

// chatReadStateSpec 对应 private_chat_read_state 行（迁移 016）。
type chatReadStateSpec struct {
	UserID            string
	LastReadMessageID string
}

// chatSpec 对应 private_chats + private_messages + 读状态/用户状态。
// UserA/UserB 的先后顺序不重要，构建层用 chatUsers() 归一。
type chatSpec struct {
	UserID       string // 会话视角用户（QA 主账号 lumos）
	TargetUserID string
	CreatedAgo   time.Duration
	Messages     []chatMessageSpec
	States       []chatUserStateSpec
	ReadStates   []chatReadStateSpec
}

// notifySpec 对应 notifications 行（迁移 005 + 050）。Payload 在构建层用
// payloadContext 生成（需要其它 seed 行的昵称/头像/摘要）。
type notifySpec struct {
	NotifyID   string
	UserID     string
	Type       string
	IsRead     bool
	CreatedAgo time.Duration
	Payload    func(payloadContext) map[string]any
}

// shareSpec 对应 shares 行（迁移 006）。
type shareSpec struct {
	ShareID    string
	ShareCode  string
	CreatorID  string
	ShareType  string // user / group / moment / location
	Payload    map[string]any
	CreatedAgo time.Duration
	ExpiresIn  time.Duration // 0 = 不过期（expires_at NULL）
	Revoked    bool
}

// reportSpec 对应 reports 行（迁移 006 + 039 + 051）。
type reportSpec struct {
	ReportID    string
	CreatorID   string
	TargetType  string // user / moment / group / message
	TargetID    string
	Reason      string // ReportReason 枚举
	Description string
	Status      string // pending / resolved / dismissed
	CreatedAgo  time.Duration
	HandledAgo  time.Duration // 0 = 未处理
	HandledBy   string
	HandleNote  string
	// MessageSnapshot 仅 target_type='message' 时非空：服务端冻结的消息快照
	// （api_spec_types.md ReportSnapshot）。
	MessageSnapshot func(baseTime time.Time) map[string]any
}

// adminRoleSpec 对应 admin_roles + admin_role_permissions 行（迁移 008）。
type adminRoleSpec struct {
	RoleID      string
	Name        string
	Description string
	Permissions []string
	CreatedAgo  time.Duration
}

// adminUserSpec 对应 admin_users + admin_user_roles 行（迁移 008）。
type adminUserSpec struct {
	UserID     string
	GrantAll   bool
	RoleIDs    []string
	GrantedAgo time.Duration
}

// auditLogSpec 对应 admin_audit_logs 行（迁移 009）。
type auditLogSpec struct {
	LogID         string
	Action        string
	OperatorID    string
	TargetType    string
	TargetID      string
	TargetName    string
	Details       map[string]any
	IPLocation    string
	IPAddress     string
	DeviceSummary string
	CreatedAgo    time.Duration
}

// blockedTagSpec 对应 admin_blocked_tags 行（迁移 009）。
type blockedTagSpec struct {
	Name       string
	BlockedAgo time.Duration
}

// systemAnnouncementSpec 对应 system_announcements 行（迁移 009）。
type systemAnnouncementSpec struct {
	AnnouncementID string
	Title          string
	Content        string
	Severity       string // info / warning / error
	ActionURL      string
	ViewCount      int
	CreatedAgo     time.Duration
}

// seedData 是全部静态规格的聚合。
type seedData struct {
	users             []userSpec
	relations         []relationSpec
	accounts          []accountSpec
	moments           []momentSpec
	groups            []groupSpec
	groupMessages     []groupMessageSpec
	groupReadStates   []groupReadStateSpec
	groupBans         []groupBanSpec
	groupInvites      []groupInviteSpec
	groupApplications []groupApplicationSpec
	chats             []chatSpec
	notifications     []notifySpec
	shares            []shareSpec
	reports           []reportSpec
	adminRoles        []adminRoleSpec
	adminUsers        []adminUserSpec
	adminAudits       []auditLogSpec
	blockedTags       []blockedTagSpec
	announcements     []systemAnnouncementSpec
}

// buildSeedData 组装全部静态规格。纯函数，无随机、无 IO。
func buildSeedData() seedData {
	return seedData{
		users:             seedUsers(),
		relations:         seedRelations(),
		accounts:          seedAccounts(),
		moments:           seedMoments(),
		groups:            seedGroups(),
		groupMessages:     seedGroupMessages(),
		groupReadStates:   seedGroupReadStates(),
		groupBans:         seedGroupBans(),
		groupInvites:      seedGroupInvites(),
		groupApplications: seedGroupApplications(),
		chats:             seedChats(),
		notifications:     seedNotifications(),
		shares:            seedShares(),
		reports:           seedReports(),
		adminRoles:        seedAdminRoles(),
		adminUsers:        seedAdminUsers(),
		adminAudits:       seedAdminAudits(),
		blockedTags:       seedBlockedTags(),
		announcements:     seedSystemAnnouncements(),
	}
}
