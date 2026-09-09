package groups

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/rbac"
	"hotspot/backend-go/internal/platform/types"
)

func TestServiceRoleOf(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)

	// 1. 未登录上下文
	role, found, err := svc.RoleOf(context.Background(), "grp_1")
	if err != nil || found || role != "" {
		t.Fatalf("expected anonymous RoleOf to return found=false, got role=%v, found=%v, err=%v", role, found, err)
	}

	// 2. 登录但群组不存在
	ctxViewer := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "usr_alice"})
	role, found, err = svc.RoleOf(ctxViewer, "grp_nonexistent")
	if err != nil || found || role != "" {
		t.Fatalf("expected nonexistent group RoleOf to return found=false, got found=%v, err=%v", found, err)
	}

	// 3. 创建群组，alice 为群主
	createRes, err := repo.Create(ctxViewer, "usr_alice", CreateGroupRequest{
		Name:        "测试管理群",
		Description: "描述",
	})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	// alice 查自身角色：应为 owner
	role, found, err = svc.RoleOf(ctxViewer, createRes.Group.GroupID)
	if err != nil || !found || role != rbac.GroupRoleOwner {
		t.Fatalf("expected owner role, got role=%v, found=%v, err=%v", role, found, err)
	}

	// 非群成员 bob 查角色：ViewerState 应为 nil，found 为 false
	ctxBob := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "usr_bob"})
	role, found, err = svc.RoleOf(ctxBob, createRes.Group.GroupID)
	if err != nil || found || role != "" {
		t.Fatalf("expected non-member RoleOf to return found=false, got found=%v", found)
	}
}

func TestServiceAndHandlerGetSettings(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	handler := NewHandler(svc)

	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "usr_alice"})
	res, err := repo.Create(ctxAlice, "usr_alice", CreateGroupRequest{
		Name:             "公开设置群",
		JoinMethod:       "password",
		Password:         "123456",
		RequiresApproval: boolPtr(true),
		IsSearchable:     boolPtr(true),
		MaxMembers:       200,
	})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	// 1. 正常获取群设置
	settings, ok, err := svc.GetSettings(ctxAlice, res.Group.GroupID)
	if err != nil || !ok {
		t.Fatalf("expected settings ok, got ok=%v, err=%v", ok, err)
	}
	if settings.JoinMethod != "password" || !settings.HasPassword || !settings.RequiresApproval || settings.MaxMembers != 200 {
		t.Fatalf("unexpected settings: %+v", settings)
	}

	// 2. 不存在的群设置
	_, ok, err = svc.GetSettings(ctxAlice, "grp_missing")
	if err != nil || ok {
		t.Fatalf("expected ok=false for missing group, got ok=%v", ok)
	}

	// 3. Handler 缺少 group_id
	rr1 := httptest.NewRecorder()
	handler.GetSettings(rr1, httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(`{}`)))
	var resp1 httpx.ApiResponse[struct{}]
	_ = json.NewDecoder(rr1.Body).Decode(&resp1)
	if resp1.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected CodeInvalidRequest, got %d", resp1.Code)
	}

	// 4. Handler 正常调用
	rr2 := httptest.NewRecorder()
	handler.GetSettings(rr2, httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(`{"group_id":"`+res.Group.GroupID+`"}`)))
	var resp2 httpx.ApiResponse[GroupSettings]
	_ = json.NewDecoder(rr2.Body).Decode(&resp2)
	if resp2.Code != 0 || resp2.Data == nil || resp2.Data.MaxMembers != 200 {
		t.Fatalf("expected success response, got %+v", resp2)
	}
}

func TestServiceAndHandlerRecallGroupMessageAsAdmin(t *testing.T) {
	orig := RecallMessageAsAdmin
	defer func() { RecallMessageAsAdmin = orig }()

	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	handler := NewHandler(svc)

	// 1. RecallMessageAsAdmin 未注入时报错
	RecallMessageAsAdmin = nil
	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "usr_alice"})
	err := svc.RecallGroupMessageAsAdmin(ctxAlice, RecallGroupMessageAdminRequest{
		GroupID:   "grp_1",
		MessageID: "msg_1",
	})
	if err == nil {
		t.Fatal("expected error when RecallMessageAsAdmin is nil")
	}

	// 2. 未登录上下文
	RecallMessageAsAdmin = func(ctx context.Context, groupID, messageID, adminUserID string) error {
		return nil
	}
	err = svc.RecallGroupMessageAsAdmin(context.Background(), RecallGroupMessageAdminRequest{
		GroupID:   "grp_1",
		MessageID: "msg_1",
	})
	if err == nil {
		t.Fatal("expected error for unauthenticated recall")
	}

	// 3. 正常撤回
	called := false
	RecallMessageAsAdmin = func(ctx context.Context, groupID, messageID, adminUserID string) error {
		called = true
		if groupID != "grp_1" || messageID != "msg_1" || adminUserID != "usr_alice" {
			t.Fatalf("unexpected recall args: %s, %s, %s", groupID, messageID, adminUserID)
		}
		return nil
	}
	err = svc.RecallGroupMessageAsAdmin(ctxAlice, RecallGroupMessageAdminRequest{
		GroupID:   "grp_1",
		MessageID: "msg_1",
	})
	if err != nil || !called {
		t.Fatalf("expected recall success, got err=%v", err)
	}

	// 4. 下层返回 APIError
	RecallMessageAsAdmin = func(ctx context.Context, groupID, messageID, adminUserID string) error {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "消息不存在"}
	}
	err = svc.RecallGroupMessageAsAdmin(ctxAlice, RecallGroupMessageAdminRequest{
		GroupID:   "grp_1",
		MessageID: "msg_1",
	})
	if err == nil {
		t.Fatal("expected APIError propagated")
	}

	// 5. 下层返回普通错误转为 10600
	RecallMessageAsAdmin = func(ctx context.Context, groupID, messageID, adminUserID string) error {
		return errors.New("db disconnect")
	}
	err = svc.RecallGroupMessageAsAdmin(ctxAlice, RecallGroupMessageAdminRequest{
		GroupID:   "grp_1",
		MessageID: "msg_1",
	})
	var apiErr *httpx.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected 10600 on generic error, got %v", err)
	}

	// 6. Handler 缺少参数与正常调用
	rr1 := httptest.NewRecorder()
	handler.RecallGroupMessageAsAdmin(rr1, httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"group_id":"grp_1"}`)))
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200 envelope, got %d", rr1.Code)
	}

	RecallMessageAsAdmin = func(ctx context.Context, groupID, messageID, adminUserID string) error {
		return nil
	}
	reqGood := httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"group_id":"grp_1","message_id":"msg_1"}`))
	reqGood = reqGood.WithContext(ctxAlice)
	rr2 := httptest.NewRecorder()
	handler.RecallGroupMessageAsAdmin(rr2, reqGood)
	var resp2 httpx.ApiResponse[struct{}]
	_ = json.NewDecoder(rr2.Body).Decode(&resp2)
	if resp2.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp2.Code)
	}
}

func TestServiceAndHandlerListAdminMembers(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	handler := NewHandler(svc)

	ctxAlice := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "usr_alice"})
	res, err := repo.Create(ctxAlice, "usr_alice", CreateGroupRequest{
		Name: "成员列表测试群",
	})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	// 1. Service ListAdminMembers 正常返回
	members, cursor, err := svc.ListAdminMembers(ctxAlice, ListAdminGroupMembersRequest{
		GroupID: res.Group.GroupID,
		Pagination: &types.PaginationParams{
			Limit: 10,
		},
	})
	if err != nil {
		t.Fatalf("list admin members failed: %v", err)
	}
	if len(members) == 0 {
		t.Fatal("expected at least 1 member (the owner)")
	}
	_ = cursor

	// 2. Handler 缺少 group_id
	rr1 := httptest.NewRecorder()
	handler.ListAdminMembers(rr1, httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{}`)))
	var resp1 httpx.ApiResponse[struct{}]
	_ = json.NewDecoder(rr1.Body).Decode(&resp1)
	if resp1.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected CodeInvalidRequest, got %d", resp1.Code)
	}

	// 3. Handler 正常查询
	rr2 := httptest.NewRecorder()
	reqGood := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"group_id":"`+res.Group.GroupID+`"}`))
	reqGood = reqGood.WithContext(ctxAlice)
	handler.ListAdminMembers(rr2, reqGood)
	var resp2 httpx.ApiResponse[PaginatedAdminGroupMembers]
	_ = json.NewDecoder(rr2.Body).Decode(&resp2)
	if resp2.Code != 0 || resp2.Data == nil || len(resp2.Data.Items) == 0 {
		t.Fatalf("expected admin members returned, got %+v", resp2)
	}
}

func TestServiceUserGroupAggregationDelegation(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)

	// ListGroupsByUser 委托调用验证
	pGroups, err := svc.ListGroupsByUser(ctx, "usr_alice", 20, "")
	if err != nil {
		t.Fatalf("ListGroupsByUser failed: %v", err)
	}
	_ = pGroups

	// CountCommonGroups 委托调用验证
	commonCount, err := svc.CountCommonGroups(ctx, "usr_alice", "usr_bob")
	if err != nil {
		t.Fatalf("CountCommonGroups failed: %v", err)
	}
	if commonCount != 0 {
		t.Fatalf("expected 0 common groups, got %d", commonCount)
	}
}

func TestInMemoryRepositoryMemberManagement(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()

	// 1. 创建群
	createRes, err := repo.Create(ctx, "usr_owner", CreateGroupRequest{
		Name: "成员管理测试群",
	})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}
	groupID := createRes.Group.GroupID

	// 2. 加入成员 bob
	repo.members[groupID]["usr_bob"] = inMemoryMembership{
		Role:     GroupRoleMember,
		JoinedAt: "2026-01-01T00:00:00Z",
	}

	// 3. 搜索成员 SearchMembers
	members, cursor, err := repo.SearchMembers(ctx, groupID, "usr_bob", nil, 10, "")
	if err != nil || len(members) == 0 {
		t.Fatalf("SearchMembers failed: %v, members=%+v", err, members)
	}
	_ = cursor

	// 4. 禁言与解禁 UnmuteMember
	err = repo.MuteMember(ctx, "usr_owner", MuteMemberRequest{
		GroupID:         groupID,
		UserID:          "usr_bob",
		DurationMinutes: 60,
	})
	if err != nil {
		t.Fatalf("MuteMember failed: %v", err)
	}
	err = repo.UnmuteMember(ctx, "usr_owner", UnmuteMemberRequest{
		GroupID: groupID,
		UserID:  "usr_bob",
	})
	if err != nil {
		t.Fatalf("UnmuteMember failed: %v", err)
	}

	// 5. 封禁 BanMember、列表 ListBannedMembers、解封 UnbanMember
	reason := "违规发言"
	err = repo.BanMember(ctx, "usr_owner", BanMemberRequest{
		GroupID: groupID,
		UserID:  "usr_bob",
		Reason:  &reason,
	})
	if err != nil {
		t.Fatalf("BanMember failed: %v", err)
	}
	bannedList, _, err := repo.ListBannedMembers(ctx, groupID, 10, "")
	if err != nil || len(bannedList) == 0 {
		t.Fatalf("ListBannedMembers failed: %v, len=%d", err, len(bannedList))
	}
	err = repo.UnbanMember(ctx, "usr_owner", UnbanMemberRequest{
		GroupID: groupID,
		UserID:  "usr_bob",
	})
	if err != nil {
		t.Fatalf("UnbanMember failed: %v", err)
	}

	// 6. 重新加入 bob，测试所有权转让 TransferOwnership
	repo.members[groupID]["usr_bob"] = inMemoryMembership{
		Role:     GroupRoleMember,
		JoinedAt: "2026-01-01T00:00:00Z",
	}
	err = repo.TransferOwnership(ctx, "usr_owner", TransferOwnershipRequest{
		GroupID:    groupID,
		NewOwnerID: "usr_bob",
	})
	if err != nil {
		t.Fatalf("TransferOwnership failed: %v", err)
	}

	// 7. 踢出成员 RemoveMember
	err = repo.RemoveMember(ctx, "usr_bob", RemoveMemberRequest{
		GroupID: groupID,
		UserID:  "usr_owner",
	})
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}
}
