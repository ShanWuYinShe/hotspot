package seed

import "time"

// specs_moments.go — 瞬间规格（seed-spec §3.2）。
//
// 24 条瞬间，作者轮换与规格一致：lumos×6（01-05 + 24 私密）、sophia×4（06-09）、
// marco×4（10-13）、mika×3（14-16）、chen×2（17-18）、lin×2（19-20）、
// xiaoman×2（21-22）、admin×1（23 软删）。
//
// 图片桶（规格 §3.2）：0 图 10 条、1 图 8 条、2 图 4 条、3 图 2 条，共 22 个
// 图片位；素材由 mediaplan 在 12 张 moment_*.jpg 间轮换（同一素材可被多个
// 上传会话复用，file_id 因 upload_id 不同而不同）。
//
// 位置均距 lumos 位置 100–3000m（data_test 用 geo.DistanceMeters 断言），
// 覆盖云龙湖/回龙窝/户部山/彭城广场/汉文化景区/植物园/老街等地。

// locationSpec 是瞬间/群组位置池条目。
type locationSpec struct {
	Name      string
	Latitude  float64
	Longitude float64
}

// seedLocations 返回与文案池一一对应的位置池（按 index 循环使用）。
func seedLocations() []locationSpec {
	return []locationSpec{
		{Name: "云龙湖东岸", Latitude: 34.2615, Longitude: 117.1888},
		{Name: "回龙窝", Latitude: 34.2578, Longitude: 117.1817},
		{Name: "户部山", Latitude: 34.2567, Longitude: 117.1843},
		{Name: "彭城广场", Latitude: 34.2593, Longitude: 117.1790},
		{Name: "汉文化景区", Latitude: 34.2517, Longitude: 117.1902},
		{Name: "植物园", Latitude: 34.2642, Longitude: 117.1912},
		{Name: "老街", Latitude: 34.2555, Longitude: 117.1872},
		{Name: "云龙山北门", Latitude: 34.2680, Longitude: 117.1870},
		{Name: "云龙湖大堤", Latitude: 34.2570, Longitude: 117.1930},
		{Name: "市中心商圈", Latitude: 34.2635, Longitude: 117.1780},
		{Name: "云龙湖西岸", Latitude: 34.2650, Longitude: 117.1790},
		{Name: "彭城广场南", Latitude: 34.2560, Longitude: 117.1810},
	}
}

// momentDescriptionPool 是 12 条中文生活向文案（原 MSW mock data.ts 文案池的润色版）。
func momentDescriptionPool() []string {
	return []string{
		"云龙湖的落日把水面染成金色，风一吹全是碎光。",
		"回龙窝的巷子里藏着一家很小的咖啡店，安静得只剩下磨豆机的声音。",
		"户部山的台阶被雨水洗得很干净，适合慢慢走。",
		"彭城广场的夜风正好，带了相机却没拍几张，光站着就很舒服。",
		"汉文化景区的午后，绿荫里有人在弹古琴。",
		"植物园的温室今天开了新的花，凑近闻有淡淡的香。",
		"老街的清晨属于早餐摊和遛狗的人。",
		"云龙山北门上去二十分钟，能看到整片湖。",
		"把上周的胶卷洗出来了，颗粒感刚刚好。",
		"市中心的梧桐开始泛黄，秋天真的来了。",
		"今天的云很像被谁随手揉皱的纸。",
		"雨停之后空气很清，适合把所有窗户都打开。",
	}
}

// reactions 是常用表情（必须在 appconfig 默认白名单内：👍 ❤️ 😂 😮 😢 🙏；
// 默认配置见 platform/appconfig/source.go defaultConfig.Limits.ReactionEmojis）。
const (
	emojiThumb = "👍"
	emojiHeart = "❤️"
	emojiLaugh = "😂"
	emojiWow   = "😮"
	emojiPray  = "🙏"
)

// seedMoments 返回 24 条瞬间规格。
func seedMoments() []momentSpec {
	locs := seedLocations()
	descs := momentDescriptionPool()
	loc := func(i int) (string, float64, float64) {
		item := locs[i%len(locs)]
		return item.Name, item.Latitude, item.Longitude
	}
	desc := func(i int) string { return descs[i%len(descs)] }

	// 统一按「一条瞬间 = 一个 momentSpec」书写：createdAgo 越大越旧，
	// moment_seed_01 最新、moment_seed_24 最旧。
	specs := make([]momentSpec, 0, 24)
	add := func(index int, author string, description string, imageCount int, tags []string, mentions []string, reactions []reactionSpec, comments []commentSpec) {
		name, lat, lng := loc(index)
		specs = append(specs, momentSpec{
			MomentID:     momentIDFor(index),
			AuthorID:     author,
			Description:  description,
			LocationName: name,
			Latitude:     lat,
			Longitude:    lng,
			CreatedAgo:   momentCreatedAgo(index),
			PrivacyLevel: "public",
			ImageCount:   imageCount,
			Tags:         tags,
			Mentions:     mentions,
			Reactions:    reactions,
			Comments:     comments,
		})
	}

	// 01 lumos —— 云龙湖落日（互动最全：图片 + 反应 + 两级评论 + 收藏 + 分享目标）
	add(0, "user_seed_lumos", desc(0), 1, []string{"云龙湖", "摄影"}, nil,
		[]reactionSpec{{"user_seed_sophia", emojiThumb}, {"user_seed_marco", emojiHeart}, {"user_seed_mika", emojiLaugh}},
		[]commentSpec{
			{CommentID: "comment_seed_01_01", UserID: "user_seed_sophia", Content: "光影真的好，求机位！", Likes: []string{"user_seed_lumos", "user_seed_marco"}, CreatedAgo: momentCreatedAgo(0) - 10*oneMinute},
			{CommentID: "comment_seed_01_02", UserID: "user_seed_marco", ParentID: "comment_seed_01_01", Content: "云龙湖北岸那个木栈道。", CreatedAgo: momentCreatedAgo(0) - 20*oneMinute},
			{CommentID: "comment_seed_01_03", UserID: "user_seed_mika", Content: "周末也想去。", Likes: []string{"user_seed_sophia"}, CreatedAgo: momentCreatedAgo(0) - 30*oneMinute},
		})
	// 02 lumos —— 回龙窝咖啡
	add(1, "user_seed_lumos", desc(1), 0, []string{"咖啡", "回龙窝"}, nil,
		[]reactionSpec{{"user_seed_lin", emojiThumb}}, nil)
	// 03 lumos —— 户部山 @青砚（覆盖 moment_mentions）
	add(2, "user_seed_lumos", "在户部山遇到 @青砚，一起把这条巷子拍完了。", 1, []string{"户部山", "摄影"}, []string{"user_seed_chen"},
		[]reactionSpec{{"user_seed_xiaoman", emojiHeart}},
		[]commentSpec{{CommentID: "comment_seed_03_01", UserID: "user_seed_chen", Content: "下次带长焦。", Likes: []string{"user_seed_lumos"}, CreatedAgo: momentCreatedAgo(2) - 15*oneMinute}})
	// 04 lumos —— 彭城广场夜景（无图，覆盖纯文字瞬间）
	add(3, "user_seed_lumos", desc(3), 0, []string{"彭城广场", "夜景"}, nil,
		[]reactionSpec{{"user_seed_chen", emojiThumb}}, nil)
	// 05 lumos —— 汉文化景区（lumos 收藏）
	add(4, "user_seed_lumos", desc(4), 1, []string{"汉文化景区", "散步"}, nil,
		[]reactionSpec{{"user_seed_sophia", emojiHeart}, {"user_seed_admin", emojiThumb}},
		[]commentSpec{
			{CommentID: "comment_seed_05_01", UserID: "user_seed_sophia", Content: "古琴声是不是每天都有？", Likes: []string{"user_seed_mika"}, CreatedAgo: momentCreatedAgo(4) - 20*oneMinute},
			{CommentID: "comment_seed_05_02", UserID: "user_seed_lumos", ParentID: "comment_seed_05_01", Content: "周末下午基本都有。", CreatedAgo: momentCreatedAgo(4) - 10*oneMinute},
		})
	// 06 sophia —— 植物园
	add(5, "user_seed_sophia", desc(5), 1, []string{"植物园", "花"}, nil,
		[]reactionSpec{{"user_seed_marco", emojiThumb}}, nil)
	// 07 sophia —— 老街（被举报目标，保留图片便于举报详情展示）
	add(6, "user_seed_sophia", desc(6), 1, []string{"老街", "日常"}, nil,
		[]reactionSpec{{"user_seed_lumos", emojiWow}},
		[]commentSpec{{CommentID: "comment_seed_07_01", UserID: "user_seed_lumos", Content: "这角度可以。", CreatedAgo: momentCreatedAgo(7) - 20*oneMinute}})
	// 08 sophia —— 云龙山北门（纯文字）
	add(7, "user_seed_sophia", desc(7), 0, []string{"云龙山", "徒步"}, nil, nil, nil)
	// 09 sophia —— 云龙湖大堤（2 图 + 两级评论）
	add(8, "user_seed_sophia", desc(8), 2, []string{"云龙湖", "胶片"}, nil,
		[]reactionSpec{{"user_seed_marco", emojiHeart}, {"user_seed_mika", emojiThumb}, {"user_seed_xiaoman", emojiPray}},
		[]commentSpec{
			{CommentID: "comment_seed_09_01", UserID: "user_seed_marco", Content: "颗粒感绝了，什么卷？", Likes: []string{"user_seed_sophia"}, CreatedAgo: momentCreatedAgo(9) - 20*oneMinute},
			{CommentID: "comment_seed_09_02", UserID: "user_seed_sophia", ParentID: "comment_seed_09_01", Content: "Portra 400。", CreatedAgo: momentCreatedAgo(9) - 15*oneMinute},
			{CommentID: "comment_seed_09_03", UserID: "user_seed_mika", Content: "求链接。", CreatedAgo: momentCreatedAgo(9) - 5*oneMinute},
		})
	// 10 marco —— 市中心梧桐
	add(9, "user_seed_marco", desc(9), 1, []string{"市中心", "秋天"}, nil,
		[]reactionSpec{{"user_seed_mika", emojiThumb}}, nil)
	// 11 marco —— 云龙湖西岸（无文字位置文案，纯图 + 反应）
	add(10, "user_seed_marco", desc(10), 1, []string{"云龙湖", "胶片"}, nil,
		[]reactionSpec{{"user_seed_chen", emojiHeart}, {"user_seed_lin", emojiThumb}}, nil)
	// 12 marco —— 彭城广场南（纯文字）
	add(11, "user_seed_marco", desc(11), 0, []string{"日常"}, nil, nil, nil)
	// 13 marco —— 云龙湖东岸 @林一（3 图，覆盖三图网格 + 群组分享卡片同源）
	add(12, "user_seed_marco", "把 @林一 的吉他声录进了黄昏里。", 3, []string{"音乐", "黄昏"}, []string{"user_seed_lin"},
		[]reactionSpec{{"user_seed_sophia", emojiThumb}, {"user_seed_mika", emojiHeart}, {"user_seed_lin", emojiLaugh}, {"user_seed_admin", emojiPray}},
		[]commentSpec{{CommentID: "comment_seed_13_01", UserID: "user_seed_lin", Content: "下次一起录一段。", Likes: []string{"user_seed_marco"}, CreatedAgo: momentCreatedAgo(13) - 20*oneMinute}})
	// 14 mika —— 回龙窝甜品（附近新瞬间通知的 actor）
	add(13, "user_seed_mika", desc(1), 2, []string{"回龙窝", "甜品"}, nil,
		[]reactionSpec{{"user_seed_lumos", emojiThumb}, {"user_seed_marco", emojiHeart}},
		[]commentSpec{{CommentID: "comment_seed_14_01", UserID: "user_seed_lumos", Content: "甜品店地址发我。", CreatedAgo: momentCreatedAgo(14) - 20*oneMinute}})
	// 15 mika —— 户部山
	add(14, "user_seed_mika", desc(2), 2, []string{"户部山", "探店"}, nil,
		[]reactionSpec{{"user_seed_sophia", emojiThumb}}, nil)
	// 16 mika —— 彭城广场（纯文字）
	add(15, "user_seed_mika", desc(3), 0, []string{"彭城广场"}, nil, nil, nil)
	// 17 chen —— 汉文化景区夜跑
	add(16, "user_seed_chen", desc(4), 1, []string{"夜跑", "汉文化景区"}, nil,
		[]reactionSpec{{"user_seed_mika", emojiThumb}}, nil)
	// 18 chen —— 植物园
	add(17, "user_seed_chen", desc(5), 2, []string{"夜跑", "植物园"}, nil,
		[]reactionSpec{{"user_seed_sophia", emojiHeart}}, nil)
	// 19 lin —— 老街（3 图 + 评论）
	add(18, "user_seed_lin", desc(6), 3, []string{"音乐", "老街"}, nil,
		[]reactionSpec{{"user_seed_marco", emojiThumb}},
		[]commentSpec{{CommentID: "comment_seed_19_01", UserID: "user_seed_xiaoman", Content: "下次唱这首。", Likes: []string{"user_seed_lin"}, CreatedAgo: momentCreatedAgo(19) - 20*oneMinute}})
	// 20 lin —— 云龙山北门（纯文字）
	add(19, "user_seed_lin", desc(7), 0, []string{"吉他"}, nil, nil, nil)
	// 21 xiaoman —— 云龙湖大堤
	add(20, "user_seed_xiaoman", desc(8), 0, []string{"汉服"}, nil,
		[]reactionSpec{{"user_seed_lumos", emojiThumb}}, nil)
	// 22 xiaoman —— 市中心手作
	add(21, "user_seed_xiaoman", desc(9), 0, []string{"手作"}, nil,
		[]reactionSpec{{"user_seed_lumos", emojiHeart}, {"user_seed_admin", emojiThumb}},
		[]commentSpec{{CommentID: "comment_seed_22_01", UserID: "user_seed_lumos", Content: "手作好精致。", Likes: []string{"user_seed_xiaoman"}, CreatedAgo: momentCreatedAgo(22) - 20*oneMinute}})
	// 23 admin —— 软删目标（迁移 037：is_deleted=true 且 description 清空）。
	// 该条带两个已被平台屏蔽的标签：管理端「标签管理」页需要能展示
	// admin_blocked_tags 中的标签（querySearchTags 从 moment_tags 聚合，屏蔽
	// 但从未被使用的标签不会出现在列表里），软删瞬间不会出现在任何用户视图，
	// 因此对前端展示无副作用。
	add(22, "user_seed_admin", "", 0, []string{"广告", "引战"}, nil, nil, nil)
	specs[len(specs)-1].IsDeleted = true
	// 24 lumos —— 私密瞬间（仅作者可见，覆盖 privacy_level='private'）
	add(23, "user_seed_lumos", "只给自己看的一张：雨后的窗。", 0, []string{"日常"}, nil, nil, nil)
	specs[len(specs)-1].PrivacyLevel = "private"

	return specs
}

// momentIDFor 返回第 index（0-based）条瞬间的固定 ID。
func momentIDFor(index int) string {
	return momentIDPrefix + twoDigits(index+1)
}

// momentCreatedAgo 返回第 index 条瞬间相对基准时刻的发布偏移（01 最新）。
// 间隔 2–3 小时，整体铺满最近 3 天，Feed 按时间倒序时有稳定层次。
func momentCreatedAgo(index int) time.Duration {
	return momentCreatedOffsets[index]
}

// momentCreatedOffsets 是 24 条瞬间的发布偏移（单位：小时）。
var momentCreatedOffsets = [24]time.Duration{
	2, 5, 7, 9, 12, 15, 18, 21, 24, 27, 30, 33,
	36, 39, 42, 45, 48, 51, 54, 57, 60, 63, 66, 69,
}

// oneMinute 供评论/回复的细分时间偏移使用。
const oneMinute = time.Minute
