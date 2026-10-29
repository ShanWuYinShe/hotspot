package seed

import (
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

// data_groups.go — 群组相关表行构建（groups / group_members /
// group_announcements / group_bans / group_invites / group_applications /
// group_messages / group_message_read_state）。
//
// groups.s2_cell_token 的层级口径与 groups 模块一致（repo_postgres.go：
// geo.CellTokenAtLevel(point, 14)），s2_cell_id 用叶级 cell。

// groupS2TokenLevel 是 groups.s2_cell_token 使用的 S2 层级（14 级 ≈ 1km 量级，
// 与 groups 模块写入口径一致）。
const groupS2TokenLevel = 14

func buildGroupsTable(baseTime time.Time, groups []groupSpec, media *mediaBundle) tableData {
	table := tableData{
		Table: "groups",
		Columns: []column{
			col("group_id"), col("name"), col("avatar_url"), col("description"),
			col("has_password"), col("password_hash"), col("location_name"), col("latitude"),
			col("longitude"), col("join_method"), col("requires_approval"), col("next_activity_at"),
			col("is_searchable"), col("members_searchable"), col("max_members"), col("owner_id"),
			col("created_at"), col("last_activity_at"), col("tags"), col("s2_cell_token"), col("s2_cell_id"),
		},
	}
	for _, group := range groups {
		var avatar any
		if fileID, ok := media.groupAvatarFileID[group.GroupID]; ok {
			avatar = avatarURL(fileID)
		}
		var passwordHash any
		hasPassword := group.Password != ""
		if hasPassword {
			passwordHash = passwordPlain(group.Password)
		}
		s2CellID := geo.LeafCellID(geo.Point{Latitude: group.Latitude, Longitude: group.Longitude})
		s2CellToken := geo.CellTokenAtLevel(geo.Point{Latitude: group.Latitude, Longitude: group.Longitude}, groupS2TokenLevel)
		if s2CellID == 0 || s2CellToken == "" {
			panic(fmt.Sprintf("群 %s 坐标非法", group.GroupID))
		}
		table.addRow(
			group.GroupID, group.Name, avatar, nullableString(group.Description),
			hasPassword, passwordHash, group.LocationName, group.Latitude,
			group.Longitude, group.JoinMethod, group.RequiresApproval, nullableString(group.NextActivityAt),
			group.IsSearchable, group.MembersSearchable, group.MaxMembers, group.OwnerID,
			baseTime.Add(-group.CreatedAgo), baseTime.Add(-group.LastActivityAgo), nonNilStrings(group.Tags), s2CellToken, s2CellID,
		)
	}
	return table
}

// groupMemberEntry 是展开后的群成员（含角色与加入顺序）。
type groupMemberEntry struct {
	GroupID  string
	UserID   string
	Role     string // owner / admin / member
	JoinedAt time.Time
}

// expandGroupMembers 展开群成员：owner → admins → members。
// joined_at 以群创建时间为起点按顺序错开（每行 +1 小时），保证列表顺序稳定且
// 不会晚于当前时刻。
func expandGroupMembers(baseTime time.Time, groups []groupSpec) []groupMemberEntry {
	entries := make([]groupMemberEntry, 0, len(groups)*4)
	for _, group := range groups {
		created := baseTime.Add(-group.CreatedAgo)
		order := 0
		add := func(userID, role string) {
			if userID == "" {
				return
			}
			entries = append(entries, groupMemberEntry{
				GroupID:  group.GroupID,
				UserID:   userID,
				Role:     role,
				JoinedAt: created.Add(time.Duration(order) * time.Hour),
			})
			order++
		}
		add(group.OwnerID, "owner")
		for _, adminID := range group.AdminIDs {
			add(adminID, "admin")
		}
		for _, memberID := range group.MemberIDs {
			add(memberID, "member")
		}
	}
	return entries
}

func buildGroupMembersTable(baseTime time.Time, groups []groupSpec) tableData {
	table := tableData{
		Table:   "group_members",
		Columns: []column{col("group_id"), col("user_id"), col("role"), col("joined_at")},
	}
	for _, entry := range expandGroupMembers(baseTime, groups) {
		table.addRow(entry.GroupID, entry.UserID, entry.Role, entry.JoinedAt)
	}
	return table
}

func buildGroupAnnouncementsTable(baseTime time.Time, groups []groupSpec) tableData {
	table := tableData{
		Table: "group_announcements",
		Columns: []column{
			col("announcement_id"), col("group_id"), col("title"), col("content"),
			col("created_by_user_id"), col("is_pinned"), col("created_at"),
		},
	}
	for _, group := range groups {
		for _, announcement := range group.Announcements {
			table.addRow(
				announcement.AnnouncementID, group.GroupID, announcement.Title, announcement.Content,
				announcement.CreatorID, announcement.Pinned, baseTime.Add(-announcement.CreatedAgo),
			)
		}
	}
	return table
}

func buildGroupBansTable(baseTime time.Time, bans []groupBanSpec) tableData {
	table := tableData{
		Table: "group_bans",
		Columns: []column{
			col("group_id"), col("user_id"), col("banned_by_user_id"), col("reason"), col("banned_at"),
		},
	}
	for _, ban := range bans {
		table.addRow(ban.GroupID, ban.UserID, nullableString(ban.BannedByID), nullableString(ban.Reason), baseTime.Add(-ban.BannedAgo))
	}
	return table
}

func buildGroupInvitesTable(baseTime time.Time, invites []groupInviteSpec) tableData {
	table := tableData{
		Table: "group_invites",
		Columns: []column{
			col("invite_id"), col("group_id"), col("inviter_user_id"), col("invitee_user_id"),
			col("message"), col("status"), col("created_at"), col("expires_at"),
		},
	}
	for _, invite := range invites {
		var expiresAt any
		if invite.ExpiresIn > 0 {
			expiresAt = baseTime.Add(invite.ExpiresIn)
		}
		table.addRow(
			invite.InviteID, invite.GroupID, invite.InviterID, invite.InviteeID,
			nullableString(invite.Message), invite.Status, baseTime.Add(-invite.CreatedAgo), expiresAt,
		)
	}
	return table
}

func buildGroupApplicationsTable(baseTime time.Time, applications []groupApplicationSpec) tableData {
	table := tableData{
		Table: "group_applications",
		Columns: []column{
			col("application_id"), col("group_id"), col("applicant_user_id"), col("message"),
			col("status"), col("created_at"), col("handled_at"), col("handled_by_user_id"),
		},
	}
	for _, application := range applications {
		var handledAt, handledBy any
		if application.HandledAgo > 0 {
			handledAt = baseTime.Add(-application.HandledAgo)
		}
		handledBy = nullableString(application.HandledBy)
		table.addRow(
			application.ApplicationID, application.GroupID, application.ApplicantID, nullableString(application.Message),
			application.Status, baseTime.Add(-application.CreatedAgo), handledAt, handledBy,
		)
	}
	return table
}

func buildGroupMessagesTable(baseTime time.Time, messages []groupMessageSpec, media *mediaBundle) tableData {
	table := tableData{
		Table: "group_messages",
		Columns: []column{
			col("message_id"), col("group_id"), col("sender_user_id"), col("type"), col("content"),
			col("share_code"), col("file_id"), col("reply_to_message_id"), col("mentions"), col("sent_at"),
			col("is_recalled"), col("recalled_at"), col("content_prev"), col("edited_at"), col("recalled_by"),
		},
	}
	for _, message := range messages {
		var sender any
		if message.SenderID != "" {
			sender = message.SenderID
		}
		content := any(message.Content)
		if message.Recalled {
			content = nil
		}
		var fileID any
		if message.MediaKey != "" {
			resolved, ok := media.chatImageFileIDs[message.MediaKey]
			if !ok {
				panic(fmt.Sprintf("群消息 %s 引用了未生成的聊天媒体 %s", message.MessageID, message.MediaKey))
			}
			fileID = resolved
		}
		var recalledAt any
		if message.Recalled {
			recalledAt = baseTime.Add(-message.RecalledAgo)
		}
		var editedAt any
		if message.EditedAgo > 0 {
			editedAt = baseTime.Add(-message.EditedAgo)
		}
		table.addRow(
			message.MessageID, message.GroupID, sender, message.Type, content,
			nullableString(message.ShareCode), fileID, nullableString(message.ReplyTo),
			nonNilStrings(message.Mentions), baseTime.Add(-message.SentAgo),
			message.Recalled, recalledAt, nullableString(message.ContentPrev), editedAt,
			nullableString(message.RecalledBy),
		)
	}
	return table
}

func buildGroupReadStatesTable(baseTime time.Time, messages []groupMessageSpec, states []groupReadStateSpec) tableData {
	sentAtByMessageID := make(map[string]time.Time, len(messages))
	for _, message := range messages {
		sentAtByMessageID[message.MessageID] = baseTime.Add(-message.SentAgo)
	}
	table := tableData{
		Table: "group_message_read_state",
		Columns: []column{
			col("group_id"), col("user_id"), col("last_read_message_id"),
			col("last_read_sent_at"), col("updated_at"),
		},
	}
	for _, state := range states {
		lastReadSentAt, ok := sentAtByMessageID[state.LastReadMessageID]
		if !ok {
			panic(fmt.Sprintf("群已读游标引用了不存在的消息 %s", state.LastReadMessageID))
		}
		table.addRow(state.GroupID, state.UserID, state.LastReadMessageID, lastReadSentAt, lastReadSentAt)
	}
	return table
}
