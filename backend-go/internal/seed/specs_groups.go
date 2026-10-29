package seed

import "time"

// specs_groups.go — 群组规格（seed-spec §3.3）。
//
// 覆盖四种准入形态：
//   - group_seed_1：direct + 完整成员/公告/群消息（含 @提及、撤回、已编辑、图片、分享卡片）
//   - group_seed_2：password（口令 run123456，bcrypt 落 password_hash；lumos 未加入）
//   - group_seed_3：invite_only + requires_approval（lumos 有待审申请）
//   - group_seed_4：direct + requires_approval（lumos 收到待处理邀请）
//
// 群消息的 10 条按时间正序排列；lumos 的已读游标落在第 7 条，因此第 8-10 条
// 构成 3 条未读（group_messages 未读口径 = 已读游标之后且未撤回的消息）。

// seedGroups 返回 4 个群组规格。
func seedGroups() []groupSpec {
	return []groupSpec{
		{
			GroupID: "group_seed_1", Name: "回龙窝胶片会",
			Description: "每周一次胶片扫街，暗房共洗。", AvatarAsset: "group_1.jpg",
			OwnerID: "user_seed_marco", AdminIDs: []string{"user_seed_sophia"},
			MemberIDs:  []string{"user_seed_lumos", "user_seed_mika", "user_seed_chen", "user_seed_lin", "user_seed_xiaoman"},
			JoinMethod: "direct", RequiresApproval: false,
			IsSearchable: true, MembersSearchable: true, MaxMembers: 200,
			Tags:         []string{"胶片", "摄影", "扫街"},
			LocationName: "回龙窝", Latitude: 34.2578, Longitude: 117.1817,
			NextActivityAt: "周六 15:00 回龙窝南门集合",
			CreatedAgo:     60 * oneDay, LastActivityAgo: oneHour,
			Announcements: []groupAnnouncementSpec{
				{AnnouncementID: "ann_seed_1", Title: "本周六胶片扫街", Content: "15:00 回龙窝南门集合，记得带 135 胶卷，暗房当天开放。", CreatorID: "user_seed_marco", Pinned: true, CreatedAgo: 2 * oneDay},
				{AnnouncementID: "ann_seed_2", Title: "暗房使用须知", Content: "冲洗液请按标签归位，废液统一回收，离开前检查水洗槽。", CreatorID: "user_seed_sophia", Pinned: false, CreatedAgo: 5 * oneDay},
			},
		},
		{
			GroupID: "group_seed_2", Name: "云龙湖晨跑团",
			Description: "清晨六点半，环湖慢跑。", AvatarAsset: "group_2.jpg",
			OwnerID:    "user_seed_sophia",
			MemberIDs:  []string{"user_seed_mika", "user_seed_lin"},
			JoinMethod: "password", RequiresApproval: false, Password: groupSeedPassword,
			IsSearchable: true, MembersSearchable: true, MaxMembers: 200,
			Tags:         []string{"跑步", "晨练"},
			LocationName: "云龙湖大堤", Latitude: 34.2570, Longitude: 117.1930,
			NextActivityAt: "周日 6:30 云龙湖大堤集合",
			CreatedAgo:     45 * oneDay, LastActivityAgo: 3 * oneHour,
			Announcements: []groupAnnouncementSpec{
				{AnnouncementID: "ann_seed_4", Title: "晨跑路线与配速", Content: "新手 6:30 集合，配速 6:00-6:30，全程 5km，跑完一起喝豆浆。", CreatorID: "user_seed_sophia", Pinned: true, CreatedAgo: 10 * oneDay},
			},
		},
		{
			GroupID: "group_seed_3", Name: "户部山弹唱角",
			Description: "带琴来就能唱，弹唱为主。", AvatarAsset: "group_3.jpg",
			OwnerID:    "user_seed_marco",
			JoinMethod: "invite_only", RequiresApproval: true,
			IsSearchable: true, MembersSearchable: true, MaxMembers: 100,
			Tags:         []string{"音乐", "弹唱"},
			LocationName: "户部山", Latitude: 34.2567, Longitude: 117.1843,
			NextActivityAt: "每月最后一个周六 19:00 户部山戏马台",
			CreatedAgo:     30 * oneDay, LastActivityAgo: 2 * oneDay,
		},
		{
			GroupID: "group_seed_4", Name: "黄楼夜读角",
			Description: "每周共读一本书，安静地聊。", AvatarAsset: "group_4.jpg",
			OwnerID: "user_seed_chen", AdminIDs: []string{"user_seed_sophia"},
			JoinMethod: "direct", RequiresApproval: true,
			IsSearchable: true, MembersSearchable: true, MaxMembers: 100,
			Tags:         []string{"阅读", "夜读"},
			LocationName: "户部山", Latitude: 34.2572, Longitude: 117.1848,
			NextActivityAt: "周三 20:00 黄楼共读《城与不确定的墙》",
			CreatedAgo:     20 * oneDay, LastActivityAgo: oneDay,
			Announcements: []groupAnnouncementSpec{
				{AnnouncementID: "ann_seed_3", Title: "本月共读书目", Content: "《城与不确定的墙》，周三晚 8 点黄楼见，自带一本书来交换。", CreatorID: "user_seed_chen", Pinned: true, CreatedAgo: 3 * oneDay},
			},
		},
	}
}

// seedGroupMessages 返回 group_seed_1 的 10 条群消息（其余群无消息）。
func seedGroupMessages() []groupMessageSpec {
	groupID := "group_seed_1"
	return []groupMessageSpec{
		{MessageID: "gmsg_seed_01", GroupID: groupID, SenderID: "", Type: "system",
			Content: "欢迎 @鹿鸣 加入群聊", SentAgo: 6 * oneDay},
		{MessageID: "gmsg_seed_02", GroupID: groupID, SenderID: "user_seed_marco", Type: "text",
			Content: "", SentAgo: 5 * oneDay, Recalled: true, RecalledAgo: 5*oneDay - 30*oneMinute,
			RecalledBy: "user_seed_marco", ContentPrev: "这周六天气不错，扫街走起？"},
		{MessageID: "gmsg_seed_03", GroupID: groupID, SenderID: "user_seed_sophia", Type: "text",
			Content: "报名！我带两卷 Portra 400。", SentAgo: 5*oneDay - 59*oneMinute},
		{MessageID: "gmsg_seed_04", GroupID: groupID, SenderID: "user_seed_lumos", Type: "text",
			Content: "算我一个，刚好试下新镜头。", SentAgo: 5*oneDay - 58*oneMinute},
		{MessageID: "gmsg_seed_05", GroupID: groupID, SenderID: "user_seed_mika", Type: "image",
			MediaKey: "gmsg_img_1", SentAgo: 4 * oneDay},
		{MessageID: "gmsg_seed_06", GroupID: groupID, SenderID: "user_seed_marco", Type: "share",
			ShareCode: shareCodeMomentSeed, SentAgo: 4*oneDay - 59*oneMinute},
		{MessageID: "gmsg_seed_07", GroupID: groupID, SenderID: "user_seed_chen", Type: "text",
			Content: "@鹿鸣 上次说的暗房几点开？", Mentions: []string{"user_seed_lumos"}, SentAgo: 3 * oneDay},
		{MessageID: "gmsg_seed_08", GroupID: groupID, SenderID: "user_seed_lin", Type: "text",
			Content: "我把暗房钥匙放门卫了，晚上七点后都能用。", ContentPrev: "我把钥匙放门卫了。",
			SentAgo: 2 * oneDay, EditedAgo: 2*oneDay - oneHour},
		{MessageID: "gmsg_seed_09", GroupID: groupID, SenderID: "user_seed_xiaoman", Type: "text",
			Content: "周六我也来，带点 120 的卷。", SentAgo: oneDay},
		{MessageID: "gmsg_seed_10", GroupID: groupID, SenderID: "user_seed_marco", Type: "text",
			Content: "那就本周六 15:00，回龙窝南门见。", SentAgo: oneHour},
	}
}

// seedGroupReadStates 返回群已读游标。
// lumos 停在第 7 条 → 第 8/9/10 条为未读（3 条，符合规格）；其余成员读到最新。
func seedGroupReadStates() []groupReadStateSpec {
	return []groupReadStateSpec{
		{GroupID: "group_seed_1", UserID: "user_seed_lumos", LastReadMessageID: "gmsg_seed_07"},
		{GroupID: "group_seed_1", UserID: "user_seed_marco", LastReadMessageID: "gmsg_seed_10"},
		{GroupID: "group_seed_1", UserID: "user_seed_sophia", LastReadMessageID: "gmsg_seed_10"},
		{GroupID: "group_seed_1", UserID: "user_seed_mika", LastReadMessageID: "gmsg_seed_10"},
		{GroupID: "group_seed_1", UserID: "user_seed_chen", LastReadMessageID: "gmsg_seed_10"},
		{GroupID: "group_seed_1", UserID: "user_seed_lin", LastReadMessageID: "gmsg_seed_10"},
		{GroupID: "group_seed_1", UserID: "user_seed_xiaoman", LastReadMessageID: "gmsg_seed_10"},
	}
}

// seedGroupBans 返回群封禁记录（覆盖群管理页封禁名单）。
func seedGroupBans() []groupBanSpec {
	return []groupBanSpec{
		{GroupID: "group_seed_1", UserID: "user_seed_banned", BannedByID: "user_seed_lumos", Reason: "群内发布广告（seed 数据）", BannedAgo: 4 * oneDay},
	}
}

// seedGroupInvites 返回入群邀请（group_seed_4 → lumos 待处理）。
func seedGroupInvites() []groupInviteSpec {
	return []groupInviteSpec{
		{
			InviteID: "invite_seed_1", GroupID: "group_seed_4", InviterID: "user_seed_chen", InviteeID: "user_seed_lumos",
			Message: "夜读角本周读《城与不确定的墙》，来吗？", Status: "pending",
			CreatedAgo: 18 * oneHour, ExpiresIn: 6*oneDay + 6*oneHour,
		},
	}
}

// seedGroupApplications 返回入群申请：
//   - application_seed_1：lumos → group_seed_3 待审（覆盖审核流程页）
//   - application_seed_2：lumos → group_seed_4 已被拒（为 group_application_rejected 通知提供实体）
func seedGroupApplications() []groupApplicationSpec {
	return []groupApplicationSpec{
		{ApplicationID: "application_seed_1", GroupID: "group_seed_3", ApplicantID: "user_seed_lumos",
			Message: "想跟大家一起学弹唱，目前会一点吉他。", Status: "pending", CreatedAgo: 2 * oneDay},
		{ApplicationID: "application_seed_2", GroupID: "group_seed_4", ApplicantID: "user_seed_lumos",
			Message: "想加入夜读角。", Status: "rejected", CreatedAgo: 10 * oneDay,
			HandledAgo: 9 * oneDay, HandledBy: "user_seed_chen"},
	}
}

// 以下为群相关行规格（放在本文件便于集中维护；构建层直接消费）。

// groupBanSpec 对应 group_bans 行（迁移 027）。
type groupBanSpec struct {
	GroupID    string
	UserID     string
	BannedByID string
	Reason     string
	BannedAgo  time.Duration
}

// groupInviteSpec 对应 group_invites 行（迁移 023）。
type groupInviteSpec struct {
	InviteID   string
	GroupID    string
	InviterID  string
	InviteeID  string
	Message    string
	Status     string // pending / accepted / rejected / cancelled
	CreatedAgo time.Duration
	ExpiresIn  time.Duration
}

// groupApplicationSpec 对应 group_applications 行（迁移 023）。
type groupApplicationSpec struct {
	ApplicationID string
	GroupID       string
	ApplicantID   string
	Message       string
	Status        string // pending / approved / rejected
	CreatedAgo    time.Duration
	HandledAgo    time.Duration
	HandledBy     string
}
