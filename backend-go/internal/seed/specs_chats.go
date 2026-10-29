package seed

// specs_chats.go — 私聊会话规格（seed-spec §3.4）。
//
// 3 个会话，视角用户均为 lumos（QA 主账号）：
//   - lumos ↔ sophia：12 条消息（text / image / share / 回复 / 撤回 / 已编辑各覆盖），
//     lumos 未读 2、会话置顶
//   - lumos ↔ marco：4 条 text，未读 0
//   - lumos ↔ mika：3 条 text + 1 条 image，未读 1
//
// 未读口径 = 「对方发送、sent_at 晚于本人已读游标、且未撤回」的消息数
// （privatemessages/repo_unread 查询），所以已读游标直接决定未读角标。

// seedChats 返回 3 个私聊会话规格。
func seedChats() []chatSpec {
	return []chatSpec{
		{
			UserID: "user_seed_lumos", TargetUserID: "user_seed_sophia",
			CreatedAgo: 49 * oneHour,
			Messages: []chatMessageSpec{
				{MessageID: "chat_seed_sophia_01", SenderID: "user_seed_sophia", Type: "text", Content: "在吗？周末去云龙湖拍落日吗？", SentAgo: 48 * oneHour},
				{MessageID: "chat_seed_sophia_02", SenderID: "user_seed_lumos", Type: "text", Content: "在，周日下午可以。", SentAgo: 47 * oneHour},
				{MessageID: "chat_seed_sophia_03", SenderID: "user_seed_sophia", Type: "text", Content: "那带胶片机？", SentAgo: 46 * oneHour},
				{MessageID: "chat_seed_sophia_04", SenderID: "user_seed_lumos", Type: "text", Content: "嗯，顺便试试新买的 50mm。", SentAgo: 45 * oneHour},
				{MessageID: "chat_seed_sophia_05", SenderID: "user_seed_sophia", Type: "text", Content: "",
					SentAgo: 40 * oneHour, Recalled: true, RecalledAgo: 40*oneHour - 30*oneMinute,
					RecalledBy: "user_seed_sophia", ContentPrev: "记得带三脚架。"},
				{MessageID: "chat_seed_sophia_06", SenderID: "user_seed_lumos", Type: "text", Content: "周五下午三点老地方见。",
					SentAgo: 24 * oneHour, EditedAgo: 23 * oneHour, ContentPrev: "下午三点见。"},
				{MessageID: "chat_seed_sophia_07", SenderID: "user_seed_sophia", Type: "image", MediaKey: "chat_img_sophia", SentAgo: 20 * oneHour},
				{MessageID: "chat_seed_sophia_08", SenderID: "user_seed_lumos", Type: "text", Content: "这张光线真好", ReplyTo: "chat_seed_sophia_07", SentAgo: 19 * oneHour},
				{MessageID: "chat_seed_sophia_09", SenderID: "user_seed_sophia", Type: "share", ShareCode: shareCodeMomentSeed, SentAgo: 18 * oneHour},
				{MessageID: "chat_seed_sophia_10", SenderID: "user_seed_sophia", Type: "text", Content: "这组照片可以参考下。", SentAgo: 17 * oneHour},
				{MessageID: "chat_seed_sophia_11", SenderID: "user_seed_sophia", Type: "text", Content: "那就说定啦。", SentAgo: 2 * oneHour},
				{MessageID: "chat_seed_sophia_12", SenderID: "user_seed_sophia", Type: "text", Content: "记得带备用电池。", SentAgo: oneHour},
			},
			States: []chatUserStateSpec{{UserID: "user_seed_lumos", Pinned: true}},
			ReadStates: []chatReadStateSpec{
				{UserID: "user_seed_lumos", LastReadMessageID: "chat_seed_sophia_10"},
				{UserID: "user_seed_sophia", LastReadMessageID: "chat_seed_sophia_12"},
			},
		},
		{
			UserID: "user_seed_lumos", TargetUserID: "user_seed_marco",
			CreatedAgo: 31 * oneHour,
			Messages: []chatMessageSpec{
				{MessageID: "chat_seed_marco_01", SenderID: "user_seed_marco", Type: "text", Content: "胶片扫街的照片洗出来了吗？", SentAgo: 30 * oneHour},
				{MessageID: "chat_seed_marco_02", SenderID: "user_seed_lumos", Type: "text", Content: "洗出来了，晚点发你。", SentAgo: 29 * oneHour},
				{MessageID: "chat_seed_marco_03", SenderID: "user_seed_marco", Type: "text", Content: "好，暗房还缺显影液。", SentAgo: 6 * oneHour},
				{MessageID: "chat_seed_marco_04", SenderID: "user_seed_lumos", Type: "text", Content: "我明天带一瓶过去。", SentAgo: 5 * oneHour},
			},
			ReadStates: []chatReadStateSpec{
				{UserID: "user_seed_lumos", LastReadMessageID: "chat_seed_marco_04"},
				{UserID: "user_seed_marco", LastReadMessageID: "chat_seed_marco_04"},
			},
		},
		{
			UserID: "user_seed_lumos", TargetUserID: "user_seed_mika",
			CreatedAgo: 27 * oneHour,
			Messages: []chatMessageSpec{
				{MessageID: "chat_seed_mika_01", SenderID: "user_seed_lumos", Type: "text", Content: "老街那家甜品店叫什么？", SentAgo: 26 * oneHour},
				{MessageID: "chat_seed_mika_02", SenderID: "user_seed_mika", Type: "text", Content: "叫拾光，在回龙窝拐角。", SentAgo: 25 * oneHour},
				{MessageID: "chat_seed_mika_03", SenderID: "user_seed_mika", Type: "image", MediaKey: "chat_img_mika", SentAgo: 24 * oneHour},
				{MessageID: "chat_seed_mika_04", SenderID: "user_seed_mika", Type: "text", Content: "就是这家，抹茶千层很推荐。", SentAgo: 2 * oneHour},
			},
			ReadStates: []chatReadStateSpec{
				{UserID: "user_seed_lumos", LastReadMessageID: "chat_seed_mika_03"},
				{UserID: "user_seed_mika", LastReadMessageID: "chat_seed_mika_04"},
			},
		},
	}
}
