package seed

import (
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

// data_groups_chats_test.go — 群组/私聊一致性：准入形态、成员角色、S2 口径、
// 群消息载荷与未读、私聊 chat_key 与未读。

func TestGroupsFormsAndMembers(t *testing.T) {
	data := buildSeedData()
	groups := map[string]groupSpec{}
	for _, group := range data.groups {
		groups[group.GroupID] = group
	}
	if len(groups) != 4 {
		t.Fatalf("群组数 = %d，期望 4", len(groups))
	}
	roleByKey := map[string]string{}
	for _, entry := range expandGroupMembers(testBaseTime, data.groups) {
		roleByKey[entry.GroupID+"|"+entry.UserID] = entry.Role
	}
	if roleByKey["group_seed_1|user_seed_marco"] != "owner" {
		t.Errorf("group_seed_1 owner 应为 marco，实际 %q", roleByKey["group_seed_1|user_seed_marco"])
	}
	if roleByKey["group_seed_1|user_seed_sophia"] != "admin" {
		t.Errorf("group_seed_1 admin 应为 sophia，实际 %q", roleByKey["group_seed_1|user_seed_sophia"])
	}
	if roleByKey["group_seed_1|user_seed_lumos"] != "member" {
		t.Errorf("group_seed_1 中 lumos 应为 member")
	}
	for _, memberID := range []string{"user_seed_mika", "user_seed_chen", "user_seed_lin", "user_seed_xiaoman"} {
		if roleByKey["group_seed_1|"+memberID] != "member" {
			t.Errorf("group_seed_1 成员缺少/角色错误: %s", memberID)
		}
	}
	if groups["group_seed_2"].JoinMethod != "password" || groups["group_seed_2"].Password == "" {
		t.Errorf("group_seed_2 应为 password 群且设置口令")
	}
	if _, joined := roleByKey["group_seed_2|user_seed_lumos"]; joined {
		t.Errorf("lumos 不应加入 group_seed_2（覆盖密码加入流程）")
	}
	if groups["group_seed_3"].JoinMethod != "invite_only" || !groups["group_seed_3"].RequiresApproval {
		t.Errorf("group_seed_3 应为 invite_only + requires_approval")
	}
	if groups["group_seed_4"].JoinMethod != "direct" || !groups["group_seed_4"].RequiresApproval {
		t.Errorf("group_seed_4 应为 direct + requires_approval")
	}
	// 待审申请 / 待处理邀请 / 群公告 / 群封禁覆盖。
	pendingApplications := 0
	for _, application := range data.groupApplications {
		if application.ApplicantID == "user_seed_lumos" && application.Status == "pending" {
			pendingApplications++
		}
	}
	if pendingApplications != 1 {
		t.Errorf("lumos 的待审申请数 = %d，期望 1", pendingApplications)
	}
	pendingInvites := 0
	for _, invite := range data.groupInvites {
		if invite.InviteeID == "user_seed_lumos" && invite.Status == "pending" {
			pendingInvites++
		}
	}
	if pendingInvites != 1 {
		t.Errorf("lumos 的待处理邀请数 = %d，期望 1", pendingInvites)
	}
	if len(data.groupBans) != 1 {
		t.Errorf("群封禁记录数 = %d，期望 1", len(data.groupBans))
	}
	announcements := 0
	for _, group := range data.groups {
		announcements += len(group.Announcements)
	}
	if announcements < 3 {
		t.Errorf("群公告数 = %d，期望 >= 3（group1 两条 + group4 一条）", announcements)
	}
}

// TestGroupsS2 群组 s2_cell_token（14 级）/ s2_cell_id（叶级）口径与 groups 模块一致。
func TestGroupsS2(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "groups")
	columns := columnsOfTable(t, plan, "groups")
	latIndex := columnIndex(t, columns, "latitude")
	lngIndex := columnIndex(t, columns, "longitude")
	tokenIndex := columnIndex(t, columns, "s2_cell_token")
	s2Index := columnIndex(t, columns, "s2_cell_id")
	if len(rows) != 4 {
		t.Fatalf("groups 行数 = %d", len(rows))
	}
	for _, row := range rows {
		point := geo.Point{Latitude: mustFloat(t, row[latIndex]), Longitude: mustFloat(t, row[lngIndex])}
		wantCell := geo.LeafCellID(point)
		cell, ok := row[s2Index].(int64)
		if !ok {
			t.Fatalf("群 s2_cell_id 参数类型 = %T，期望 int64", row[s2Index])
		}
		if wantCell == 0 || cell != wantCell {
			t.Errorf("群 %s s2_cell_id = %d，期望 %d", mustString(t, row[0]), cell, wantCell)
		}
		token := mustString(t, row[tokenIndex])
		if level := geo.CellTokenLevel(token); level != groupS2TokenLevel {
			t.Errorf("群 s2_cell_token 层级 = %d，期望 %d", level, groupS2TokenLevel)
		}
		if want := geo.CellTokenAtLevel(point, groupS2TokenLevel); token != want {
			t.Errorf("群 s2_cell_token = %q，期望 %q", token, want)
		}
	}
}

// TestGroupMessagesPayload 群消息覆盖 system/image/share/@提及/撤回/已编辑。
func TestGroupMessagesPayload(t *testing.T) {
	data := buildSeedData()
	if len(data.groupMessages) != 10 {
		t.Fatalf("群消息数 = %d，期望 10", len(data.groupMessages))
	}
	recalled, edited, mentionsLumos := 0, 0, 0
	types := map[string]int{}
	for _, message := range data.groupMessages {
		types[message.Type]++
		if message.Recalled {
			recalled++
			if message.RecalledBy == "" || message.ContentPrev == "" {
				t.Errorf("撤回消息 %s 缺少 recalled_by/content_prev", message.MessageID)
			}
		}
		if message.EditedAgo > 0 {
			edited++
			if message.ContentPrev == "" {
				t.Errorf("已编辑消息 %s 缺少 content_prev", message.MessageID)
			}
		}
		for _, mentioned := range message.Mentions {
			if mentioned == "user_seed_lumos" {
				mentionsLumos++
			}
		}
	}
	if recalled != 1 {
		t.Errorf("撤回消息数 = %d，期望 1", recalled)
	}
	if edited != 1 {
		t.Errorf("已编辑消息数 = %d，期望 1", edited)
	}
	if mentionsLumos != 1 {
		t.Errorf("含 @lumos 的群消息数 = %d，期望 1", mentionsLumos)
	}
	for _, wantType := range []string{"text", "image", "share", "system"} {
		if types[wantType] == 0 {
			t.Errorf("群消息缺少类型 %s", wantType)
		}
	}
}

// TestGroupUnreadCount lumos 在 group_seed_1 未读 3（与 queryListMyGroupsBase 口径一致）。
func TestGroupUnreadCount(t *testing.T) {
	data := buildSeedData()
	sentAt := map[string]time.Time{}
	for _, message := range data.groupMessages {
		sentAt[message.MessageID] = testBaseTime.Add(-message.SentAgo)
	}
	var cursor time.Time
	var groupID string
	for _, state := range data.groupReadStates {
		if state.UserID == "user_seed_lumos" {
			cursor = sentAt[state.LastReadMessageID]
			groupID = state.GroupID
		}
	}
	if cursor.IsZero() {
		t.Fatal("缺少 lumos 的群已读游标")
	}
	unread := 0
	for _, message := range data.groupMessages {
		if message.GroupID != groupID || message.Recalled {
			continue
		}
		if testBaseTime.Add(-message.SentAgo).After(cursor) {
			unread++
		}
	}
	if unread != 3 {
		t.Errorf("lumos 在 %s 的未读消息数 = %d，期望 3", groupID, unread)
	}
}

// TestPrivateChatUnread 私聊未读：sophia 2 / marco 0 / mika 1。
func TestPrivateChatUnread(t *testing.T) {
	data := buildSeedData()
	if len(data.chats) != 3 {
		t.Fatalf("私聊会话数 = %d，期望 3", len(data.chats))
	}
	wantUnread := map[string]int{"user_seed_sophia": 2, "user_seed_marco": 0, "user_seed_mika": 1}
	wantMessageCounts := map[string]int{"user_seed_sophia": 12, "user_seed_marco": 4, "user_seed_mika": 4}
	for _, chat := range data.chats {
		target := chat.TargetUserID
		if got := len(chat.Messages); got != wantMessageCounts[target] {
			t.Errorf("与 %s 的私聊消息数 = %d，期望 %d", target, got, wantMessageCounts[target])
		}
		sentAt := map[string]time.Time{}
		for _, message := range chat.Messages {
			sentAt[message.MessageID] = testBaseTime.Add(-message.SentAgo)
		}
		var cursor time.Time
		for _, state := range chat.ReadStates {
			if state.UserID == "user_seed_lumos" {
				cursor = sentAt[state.LastReadMessageID]
			}
		}
		if cursor.IsZero() {
			t.Errorf("与 %s 的私聊缺少 lumos 已读游标", target)
			continue
		}
		unread := 0
		for _, message := range chat.Messages {
			if message.SenderID == "user_seed_lumos" || message.Recalled {
				continue
			}
			if testBaseTime.Add(-message.SentAgo).After(cursor) {
				unread++
			}
		}
		if unread != wantUnread[target] {
			t.Errorf("与 %s 的未读数 = %d，期望 %d", target, unread, wantUnread[target])
		}
	}
}

// TestPrivateChatMessageFeatures sophia 会话覆盖 share/图片/回复/撤回/已编辑 + 置顶。
func TestPrivateChatMessageFeatures(t *testing.T) {
	data := buildSeedData()
	for _, chat := range data.chats {
		if chat.TargetUserID != "user_seed_sophia" {
			continue
		}
		pinned := false
		for _, state := range chat.States {
			if state.UserID == "user_seed_lumos" && state.Pinned {
				pinned = true
			}
		}
		if !pinned {
			t.Errorf("与 sophia 的会话应为 lumos 置顶")
		}
		features := map[string]bool{}
		for _, message := range chat.Messages {
			switch message.Type {
			case "share":
				features["share"] = true
			case "image":
				features["image"] = true
			}
			if message.ReplyTo != "" {
				features["reply"] = true
			}
			if message.Recalled {
				features["recalled"] = true
			}
			if message.EditedAgo > 0 {
				features["edited"] = true
			}
		}
		for _, want := range []string{"share", "image", "reply", "recalled", "edited"} {
			if !features[want] {
				t.Errorf("与 sophia 的会话缺少消息特征 %s", want)
			}
		}
		return
	}
	t.Fatal("缺少与 sophia 的私聊会话")
}

// TestPrivateChatKeys private_chats.chat_key 与 privatechats 模块规则一致。
func TestPrivateChatKeys(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "private_chats")
	columns := columnsOfTable(t, plan, "private_chats")
	keyIndex := columnIndex(t, columns, "chat_key")
	aIndex := columnIndex(t, columns, "user_a_id")
	bIndex := columnIndex(t, columns, "user_b_id")
	if len(rows) != 3 {
		t.Fatalf("private_chats 行数 = %d，期望 3", len(rows))
	}
	wantKeys := map[string]bool{
		"user_seed_lumos:user_seed_sophia": true,
		"user_seed_lumos:user_seed_marco":  true,
		"user_seed_lumos:user_seed_mika":   true,
	}
	for _, row := range rows {
		key := mustString(t, row[keyIndex])
		if !wantKeys[key] {
			t.Errorf("未知 chat_key: %q", key)
		}
		userA := mustString(t, row[aIndex])
		userB := mustString(t, row[bIndex])
		if userA >= userB {
			t.Errorf("private_chats 的 user_a_id 应小于 user_b_id: %q vs %q", userA, userB)
		}
		if chatKey(userA, userB) != key {
			t.Errorf("chat_key %q 与 (user_a_id,user_b_id) 规则不符", key)
		}
	}
}
