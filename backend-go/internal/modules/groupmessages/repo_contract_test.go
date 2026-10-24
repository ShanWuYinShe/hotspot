package groupmessages

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 群消息仓库契约（核心读路径）：发送/历史分页、编辑/撤回窗口、已读位点与
// 已读计数、成员过滤与消息辅助查询必须双实现一致。
// PG 侧 seed：users + groups + group_members；契约测试用 BeginScope 事务隔离。
type gmContractState struct {
	ctx        context.Context
	groupID    string
	otherGroup string
	userA      string
	userB      string
	userC      string
	base       time.Time
}

func TestGroupMessagesRepositoryContract(t *testing.T) {
	userA := "it_gm_contract_a"
	userB := "it_gm_contract_b"
	userC := "it_gm_contract_c"
	groupID := "it_gm_contract_group"
	otherGroup := "it_gm_contract_group2"
	base := time.Now().UTC().Truncate(time.Millisecond)

	impls := []testdb.ContractImpl[Repository, gmContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, gmContractState, func()) {
				repo := NewInMemoryRepository()
				repo.SeedMember(groupID, userA, base.Add(-time.Hour))
				repo.SeedMember(groupID, userB, base.Add(-time.Hour))
				// C 是「消息发送之后才加入」的成员：已读计数不得覆盖更早的消息。
				repo.SeedMember(groupID, userC, base.Add(time.Hour))
				repo.SeedMember(otherGroup, userA, base.Add(-time.Hour))
				repo.SetGroupName(groupID, "Contract Group")
				repo.SetGroupName(otherGroup, "Contract Group 2")
				state := gmContractState{ctx: context.Background(), groupID: groupID, otherGroup: otherGroup, userA: userA, userB: userB, userC: userC, base: base}
				return repo, state, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, gmContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				for _, id := range []string{userA, userB, userC, "it_gm_admin"} {
					testdb.InsertUser(t, txFactory, id, "nick_"+id)
				}
				qdb, ok := txFactory.DB.(persistence.QueryableDB)
				if !ok {
					t.Fatal("factory.DB is not QueryableDB")
				}
				ctx := context.Background()
				for _, group := range []struct{ id, name string }{{groupID, "Contract Group"}, {otherGroup, "Contract Group 2"}} {
					if err := qdb.Exec(ctx, "INSERT INTO groups (group_id, name, location_name, latitude, longitude, owner_id) VALUES ($1,$2,'',$3,$4,$5)", group.id, group.name, 0.0, 0.0, userA); err != nil {
						t.Fatalf("insert group %s: %v", group.id, err)
					}
				}
				members := []struct {
					group  string
					user   string
					joined time.Time
				}{
					{groupID, userA, base.Add(-time.Hour)},
					{groupID, userB, base.Add(-time.Hour)},
					{groupID, userC, base.Add(time.Hour)},
					{otherGroup, userA, base.Add(-time.Hour)},
				}
				for _, member := range members {
					if err := qdb.Exec(ctx, "INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES ($1,$2,'member',$3)", member.group, member.user, member.joined); err != nil {
						t.Fatalf("insert member: %v", err)
					}
				}
				state := gmContractState{ctx: context.Background(), groupID: groupID, otherGroup: otherGroup, userA: userA, userB: userB, userC: userC, base: base}
				return NewPostgresRepository(txFactory), state, cleanup
			},
		},
	}

	sendText := func(t *testing.T, repo Repository, state gmContractState, groupID, sender, content string) MessageSentResult {
		t.Helper()
		sent, err := repo.Send(state.ctx, SendGroupMessageRequest{GroupID: groupID, SenderUserID: sender, Type: "text", Content: content})
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		if sent.MessageID == "" || sent.SentAt == "" {
			t.Fatalf("sent = %#v", sent)
		}
		return sent
	}

	historyIDs := func(items []GroupMessageItem) []string {
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, item.MessageID)
		}
		return out
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, gmContractState]{
		{
			Name: "history is newest-first and pageable",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				first := sendText(t, repo, state, state.groupID, state.userA, "m1")
				time.Sleep(3 * time.Millisecond)
				second := sendText(t, repo, state, state.groupID, state.userA, "m2")
				time.Sleep(3 * time.Millisecond)
				third := sendText(t, repo, state, state.groupID, state.userA, "m3")

				page1, err := repo.History(state.ctx, state.groupID, &PaginationParams{Limit: 2})
				if err != nil {
					t.Fatalf("history page1: %v", err)
				}
				if len(page1.Items) != 2 || page1.NextCursor == nil {
					t.Fatalf("page1 = %d items next=%v", len(page1.Items), page1.NextCursor)
				}
				if got := historyIDs(page1.Items); got[0] != third.MessageID || got[1] != second.MessageID {
					t.Fatalf("page1 order = %v, want [third second]", got)
				}
				if _, err := time.Parse(time.RFC3339Nano, page1.Items[0].SentAt); err != nil {
					t.Fatalf("sent_at must be RFC3339: %q", page1.Items[0].SentAt)
				}
				page2, err := repo.History(state.ctx, state.groupID, &PaginationParams{Limit: 2, Cursor: *page1.NextCursor})
				if err != nil {
					t.Fatalf("history page2: %v", err)
				}
				if len(page2.Items) != 1 || page2.NextCursor != nil || page2.Items[0].MessageID != first.MessageID {
					t.Fatalf("page2 = %v next=%v", historyIDs(page2.Items), page2.NextCursor)
				}
				if _, err := repo.History(state.ctx, state.groupID, &PaginationParams{Cursor: "not-a-cursor"}); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
		{
			Name: "update enforces owner text and recalled guards",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				sent := sendText(t, repo, state, state.groupID, state.userA, "before")
				updated, err := repo.Update(state.ctx, UpdateGroupMessageRequest{MessageID: sent.MessageID, UserID: state.userA, Content: "after"})
				if err != nil || updated.MessageID != sent.MessageID || updated.Content != "after" {
					t.Fatalf("update = %#v err=%v", updated, err)
				}
				page, err := repo.History(state.ctx, state.groupID, &PaginationParams{Limit: 10})
				if err != nil {
					t.Fatalf("history: %v", err)
				}
				var item *GroupMessageItem
				for i := range page.Items {
					if page.Items[i].MessageID == sent.MessageID {
						item = &page.Items[i]
					}
				}
				if item == nil || item.Content == nil || *item.Content != "after" {
					t.Fatalf("history item = %#v", item)
				}
				if item.EditedAt == nil || *item.EditedAt == "" {
					t.Fatalf("edited_at must be set: %#v", item)
				}
				if _, err := repo.Update(state.ctx, UpdateGroupMessageRequest{MessageID: sent.MessageID, UserID: state.userB, Content: "hack"}); !errors.Is(err, ErrPermissionDenied) {
					t.Fatalf("non-owner update err = %v, want ErrPermissionDenied", err)
				}
				nonText, err := repo.Send(state.ctx, SendGroupMessageRequest{GroupID: state.groupID, SenderUserID: state.userA, Type: "image", Content: ""})
				if err != nil {
					t.Fatalf("send non-text: %v", err)
				}
				if _, err := repo.Update(state.ctx, UpdateGroupMessageRequest{MessageID: nonText.MessageID, UserID: state.userA, Content: "x"}); !errors.Is(err, ErrMessageNotEditable) {
					t.Fatalf("non-text update err = %v, want ErrMessageNotEditable", err)
				}
				if _, err := repo.Update(state.ctx, UpdateGroupMessageRequest{MessageID: "gm_missing", UserID: state.userA, Content: "x"}); !errors.Is(err, ErrMessageNotFound) {
					t.Fatalf("missing update err = %v, want ErrMessageNotFound", err)
				}
			},
		},
		{
			Name: "recall enforces window and admin path bypasses it",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				sent := sendText(t, repo, state, state.groupID, state.userA, "recall me")
				if _, err := repo.Recall(state.ctx, time.Minute, RecallGroupMessageRequest{MessageID: sent.MessageID, UserID: state.userA}); err != nil {
					t.Fatalf("recall within window: %v", err)
				}
				if _, err := repo.Recall(state.ctx, time.Minute, RecallGroupMessageRequest{MessageID: sent.MessageID, UserID: state.userA}); !errors.Is(err, ErrMessageRecalled) {
					t.Fatalf("double recall err = %v, want ErrMessageRecalled", err)
				}
				page, err := repo.History(state.ctx, state.groupID, &PaginationParams{Limit: 10})
				if err != nil {
					t.Fatalf("history: %v", err)
				}
				foundRecalled := false
				for _, item := range page.Items {
					if item.MessageID != sent.MessageID {
						continue
					}
					foundRecalled = true
					if item.RecalledAt == nil || *item.RecalledAt == "" {
						t.Fatalf("recalled_at must be set: %#v", item)
					}
					if item.ReadCount != nil {
						t.Fatalf("recalled message read_count must stay null: %#v", item.ReadCount)
					}
				}
				if !foundRecalled {
					t.Fatalf("recalled message missing from history")
				}
				disabled := sendText(t, repo, state, state.groupID, state.userA, "window disabled")
				if _, err := repo.Recall(state.ctx, 0, RecallGroupMessageRequest{MessageID: disabled.MessageID, UserID: state.userA}); !errors.Is(err, ErrMessageRecallWindow) {
					t.Fatalf("window=0 recall err = %v, want ErrMessageRecallWindow", err)
				}
				expired := sendText(t, repo, state, state.groupID, state.userA, "expired window")
				if _, err := repo.Recall(state.ctx, -time.Minute, RecallGroupMessageRequest{MessageID: expired.MessageID, UserID: state.userA}); !errors.Is(err, ErrMessageRecallWindow) {
					t.Fatalf("negative window recall err = %v, want ErrMessageRecallWindow", err)
				}
				adminTarget := sendText(t, repo, state, state.groupID, state.userA, "admin recall")
				meta, err := repo.RecallAsAdmin(state.ctx, adminTarget.MessageID, "it_gm_admin")
				if err != nil || meta.MessageID != adminTarget.MessageID {
					t.Fatalf("admin recall = %#v err=%v", meta, err)
				}
				if _, err := repo.RecallAsAdmin(state.ctx, adminTarget.MessageID, "it_gm_admin"); !errors.Is(err, ErrMessageRecalled) {
					t.Fatalf("double admin recall err = %v, want ErrMessageRecalled", err)
				}
				if _, err := repo.RecallAsAdmin(state.ctx, "gm_missing", "it_gm_admin"); !errors.Is(err, ErrMessageNotFound) {
					t.Fatalf("admin recall missing err = %v, want ErrMessageNotFound", err)
				}
			},
		},
		{
			Name: "read counts follow read positions and membership join time",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				first := sendText(t, repo, state, state.groupID, state.userA, "r1")
				time.Sleep(3 * time.Millisecond)
				second := sendText(t, repo, state, state.groupID, state.userA, "r2")

				if err := repo.MarkAsRead(state.ctx, state.userA, MarkGroupMessagesReadRequest{GroupID: state.groupID, UpToMessageID: first.MessageID}); err != nil {
					t.Fatalf("A mark read up to first: %v", err)
				}
				// 同一用户重复标记同一位置：不得重复计数（PG 是 read_state upsert）。
				if err := repo.MarkAsRead(state.ctx, state.userA, MarkGroupMessagesReadRequest{GroupID: state.groupID, UpToMessageID: first.MessageID}); err != nil {
					t.Fatalf("A mark read again: %v", err)
				}
				counts, err := repo.GetReadCounts(state.ctx, state.groupID, []string{first.MessageID, second.MessageID})
				if err != nil {
					t.Fatalf("read counts: %v", err)
				}
				if counts[first.MessageID] != 1 || counts[second.MessageID] != 0 {
					t.Fatalf("after A: counts = %#v, want first=1 second=0", counts)
				}
				if err := repo.MarkAsRead(state.ctx, state.userB, MarkGroupMessagesReadRequest{GroupID: state.groupID, UpToMessageID: second.MessageID}); err != nil {
					t.Fatalf("B mark read: %v", err)
				}
				// C 在消息之后才加入：即使已读位点覆盖，也不计入更早消息的已读人数。
				if err := repo.MarkAsRead(state.ctx, state.userC, MarkGroupMessagesReadRequest{GroupID: state.groupID, UpToMessageID: second.MessageID}); err != nil {
					t.Fatalf("C mark read: %v", err)
				}
				counts, err = repo.GetReadCounts(state.ctx, state.groupID, []string{first.MessageID, second.MessageID})
				if err != nil {
					t.Fatalf("read counts after B/C: %v", err)
				}
				if counts[first.MessageID] != 2 || counts[second.MessageID] != 1 {
					t.Fatalf("after B/C: counts = %#v, want first=2 second=1", counts)
				}
				if count, err := repo.GetReadCount(state.ctx, state.groupID, second.MessageID); err != nil || count != 1 {
					t.Fatalf("single read count = %d err=%v", count, err)
				}
				// 不存在的消息：PG COUNT 返回 0，不是 10300。
				if count, err := repo.GetReadCount(state.ctx, state.groupID, "gm_missing"); err != nil || count != 0 {
					t.Fatalf("missing message read count = %d err=%v, want 0 nil", count, err)
				}
				// 非成员标记读：up_to 路径 10300（PG JOIN group_members 0 行）。
				if err := repo.MarkAsRead(state.ctx, "it_gm_non_member", MarkGroupMessagesReadRequest{GroupID: state.groupID, UpToMessageID: first.MessageID}); !errors.Is(err, ErrMessageNotFound) {
					t.Fatalf("non-member mark read err = %v, want ErrMessageNotFound", err)
				}
				// 无 up_to：非成员为 no-op 成功（PG upsert WHERE members... 0 行）。
				if err := repo.MarkAsRead(state.ctx, "it_gm_non_member", MarkGroupMessagesReadRequest{GroupID: state.groupID}); err != nil {
					t.Fatalf("non-member mark latest must be no-op: %v", err)
				}
			},
		},
		{
			Name: "mark all read returns member groups and advances positions",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				first := sendText(t, repo, state, state.groupID, state.userA, "all-1")
				time.Sleep(3 * time.Millisecond)
				second := sendText(t, repo, state, state.groupID, state.userA, "all-2")
				groups, err := repo.MarkAllRead(state.ctx, state.userA)
				if err != nil {
					t.Fatalf("mark all read: %v", err)
				}
				sortedGroups := append([]string(nil), groups...)
				sort.Strings(sortedGroups)
				want := []string{state.groupID, state.otherGroup}
				sort.Strings(want)
				if strings.Join(sortedGroups, ",") != strings.Join(want, ",") {
					t.Fatalf("mark all read groups = %v, want %v", groups, want)
				}
				if count, err := repo.GetReadCount(state.ctx, state.groupID, second.MessageID); err != nil || count != 1 {
					t.Fatalf("A read count after mark all = %d err=%v, want 1", count, err)
				}
				if count, err := repo.GetReadCount(state.ctx, state.groupID, first.MessageID); err != nil || count != 1 {
					t.Fatalf("A read count first after mark all = %d err=%v, want 1", count, err)
				}
			},
		},
		{
			Name: "membership filtering and group name lookups",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				if member, err := repo.IsMember(state.ctx, state.groupID, state.userA); err != nil || !member {
					t.Fatalf("A member = %v err=%v", member, err)
				}
				if member, err := repo.IsMember(state.ctx, state.groupID, "it_gm_non_member"); err != nil || member {
					t.Fatalf("non-member = %v err=%v", member, err)
				}
				filtered, err := repo.FilterMemberIDs(state.ctx, state.groupID, []string{state.userA, "it_gm_non_member", state.userB})
				if err != nil {
					t.Fatalf("filter members: %v", err)
				}
				sort.Strings(filtered)
				wantFiltered := []string{state.userA, state.userB}
				sort.Strings(wantFiltered)
				if strings.Join(filtered, ",") != strings.Join(wantFiltered, ",") {
					t.Fatalf("filtered = %v, want %v", filtered, wantFiltered)
				}
				if empty, err := repo.FilterMemberIDs(state.ctx, state.groupID, nil); err != nil || len(empty) != 0 {
					t.Fatalf("empty filter = %v err=%v", empty, err)
				}
				name, found, err := repo.GetGroupName(state.ctx, state.groupID)
				if err != nil || !found || name != "Contract Group" {
					t.Fatalf("group name = %q found=%v err=%v", name, found, err)
				}
				if _, found, err := repo.GetGroupName(state.ctx, "it_gm_missing_group"); err != nil || found {
					t.Fatalf("missing group name found=%v err=%v", found, err)
				}
			},
		},
		{
			Name: "message auxiliary lookups respect recalled semantics",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				sent := sendText(t, repo, state, state.groupID, state.userA, "lookup me")
				exists, sender, err := repo.MessageExists(state.ctx, sent.MessageID)
				if err != nil || !exists || sender != state.userA {
					t.Fatalf("message exists = %v sender=%q err=%v", exists, sender, err)
				}
				groupID, found, err := repo.MessageGroupID(state.ctx, sent.MessageID)
				if err != nil || !found || groupID != state.groupID {
					t.Fatalf("message group = %q found=%v err=%v", groupID, found, err)
				}
				meta, found, err := repo.LookupMessageMeta(state.ctx, sent.MessageID)
				if err != nil || !found || meta.GroupID != state.groupID || meta.SenderUserID != state.userA {
					t.Fatalf("lookup meta = %#v found=%v err=%v", meta, found, err)
				}
				reply, found, err := repo.GetMessageForReply(state.ctx, sent.MessageID)
				if err != nil || !found {
					t.Fatalf("reply brief found=%v err=%v", found, err)
				}
				if reply.Type != "text" || reply.Content == nil || *reply.Content != "lookup me" || reply.Sender.UserID != state.userA || reply.Sender.Nickname == "" {
					t.Fatalf("reply brief = %#v", reply)
				}
				if exists, _, err := repo.MessageExists(state.ctx, "gm_missing"); err != nil || exists {
					t.Fatalf("missing exists err=%v", err)
				}
				if _, found, err := repo.MessageGroupID(state.ctx, "gm_missing"); err != nil || found {
					t.Fatalf("missing group found=%v err=%v", found, err)
				}
				if _, found, err := repo.LookupMessageMeta(state.ctx, "gm_missing"); err != nil || found {
					t.Fatalf("missing lookup meta found=%v err=%v", found, err)
				}
				if _, found, err := repo.GetMessageForReply(state.ctx, "gm_missing"); err != nil || found {
					t.Fatalf("missing reply brief found=%v err=%v", found, err)
				}

				if _, err := repo.RecallAsAdmin(state.ctx, sent.MessageID, "it_gm_admin"); err != nil {
					t.Fatalf("recall for aux lookups: %v", err)
				}
				// 撤回后：MessageGroupID 按不存在处理（反应可见性口径），
				// MessageExists / LookupMessageMeta / GetMessageForReply 仍可读真值（PG 无过滤）。
				if _, found, err := repo.MessageGroupID(state.ctx, sent.MessageID); err != nil || found {
					t.Fatalf("recalled message group found=%v err=%v", found, err)
				}
				if exists, _, err := repo.MessageExists(state.ctx, sent.MessageID); err != nil || !exists {
					t.Fatalf("recalled message exists=%v err=%v", exists, err)
				}
				if _, found, err := repo.LookupMessageMeta(state.ctx, sent.MessageID); err != nil || !found {
					t.Fatalf("recalled lookup meta found=%v err=%v", found, err)
				}
			},
		},
		{
			Name: "report snapshot reads message truth",
			Run: func(t *testing.T, repo Repository, state gmContractState) {
				sent := sendText(t, repo, state, state.groupID, state.userA, "snapshot me")
				snapshot, found, err := repo.GetMessageReportSnapshot(state.ctx, sent.MessageID)
				if err != nil || !found {
					t.Fatalf("snapshot found=%v err=%v", found, err)
				}
				if snapshot.GroupID != state.groupID || snapshot.Type != "text" || snapshot.Content != "snapshot me" || snapshot.SentAt == "" {
					t.Fatalf("snapshot = %#v", snapshot)
				}
				if _, found, err := repo.GetMessageReportSnapshot(state.ctx, "gm_missing"); err != nil || found {
					t.Fatalf("missing snapshot found=%v err=%v", found, err)
				}
			},
		},
	})
}
