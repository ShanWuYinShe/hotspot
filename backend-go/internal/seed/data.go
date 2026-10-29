package seed

import (
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

// data.go — 把静态规格展开成「按 FK 顺序排列的多表行集合」。
//
// 构建层职责：
//   - 计算派生值（likes_count/comments_count/replies_count/shares_count 等计数
//     列与真实行数保持一致；s2_cell_id / s2_cell_token 用 platform/geo 真实口径）；
//   - 引用 mediaBundle 里的 file_id（头像/瞬间图/聊天图）；
//   - 生成 notifications.payload 等 JSONB 内容。
//
// 不写 SQL、不连数据库，纯内存计算，可被单测直接断言。

// dataset 是待写入的完整行集合（含账号口令占位值）。
type dataset struct {
	steps    []tableData
	accounts []accountSpec
}

// buildDatasetOptions 汇总构建所需的动态输入。
type buildDatasetOptions struct {
	baseTime      time.Time
	appConfigJSON string
	qaPassword    string
	adminPassword string
}

// buildDataset 展开全部表行。返回错误的场景都是数据规格自相矛盾（引用了
// 不存在的媒体键等），在单测与首次执行时即可暴露。
func buildDataset(data seedData, media *mediaBundle, opts buildDatasetOptions) (*dataset, error) {
	payloadCtx, err := newPayloadContext(opts.baseTime, data, media)
	if err != nil {
		return nil, err
	}
	ds := &dataset{accounts: data.accounts}

	usersTable, err := buildUsersTable(opts.baseTime, data.users, media)
	if err != nil {
		return nil, err
	}
	notificationsTable, err := buildNotificationsTable(opts.baseTime, data.notifications, payloadCtx)
	if err != nil {
		return nil, err
	}
	sharesTable := buildSharesTable(opts.baseTime, data.shares)
	reportsTable, err := buildReportsTable(opts.baseTime, data.reports)
	if err != nil {
		return nil, err
	}
	commentsTop, commentsReplies := buildCommentsTables(opts.baseTime, data.moments)
	momentLikes, momentReactions := buildMomentInteractionRows(opts.baseTime, data.moments)

	ds.steps = []tableData{
		usersTable,
		buildUserSettingsTable(data.users),
		buildUserProfileTagsTable(data.users),
		buildUserRelationsTable(opts.baseTime, data.relations),
		buildAuthAccountsTable(data.accounts, opts),
		buildMediaSessionsTable(media),
		buildGroupsTable(opts.baseTime, data.groups, media),
		buildGroupMembersTable(opts.baseTime, data.groups),
		buildGroupAnnouncementsTable(opts.baseTime, data.groups),
		buildGroupBansTable(opts.baseTime, data.groupBans),
		buildGroupInvitesTable(opts.baseTime, data.groupInvites),
		buildGroupApplicationsTable(opts.baseTime, data.groupApplications),
		buildGroupMessagesTable(opts.baseTime, data.groupMessages, media),
		buildGroupReadStatesTable(opts.baseTime, data.groupMessages, data.groupReadStates),
		buildPrivateChatsTable(opts.baseTime, data.chats),
		buildPrivateMessagesTable(opts.baseTime, data.chats, media),
		buildChatUserStatesTable(opts.baseTime, data.chats),
		buildChatReadStatesTable(opts.baseTime, data.chats),
		buildMomentsTable(opts.baseTime, data.moments, data.shares),
		buildMomentImagesTable(opts.baseTime, data.moments, media),
		buildMomentTagsTable(opts.baseTime, data.moments),
		buildMomentMentionsTable(opts.baseTime, data.moments),
		momentLikes,
		buildMomentFavoritesTable(opts.baseTime),
		commentsTop,
		commentsReplies,
		buildCommentLikesTable(opts.baseTime, data.moments),
		momentReactions,
		notificationsTable,
		sharesTable,
		reportsTable,
		buildAdminRolesTable(opts.baseTime, data.adminRoles),
		buildAdminRolePermissionsTable(opts.baseTime, data.adminRoles),
		buildAdminUsersTable(opts.baseTime, data.adminUsers),
		buildAdminUserRolesTable(opts.baseTime, data.adminUsers),
		buildAdminAuditLogsTable(opts.baseTime, data.adminAudits, data.users, media),
		buildBlockedTagsTable(opts.baseTime, data.blockedTags),
		buildSystemAnnouncementsTable(opts.baseTime, data.announcements),
		buildAppConfigsTable(opts.baseTime, opts.appConfigJSON),
	}
	return ds, nil
}

// newPayloadContext 构建通知 payload 的跨行引用上下文。
func newPayloadContext(baseTime time.Time, data seedData, media *mediaBundle) (payloadContext, error) {
	ctx := payloadContext{
		baseTime:        baseTime,
		users:           make(map[string]userSpec, len(data.users)),
		groups:          make(map[string]groupSpec, len(data.groups)),
		moments:         make(map[string]momentSpec, len(data.moments)),
		userAvatarURLs:  make(map[string]string, len(media.userAvatarFileID)),
		groupAvatarURLs: make(map[string]string, len(media.groupAvatarFileID)),
	}
	for _, user := range data.users {
		ctx.users[user.UserID] = user
	}
	for _, group := range data.groups {
		ctx.groups[group.GroupID] = group
	}
	for _, moment := range data.moments {
		ctx.moments[moment.MomentID] = moment
	}
	for userID, fileID := range media.userAvatarFileID {
		ctx.userAvatarURLs[userID] = avatarURL(fileID)
	}
	for groupID, fileID := range media.groupAvatarFileID {
		ctx.groupAvatarURLs[groupID] = avatarURL(fileID)
	}
	return ctx, nil
}

// nullableString 把空串转为 nil（写入 nullable TEXT 列）。
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// nonNilStrings 保证写入 TEXT[] NOT NULL 列的值不是 nil（pgx 会把 nil slice
// 编码成 NULL，违反 NOT NULL 约束）。
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// table 构造函数：列出各表的列集合（列顺序 = 行参数顺序）。

func buildUsersTable(baseTime time.Time, users []userSpec, media *mediaBundle) (tableData, error) {
	table := tableData{
		Table: "users",
		Columns: []column{
			col("user_id"), col("nickname"), col("avatar_url"), col("user_type"), col("bio"),
			col("gender"), col("email"), col("phone"), col("location_name"), col("latitude"),
			col("longitude"), col("presence_status"), col("last_online"), col("created_at"),
			col("status"), col("banned_reason"), col("banned_duration_days"), col("banned_at"),
			col("s2_cell_id"),
		},
	}
	for _, user := range users {
		var avatar any
		if fileID, ok := media.userAvatarFileID[user.UserID]; ok {
			avatar = avatarURL(fileID)
		}
		email := any(nil)
		switch user.UserID {
		case "user_seed_lumos":
			email = "qa@hotspot.local"
		case "user_seed_admin":
			email = "admin@hotspot.local"
		}
		s2CellID := geo.LeafCellID(geo.Point{Latitude: user.Latitude, Longitude: user.Longitude})
		if s2CellID == 0 {
			return table, fmt.Errorf("用户 %s 的坐标非法，S2 cell 为 0", user.UserID)
		}
		bannedReason, bannedDays, bannedAt := any(nil), any(nil), any(nil)
		if user.Status == "banned" {
			bannedReason = nullableString(user.BannedReason)
			bannedDays = user.BannedDays
			bannedAt = baseTime.Add(-user.BannedAgo)
		}
		table.addRow(
			user.UserID, user.Nickname, avatar, "normal", nullableString(user.Bio),
			any(nil), email, any(nil), user.LocationName, user.Latitude,
			user.Longitude, user.Presence, baseTime.Add(-user.LastOnlineAgo), baseTime.Add(-user.CreatedAgo),
			user.Status, bannedReason, bannedDays, bannedAt,
			s2CellID,
		)
	}
	return table, nil
}

func buildUserSettingsTable(users []userSpec) tableData {
	table := tableData{
		Table: "user_settings",
		Columns: []column{
			col("user_id"), col("privacy_level"), col("location_visible"), col("presence_visible"),
			col("social_lists_visible"), col("groups_visible"), col("reacted_moments_visible"),
			col("stats_visible"), col("allow_follow"), col("allow_private_chat"), col("searchable"),
			col("phone_searchable"), col("nickname_searchable"), col("location_searchable"),
			col("nearby_moment_notify"),
		},
	}
	for _, user := range users {
		settings := user.Settings
		table.addRow(
			user.UserID, settings.PrivacyLevel, settings.LocationVisible, settings.PresenceVisible,
			settings.SocialListsVisible, settings.GroupsVisible, settings.ReactedMomentsVisible,
			settings.StatsVisible, settings.AllowFollow, settings.AllowPrivateChat, settings.Searchable,
			settings.PhoneSearchable, settings.NicknameSearchable, settings.LocationSearchable,
			settings.NearbyMomentNotify,
		)
	}
	return table
}

func buildUserProfileTagsTable(users []userSpec) tableData {
	table := tableData{
		Table:   "user_profile_tags",
		Columns: []column{col("user_id"), col("tag_name")},
	}
	for _, user := range users {
		for _, tag := range user.Tags {
			table.addRow(user.UserID, tag)
		}
	}
	return table
}

func buildUserRelationsTable(baseTime time.Time, relations []relationSpec) tableData {
	table := tableData{
		Table: "user_relations",
		Columns: []column{
			col("user_id"), col("target_user_id"), col("is_following"), col("is_blocked"),
			col("remark"), col("followed_at"), col("blocked_at"), col("block_reason"),
		},
	}
	for _, relation := range relations {
		var followedAt, blockedAt any
		if relation.Following {
			followedAt = baseTime.Add(-relation.FollowedAgo)
		}
		if relation.Blocked {
			blockedAt = baseTime.Add(-relation.BlockedAgo)
		}
		table.addRow(
			relation.From, relation.To, relation.Following, relation.Blocked,
			any(nil), followedAt, blockedAt, nullableString(relation.BlockReason),
		)
	}
	return table
}

func buildAuthAccountsTable(accounts []accountSpec, opts buildDatasetOptions) tableData {
	table := tableData{
		Table:   "auth_accounts",
		Columns: []column{col("login_id"), col("user_id"), col("password_hash"), col("created_at"), col("updated_at")},
	}
	for _, account := range accounts {
		plain := legacyAccountPassword
		switch account.UserID {
		case "user_seed_lumos":
			plain = opts.qaPassword
		case "user_seed_admin":
			plain = opts.adminPassword
		}
		createdAt := opts.baseTime.Add(-account.CreatedAgo)
		table.addRow(account.LoginID, account.UserID, passwordPlain(plain), createdAt, createdAt)
	}
	return table
}

func buildMediaSessionsTable(media *mediaBundle) tableData {
	table := tableData{
		Table: "media_upload_sessions",
		Columns: []column{
			col("upload_id"), col("user_id"), col("storage_key"), col("file_name"),
			col("declared_file_size"), col("declared_mime_type"), col("category"), col("md5"),
			col("status"), col("file_id"), col("final_file_size"), col("final_mime_type"),
			col("thumbnail_url"), col("duration"), col("width"), col("height"),
			col("completed_idempotency_key"), col("created_at"), col("completed_at"),
		},
	}
	for _, obj := range media.objects {
		table.addRow(
			obj.UploadID, obj.UserID, obj.StorageKey, obj.FileName,
			obj.FileSize, obj.MimeType, obj.Category, obj.MD5,
			"completed", obj.FileID, obj.FileSize, obj.MimeType,
			any(nil), any(nil), obj.Width, obj.Height,
			any(nil), obj.CreatedAt, obj.CompletedAt,
		)
	}
	return table
}
