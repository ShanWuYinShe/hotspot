package groups

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// covRepo 在既有 groupErrorRepo 之上补齐 Repository 剩余方法（内嵌 Repository
// 为 nil，缺失方法被调用会 panic），并为错误注入增加少量可写字段。仅测试用。
type covRepo struct {
	*groupErrorRepo

	dismissAsAdminHit bool
	dismissAsAdminErr error

	listAdminMembersItems []AdminGroupMemberItem
	listAdminMembersCur   string
	listAdminMembersErr   error

	areMembers    map[string]bool
	areMembersErr error

	passwordHash    *string
	passwordHashErr error

	listMyApplicationsItems []GroupApplicationItem
	listMyApplicationsCur   string
	listMyApplicationsErr   error

	updatePreferencesErr error
	leaveGroupErr        error

	sentInvitesItems []GroupSentInviteItem
	sentInvitesCur   string
	sentInvitesErr   error

	muteMemberErr   error
	unmuteMemberErr error
	bannedMem       []AdminBannedMemberItem
	bannedMemCur    string
	bannedMemErr    error
	setRoleErr      error
	banMemberErr    error
	unbanMemberErr  error

	listGroupsByUserResult PaginatedGroups
	listGroupsByUserErr    error
	countCommonGroups      int
	countCommonGroupsErr   error
	countOwnedGroups       int
	countOwnedGroupsErr    error
}

func (r *covRepo) DismissAsAdmin(context.Context, string) (bool, error) {
	return r.dismissAsAdminHit, r.dismissAsAdminErr
}

func (r *covRepo) ListAdminMembers(context.Context, string, *GroupRole, int, string) ([]AdminGroupMemberItem, string, error) {
	return r.listAdminMembersItems, r.listAdminMembersCur, r.listAdminMembersErr
}

func (r *covRepo) AreMembers(context.Context, []string, string) (map[string]bool, error) {
	return r.areMembers, r.areMembersErr
}

func (r *covRepo) GetGroupPasswordHash(context.Context, string) (*string, error) {
	return r.passwordHash, r.passwordHashErr
}

func (r *covRepo) ListMyApplications(context.Context, string, *ApplicationStatus, int, string) ([]GroupApplicationItem, string, error) {
	return r.listMyApplicationsItems, r.listMyApplicationsCur, r.listMyApplicationsErr
}

func (r *covRepo) UpdatePreferences(context.Context, string, UpdateGroupPreferencesRequest) error {
	return r.updatePreferencesErr
}

func (r *covRepo) LeaveGroup(context.Context, string, string) error { return r.leaveGroupErr }

func (r *covRepo) ListSentInvites(context.Context, string, *ApplicationStatus, int, string) ([]GroupSentInviteItem, string, error) {
	return r.sentInvitesItems, r.sentInvitesCur, r.sentInvitesErr
}

func (r *covRepo) MuteMember(context.Context, string, MuteMemberRequest) error {
	return r.muteMemberErr
}

func (r *covRepo) UnmuteMember(context.Context, string, UnmuteMemberRequest) error {
	return r.unmuteMemberErr
}

func (r *covRepo) ListBannedMembers(context.Context, string, int, string) ([]AdminBannedMemberItem, string, error) {
	return r.bannedMem, r.bannedMemCur, r.bannedMemErr
}

func (r *covRepo) SetMemberRole(context.Context, string, SetMemberRoleRequest) error {
	return r.setRoleErr
}

func (r *covRepo) BanMember(context.Context, string, BanMemberRequest) error {
	return r.banMemberErr
}

func (r *covRepo) UnbanMember(context.Context, string, UnbanMemberRequest) error {
	return r.unbanMemberErr
}

func (r *covRepo) ListGroupsByUser(context.Context, string, int, string) (PaginatedGroups, error) {
	return r.listGroupsByUserResult, r.listGroupsByUserErr
}

func (r *covRepo) CountCommonGroups(context.Context, string, string) (int, error) {
	return r.countCommonGroups, r.countCommonGroupsErr
}

func (r *covRepo) CountOwnedGroups(context.Context, string) (int, error) {
	return r.countOwnedGroups, r.countOwnedGroupsErr
}

// covNewRepo 构造带常见默认值的测试仓储：可指定 Get 返回的群项与用户角色。
func covNewRepo(group GroupItem) *covRepo {
	return &covRepo{groupErrorRepo: &groupErrorRepo{getResult: group, getFound: true}}
}

func covOwnerGroup() GroupItem {
	return GroupItem{Group: GroupInfo{GroupID: "g1", Name: "cov"}, ViewerState: &GroupViewerState{Role: GroupRoleOwner}}
}

func covAdminGroup() GroupItem {
	return GroupItem{Group: GroupInfo{GroupID: "g1", Name: "cov"}, ViewerState: &GroupViewerState{Role: GroupRoleAdmin}}
}

func covMemberGroup() GroupItem {
	return GroupItem{Group: GroupInfo{GroupID: "g1", Name: "cov"}, ViewerState: &GroupViewerState{Role: GroupRoleMember}}
}

// covRunHandler 以指定请求体/身份调用 handler，返回业务错误码。
func covRunHandler(t *testing.T, repo Repository, fn func(*Handler) http.HandlerFunc, body, userID string, withIdempotencyKey bool) httpx.Code {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if withIdempotencyKey {
		req.Header.Set("Idempotency-Key", "cov0000000000000000000000000000000001")
	}
	if userID != "" {
		req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: userID, SessionID: "sess_" + userID}))
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	rr := httptest.NewRecorder()
	fn(handler)(rr, req)
	return testdb.DecodeAPICode(t, rr)
}

// covCase 描述一条 handler 错误路径断言。
type covCase struct {
	name    string
	repo    Repository
	fn      func(*Handler) http.HandlerFunc
	body    string
	userID  string
	idemKey bool
	want    httpx.Code
}

func covRunCases(t *testing.T, cases []covCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := covRunHandler(t, tc.repo, tc.fn, tc.body, tc.userID, tc.idemKey); got != tc.want {
				t.Fatalf("code = %d, want %d", got, tc.want)
			}
		})
	}
}
