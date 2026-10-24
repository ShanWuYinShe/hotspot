package groups

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// groupsContractState 契约用例上下文：两实现都需要预置用户行（PG 侧 group_members
// 有 users 外键、昵称 JOIN；内存侧昵称回退 user_id），准备逻辑放在 ContractImpl.New。
type groupsContractState struct {
	ctx     context.Context
	owner   string
	members []string
}

// TestGroupsRepositoryContract 接入双实现契约一致性基建（groups 模块）：
// 建群/入群/成员列表分页/公告排序与上限/禁言封禁状态机/退群/我的群组/
// 共同群聊计数 等行为必须与真实库实现一致。昵称仅断言非空（内存实现无用户目录，
// 回退 user_id；真实库取 users.nickname），其余字段逐项对齐。
func TestGroupsRepositoryContract(t *testing.T) {
	owner := "it_groups_contract_owner"
	members := []string{"it_groups_contract_m1", "it_groups_contract_m2", "it_groups_contract_m3"}

	impls := []testdb.ContractImpl[Repository, groupsContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, groupsContractState, func()) {
				return NewInMemoryRepository(), groupsContractState{ctx: context.Background(), owner: owner, members: members}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, groupsContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				for _, uid := range append([]string{owner}, members...) {
					testdb.InsertUser(t, txFactory, uid, "nick_"+uid)
				}
				return NewPostgresRepository(txFactory), groupsContractState{ctx: context.Background(), owner: owner, members: members}, cleanup
			},
		},
	}

	createGroup := func(t *testing.T, repo Repository, ownerID, name string) string {
		t.Helper()
		group, err := repo.Create(context.Background(), ownerID, CreateGroupRequest{Name: name})
		if err != nil {
			t.Fatalf("create group %s: %v", name, err)
		}
		if group.Group.GroupID == "" {
			t.Fatal("created group id must be non-empty")
		}
		return group.Group.GroupID
	}

	joinGroup := func(t *testing.T, repo Repository, userID, groupID string) EnterGroupResult {
		t.Helper()
		result, err := repo.JoinGroup(context.Background(), userID, groupID, nil)
		if err != nil {
			t.Fatalf("join %s: %v", userID, err)
		}
		return result
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, groupsContractState]{
		{
			Name: "create group seeds owner membership",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				groupID := createGroup(t, repo, state.owner, "Contract Owners")
				found, ok, err := repo.Get(state.ctx, groupID)
				if err != nil {
					t.Fatalf("get: %v", err)
				}
				if !ok || found.Group.Name != "Contract Owners" {
					t.Fatalf("get = %#v ok=%v", found.Group, ok)
				}
				items, next, err := repo.ListMembers(state.ctx, groupID, nil, 10, "")
				if err != nil {
					t.Fatalf("list members: %v", err)
				}
				if next != "" {
					t.Fatalf("unexpected next cursor %q", next)
				}
				if len(items) != 1 || items[0].User.UserID != state.owner || items[0].Role != GroupRoleOwner {
					t.Fatalf("members = %#v", items)
				}
				owned, err := repo.CountOwnedGroups(state.ctx, state.owner)
				if err != nil {
					t.Fatalf("count owned: %v", err)
				}
				if owned != 1 {
					t.Fatalf("owned = %d, want 1", owned)
				}
				joined, err := repo.IsMember(state.ctx, groupID, state.owner)
				if err != nil {
					t.Fatalf("is member: %v", err)
				}
				if !joined {
					t.Fatal("owner must be a member")
				}
				batch, err := repo.AreMembers(state.ctx, []string{groupID, "group_missing"}, state.owner)
				if err != nil {
					t.Fatalf("are members: %v", err)
				}
				if !batch[groupID] {
					t.Fatalf("are members = %#v", batch)
				}
				if batch["group_missing"] {
					t.Fatal("missing group must not be reported as member")
				}
			},
		},
		{
			Name: "join group and member pagination is stable",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				groupID := createGroup(t, repo, state.owner, "Contract Joins")
				for _, userID := range state.members {
					result := joinGroup(t, repo, userID, groupID)
					if result.Status != EnterGroupStatusJoined {
						t.Fatalf("join %s status = %q", userID, result.Status)
					}
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					items, next, err := repo.ListMembers(state.ctx, groupID, nil, 2, cursor)
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(items))
					}
					for _, item := range items {
						if item.JoinedAt == "" {
							t.Fatal("joined_at must be non-empty")
						}
						if _, err := time.Parse(time.RFC3339, item.JoinedAt); err != nil {
							t.Fatalf("joined_at must be RFC3339: %q", item.JoinedAt)
						}
						if item.User.Nickname == "" {
							t.Fatal("member nickname must be non-empty")
						}
						if seen[item.User.UserID] {
							t.Fatalf("duplicate member %s across pages", item.User.UserID)
						}
						seen[item.User.UserID] = true
					}
					pages++
					if next == "" {
						break
					}
					cursor = next
					if pages > 5 {
						t.Fatal("member pagination did not terminate")
					}
				}
				if len(seen) != len(state.members)+1 {
					t.Fatalf("seen %d members, want %d", len(seen), len(state.members)+1)
				}
				role := GroupRoleMember
				memberItems, _, err := repo.ListMembers(state.ctx, groupID, &role, 10, "")
				if err != nil {
					t.Fatalf("list members by role: %v", err)
				}
				if len(memberItems) != len(state.members) {
					t.Fatalf("role filter returned %d, want %d", len(memberItems), len(state.members))
				}
			},
		},
		{
			Name: "requires approval creates and dedupes pending application",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				requiresApproval := true
				group, err := repo.Create(state.ctx, state.owner, CreateGroupRequest{Name: "Contract Approval", RequiresApproval: &requiresApproval})
				if err != nil {
					t.Fatalf("create: %v", err)
				}
				groupID := group.Group.GroupID
				first, err := repo.JoinGroup(state.ctx, state.members[0], groupID, nil)
				if err != nil {
					t.Fatalf("join: %v", err)
				}
				if first.Status != EnterGroupStatusPendingApproval || first.ApplicationID == nil {
					t.Fatalf("first join = %#v", first)
				}
				if !first.IsNewApplication {
					t.Fatal("first application must be new")
				}
				second, err := repo.JoinGroup(state.ctx, state.members[0], groupID, nil)
				if err != nil {
					t.Fatalf("second join: %v", err)
				}
				if second.Status != EnterGroupStatusPendingApproval || second.ApplicationID == nil || *second.ApplicationID != *first.ApplicationID {
					t.Fatalf("second join = %#v, first = %#v", second, first)
				}
				if second.IsNewApplication {
					t.Fatal("deduped application must not be marked new")
				}
				application, ok, err := repo.GetApplication(state.ctx, *first.ApplicationID)
				if err != nil {
					t.Fatalf("get application: %v", err)
				}
				if !ok || application.Status != ApplicationStatusPending {
					t.Fatalf("application = %#v ok=%v", application, ok)
				}
				items, _, err := repo.ListMembers(state.ctx, groupID, nil, 10, "")
				if err != nil {
					t.Fatalf("list members: %v", err)
				}
				if len(items) != 1 {
					t.Fatalf("pending applicant must not become member: %#v", items)
				}
			},
		},
		{
			Name: "invite only rejects direct join",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				group, err := repo.Create(state.ctx, state.owner, CreateGroupRequest{Name: "Contract Invite Only", JoinMethod: GroupJoinMethodInviteOnly})
				if err != nil {
					t.Fatalf("create: %v", err)
				}
				if _, err := repo.JoinGroup(state.ctx, state.members[0], group.Group.GroupID, nil); !errors.Is(err, ErrAccessRestricted) {
					t.Fatalf("invite_only join err = %v, want ErrAccessRestricted", err)
				}
			},
		},
		{
			Name: "announcements pinned first and limited",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				groupID := createGroup(t, repo, state.owner, "Contract Announcements")
				first, err := repo.CreateAnnouncement(state.ctx, state.owner, CreateGroupAnnouncementRequest{GroupID: groupID, Title: "first", Content: "body"})
				if err != nil {
					t.Fatalf("create announcement 1: %v", err)
				}
				pinned := true
				second, err := repo.CreateAnnouncement(state.ctx, state.owner, CreateGroupAnnouncementRequest{GroupID: groupID, Title: "pinned", Content: "body", IsPinned: &pinned})
				if err != nil {
					t.Fatalf("create announcement 2: %v", err)
				}
				third, err := repo.CreateAnnouncement(state.ctx, state.owner, CreateGroupAnnouncementRequest{GroupID: groupID, Title: "third", Content: "body"})
				if err != nil {
					t.Fatalf("create announcement 3: %v", err)
				}
				list, err := repo.ListAnnouncements(state.ctx, groupID, 0)
				if err != nil {
					t.Fatalf("list announcements: %v", err)
				}
				if len(list) != 3 {
					t.Fatalf("announcements = %d, want 3 (default limit)", len(list))
				}
				if !list[0].IsPinned || list[0].AnnouncementID != second.AnnouncementID {
					t.Fatalf("first announcement = %#v, want pinned %s", list[0], second.AnnouncementID)
				}
				rest := map[string]bool{}
				for _, item := range list[1:] {
					rest[item.AnnouncementID] = true
				}
				if !rest[first.AnnouncementID] || !rest[third.AnnouncementID] {
					t.Fatalf("remaining announcements = %#v", rest)
				}
				limited, err := repo.ListAnnouncements(state.ctx, groupID, 1)
				if err != nil {
					t.Fatalf("list announcements limit 1: %v", err)
				}
				if len(limited) != 1 || !limited[0].IsPinned {
					t.Fatalf("limit 1 = %#v", limited)
				}
				capped, err := repo.ListAnnouncements(state.ctx, groupID, 100)
				if err != nil {
					t.Fatalf("list announcements limit 100: %v", err)
				}
				if len(capped) != 3 {
					t.Fatalf("hard-capped list = %d, want 3", len(capped))
				}
			},
		},
		{
			Name: "mute and ban state machine",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				groupID := createGroup(t, repo, state.owner, "Contract Sanctions")
				joinGroup(t, repo, state.members[0], groupID)

				if err := repo.MuteMember(state.ctx, state.owner, MuteMemberRequest{GroupID: groupID, UserID: state.members[0], Active: true, DurationMinutes: 10}); err != nil {
					t.Fatalf("mute: %v", err)
				}
				adminItems, _, err := repo.ListAdminMembers(state.ctx, groupID, nil, 10, "")
				if err != nil {
					t.Fatalf("list admin members: %v", err)
				}
				mutedFound := false
				for _, item := range adminItems {
					if item.User.UserID == state.members[0] {
						mutedFound = true
						if item.MutedUntil == nil {
							t.Fatalf("muted_until must be set: %#v", item)
						}
					}
				}
				if !mutedFound {
					t.Fatal("muted member missing from admin list")
				}
				if err := repo.UnmuteMember(state.ctx, state.owner, UnmuteMemberRequest{GroupID: groupID, UserID: state.members[0]}); err != nil {
					t.Fatalf("unmute: %v", err)
				}

				reason := "spam"
				if err := repo.BanMember(state.ctx, state.owner, BanMemberRequest{GroupID: groupID, UserID: state.members[0], Reason: &reason}); err != nil {
					t.Fatalf("ban: %v", err)
				}
				banned, err := repo.IsBanned(state.ctx, groupID, state.members[0])
				if err != nil {
					t.Fatalf("is banned: %v", err)
				}
				if !banned {
					t.Fatal("member must be banned")
				}
				bannedList, _, err := repo.ListBannedMembers(state.ctx, groupID, 10, "")
				if err != nil {
					t.Fatalf("list banned members: %v", err)
				}
				if len(bannedList) != 1 || bannedList[0].User.UserID != state.members[0] {
					t.Fatalf("banned members = %#v", bannedList)
				}
				if bannedList[0].Reason == nil || *bannedList[0].Reason != "spam" {
					t.Fatalf("ban reason = %#v", bannedList[0].Reason)
				}
				if bannedList[0].BannedBy == nil || bannedList[0].BannedBy.UserID != state.owner {
					t.Fatalf("banned_by = %#v", bannedList[0].BannedBy)
				}
				members, _, err := repo.ListMembers(state.ctx, groupID, nil, 10, "")
				if err != nil {
					t.Fatalf("list members: %v", err)
				}
				for _, item := range members {
					if item.User.UserID == state.members[0] {
						t.Fatal("banned member must be removed from member list")
					}
				}
				if err := repo.UnbanMember(state.ctx, state.owner, UnbanMemberRequest{GroupID: groupID, UserID: state.members[0]}); err != nil {
					t.Fatalf("unban: %v", err)
				}
				banned, err = repo.IsBanned(state.ctx, groupID, state.members[0])
				if err != nil {
					t.Fatalf("is banned after unban: %v", err)
				}
				if banned {
					t.Fatal("member must be unbanned")
				}
			},
		},
		{
			Name: "leave group removes membership and my groups entry",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				groupID := createGroup(t, repo, state.owner, "Contract Leave")
				joinGroup(t, repo, state.members[0], groupID)
				mine, next, err := repo.ListMyGroups(state.ctx, state.members[0], 10, "")
				if err != nil {
					t.Fatalf("list my groups: %v", err)
				}
				if len(mine) != 1 || next != "" {
					t.Fatalf("my groups = %#v next=%q", mine, next)
				}
				if err := repo.LeaveGroup(state.ctx, state.members[0], groupID); err != nil {
					t.Fatalf("leave group: %v", err)
				}
				joined, err := repo.IsMember(state.ctx, groupID, state.members[0])
				if err != nil {
					t.Fatalf("is member after leave: %v", err)
				}
				if joined {
					t.Fatal("left member must not be member")
				}
				mine, _, err = repo.ListMyGroups(state.ctx, state.members[0], 10, "")
				if err != nil {
					t.Fatalf("list my groups after leave: %v", err)
				}
				if len(mine) != 0 {
					t.Fatalf("my groups after leave = %#v", mine)
				}
			},
		},
		{
			Name: "count common groups counts shared memberships",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				first := createGroup(t, repo, state.owner, "Contract Common A")
				second := createGroup(t, repo, state.owner, "Contract Common B")
				for _, groupID := range []string{first, second} {
					joinGroup(t, repo, state.members[0], groupID)
				}
				common, err := repo.CountCommonGroups(state.ctx, state.owner, state.members[0])
				if err != nil {
					t.Fatalf("count common groups: %v", err)
				}
				if common != 2 {
					t.Fatalf("common = %d, want 2", common)
				}
				none, err := repo.CountCommonGroups(state.ctx, state.owner, state.members[1])
				if err != nil {
					t.Fatalf("count common groups (none): %v", err)
				}
				if none != 0 {
					t.Fatalf("common (none) = %d, want 0", none)
				}
			},
		},
		{
			Name: "invalid member cursor is rejected",
			Run: func(t *testing.T, repo Repository, state groupsContractState) {
				groupID := createGroup(t, repo, state.owner, "Contract Cursor")
				if _, _, err := repo.ListMembers(state.ctx, groupID, nil, 10, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
	})
}
