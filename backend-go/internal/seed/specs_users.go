package seed

import "time"

// specs_users.go — 用户/关系/账号规格（seed-spec §3.1 / §2）。
//
// 用户定位集中在徐州云龙湖—回龙窝—户部山一带（与原 MSW mock 的徐州语境
// 一致），彼此距离数百米到数公里，保证「附近的人 / 地图」页面有真实候选。
// 坐标均为 WGS-84；s2_cell_id 由构建层用 platform/geo 计算。

// seedUsers 返回 10 个固定 ID 用户（2 个 QA 账号 + 7 个普通用户 + 1 个封禁用户）。
func seedUsers() []userSpec {
	lumosSettings := defaultSettings()
	// 附近新瞬间通知：开启后才能覆盖 nearby_new_moment 通知链路
	// （user_settings 迁移 033；notify 模块发通知前会校验该开关）。
	lumosSettings.NearbyMomentNotify = true

	privacySettings := defaultSettings()
	privacySettings.PrivacyLevel = "private"
	privacySettings.LocationVisible = false
	privacySettings.PresenceVisible = false
	privacySettings.Searchable = false

	return []userSpec{
		{
			UserID: "user_seed_lumos", Nickname: "鹿鸣", AvatarAsset: "avatar_lumos.jpg",
			Bio: "记录城市光影", LocationName: seedAnchorLocationName, Latitude: seedAnchorLatitude, Longitude: seedAnchorLongitude,
			Presence: "online", Status: "normal",
			Tags:       []string{"摄影", "旅行", "咖啡"},
			CreatedAgo: 90 * oneDay, LastOnlineAgo: 2 * time.Minute,
			Settings: lumosSettings,
		},
		{
			UserID: "user_seed_sophia", Nickname: "阿禾", AvatarAsset: "avatar_sophia.jpg",
			Bio: "植物园常客，拍花拍云", LocationName: "植物园", Latitude: 34.2620, Longitude: 117.1900,
			Presence: "online", Status: "normal",
			Tags:       []string{"植物", "散步"},
			CreatedAgo: 82 * oneDay, LastOnlineAgo: 5 * time.Minute,
			Settings: defaultSettings(),
		},
		{
			UserID: "user_seed_marco", Nickname: "野原", AvatarAsset: "avatar_marco.jpg",
			Bio: "胶片与咖啡因依赖", LocationName: "市中心", Latitude: 34.2580, Longitude: 117.1800,
			Presence: "offline", Status: "normal",
			Tags:       []string{"摄影", "咖啡"},
			CreatedAgo: 75 * oneDay, LastOnlineAgo: 3 * time.Hour,
			Settings: defaultSettings(),
		},
		{
			UserID: "user_seed_mika", Nickname: "Mika", AvatarAsset: "avatar_mika.jpg",
			Bio: "甜品探店进度 3/100", LocationName: "老街", Latitude: 34.2550, Longitude: 117.1870,
			Presence: "online", Status: "normal",
			Tags:       []string{"美食", "甜品"},
			CreatedAgo: 66 * oneDay, LastOnlineAgo: time.Minute,
			Settings: defaultSettings(),
		},
		{
			UserID: "user_seed_chen", Nickname: "青砚", AvatarAsset: "avatar_chen.jpg",
			Bio: "夜跑、夜读、夜里的湖", LocationName: "云龙湖", Latitude: 34.2610, Longitude: 117.1860,
			Presence: "offline", Status: "normal",
			Tags:       []string{"跑步", "夜读"},
			CreatedAgo: 58 * oneDay, LastOnlineAgo: 8 * time.Hour,
			Settings: defaultSettings(),
		},
		{
			UserID: "user_seed_lin", Nickname: "林一", AvatarAsset: "avatar_lin.jpg",
			Bio: "独立音乐人，常在户部山弹唱", LocationName: "彭城广场", Latitude: 34.2590, Longitude: 117.1830,
			Presence: "online", Status: "normal",
			Tags:       []string{"音乐", "吉他"},
			CreatedAgo: 52 * oneDay, LastOnlineAgo: 3 * time.Minute,
			Settings: defaultSettings(),
		},
		{
			UserID: "user_seed_xiaoman", Nickname: "小满", AvatarAsset: "avatar_xiaoman.jpg",
			Bio: "汉服与手作，慢一点没关系", LocationName: "汉文化景区", Latitude: 34.2500, Longitude: 117.1900,
			Presence: "online", Status: "normal",
			Tags:       []string{"汉服", "手作"},
			CreatedAgo: 45 * oneDay, LastOnlineAgo: 6 * time.Minute,
			Settings: defaultSettings(),
		},
		{
			UserID: "user_seed_admin", Nickname: "巡检员", AvatarAsset: "avatar_admin.jpg",
			Bio: "平台巡检，欢迎反馈问题", LocationName: "云龙湖", Latitude: 34.2600, Longitude: 117.1855,
			Presence: "online", Status: "normal",
			Tags:       []string{"官方", "巡检"},
			CreatedAgo: 100 * oneDay, LastOnlineAgo: time.Minute,
			Settings: defaultSettings(),
		},
		{
			UserID: "user_seed_privacy", Nickname: "隐者", AvatarAsset: "avatar_privacy.jpg",
			Bio: "安静的人", LocationName: "附近", Latitude: 34.2570, Longitude: 117.1820,
			Presence: "offline", Status: "normal",
			Tags:       nil,
			CreatedAgo: 30 * oneDay, LastOnlineAgo: 26 * time.Hour,
			Settings: privacySettings,
		},
		{
			UserID: "user_seed_banned", Nickname: "已封禁用户", AvatarAsset: "",
			Bio: "该账号用于管理端封禁态测试", LocationName: "附近", Latitude: 34.2560, Longitude: 117.1810,
			Presence: "offline", Status: "banned",
			BannedReason: "测试封禁（seed 数据）", BannedDays: 7, BannedAgo: 3 * oneDay,
			Tags:       nil,
			CreatedAgo: 20 * oneDay, LastOnlineAgo: 4 * oneDay,
			Settings: defaultSettings(),
		},
	}
}

// seedRelations 返回关注/互关/拉黑关系（seed-spec §3.1「关系」列）。
//
// 数据事实（基于 user_relations 的双向边语义）：
//   - 好友（互关）：sophia / marco / mika / xiaoman / admin
//   - lumos 单向关注：chen
//   - 关注 lumos（粉丝）：lin
//   - lumos 拉黑 privacy（覆盖黑名单页；被拉黑方瞬间从 feed 消失）
func seedRelations() []relationSpec {
	return []relationSpec{
		{From: "user_seed_lumos", To: "user_seed_sophia", Following: true, FollowedAgo: 80 * oneDay},
		{From: "user_seed_sophia", To: "user_seed_lumos", Following: true, FollowedAgo: 79 * oneDay},
		{From: "user_seed_lumos", To: "user_seed_marco", Following: true, FollowedAgo: 70 * oneDay},
		{From: "user_seed_marco", To: "user_seed_lumos", Following: true, FollowedAgo: 69 * oneDay},
		{From: "user_seed_lumos", To: "user_seed_mika", Following: true, FollowedAgo: 60 * oneDay},
		{From: "user_seed_mika", To: "user_seed_lumos", Following: true, FollowedAgo: 58 * oneDay},
		{From: "user_seed_lumos", To: "user_seed_xiaoman", Following: true, FollowedAgo: 50 * oneDay},
		{From: "user_seed_xiaoman", To: "user_seed_lumos", Following: true, FollowedAgo: 49 * oneDay},
		{From: "user_seed_lumos", To: "user_seed_chen", Following: true, FollowedAgo: 40 * oneDay},
		{From: "user_seed_lin", To: "user_seed_lumos", Following: true, FollowedAgo: 35 * oneDay},
		{From: "user_seed_lumos", To: "user_seed_admin", Following: true, FollowedAgo: 85 * oneDay},
		{From: "user_seed_admin", To: "user_seed_lumos", Following: true, FollowedAgo: 84 * oneDay},
		{From: "user_seed_lumos", To: "user_seed_privacy", Blocked: true, BlockedAgo: 20 * oneDay, BlockReason: "seed 数据：拉黑态覆盖"},
	}
}

// seedAccounts 返回登录账号规格（seed-spec §2）。
//
// 口令来源与优先级：进程 env（由 cmd 层注入 Options）> 规格缺省值。
// 构造 seedData 时不填充 Password，由 BuildPlan 用 Options 中的口令回填，
// 避免静态数据里出现口令字面量（除 dev 缺省值外）。
func seedAccounts() []accountSpec {
	accounts := []accountSpec{
		{LoginID: "qa@hotspot.local", UserID: "user_seed_lumos", CreatedAgo: 90 * oneDay},
		{LoginID: "admin@hotspot.local", UserID: "user_seed_admin", CreatedAgo: 100 * oneDay},
	}
	others := []struct {
		loginID string
		userID  string
		ago     time.Duration
	}{
		{"sophia@hotspot.local", "user_seed_sophia", 82 * oneDay},
		{"marco@hotspot.local", "user_seed_marco", 75 * oneDay},
		{"mika@hotspot.local", "user_seed_mika", 66 * oneDay},
		{"chen@hotspot.local", "user_seed_chen", 58 * oneDay},
		{"lin@hotspot.local", "user_seed_lin", 52 * oneDay},
		{"xiaoman@hotspot.local", "user_seed_xiaoman", 45 * oneDay},
		{"privacy@hotspot.local", "user_seed_privacy", 30 * oneDay},
		{"banned@hotspot.local", "user_seed_banned", 20 * oneDay},
	}
	for _, item := range others {
		accounts = append(accounts, accountSpec{LoginID: item.loginID, UserID: item.userID, CreatedAgo: item.ago})
	}
	return accounts
}

// legacyAccountPassword 是除两个 QA 账号以外的开发账号统一口令（seed-spec §3.1）。
const legacyAccountPassword = "Seed123456"

// defaultQAPassword / defaultAdminPassword 是 dev 缺省口令（仅本地开发；
// 生产部署不得沿用，见 backend-go/AGENTS.md）。
const (
	defaultQAPassword    = "Qa12345678"
	defaultAdminPassword = "Admin12345678"
)

// groupSeedPassword 是 group_seed_2（password 群）的入群口令。
const groupSeedPassword = "run123456"
