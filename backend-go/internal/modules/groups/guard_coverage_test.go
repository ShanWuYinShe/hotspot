package groups

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/rbac"
)

func TestGuardGroupIDExtraction(t *testing.T) {
	id, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"group_id":"g1"}`)))
	if !ok || id != "g1" {
		t.Fatalf("normal: ok=%v id=%q", ok, id)
	}
	// 空格裁剪
	id, ok = guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"group_id":"  g2  "}`)))
	if !ok || id != "g2" {
		t.Fatalf("trim: ok=%v id=%q", ok, id)
	}
	// 空 body
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", nil)); ok {
		t.Fatal("nil body must fail")
	}
	// 超长 body
	big := bytes.NewReader(make([]byte, maxGuardBodyBytes+1))
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", big)); ok {
		t.Fatal("oversized body must fail")
	}
	// 非 JSON
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("not-json"))); ok {
		t.Fatal("invalid json must fail")
	}
	// 缺 group_id 字段
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"other":1}`))); ok {
		t.Fatal("missing group_id must fail")
	}
	// group_id 非字符串
	if _, ok := guardGroupID(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"group_id":123}`))); ok {
		t.Fatal("numeric group_id must fail")
	}
	// 读取错误（不可读 body）
	req := httptest.NewRequest(http.MethodPost, "/x", io.NopCloser(errReader{}))
	if _, ok := guardGroupID(req); ok {
		t.Fatal("unreadable body must fail")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return parsed
}

func TestServiceCoverageHelpers(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	_ = ctx

	// setter 注入不 panic
	SetBlockedTagChecker(nil)
	svc.SetGroupDismissAuditWriter(nil)
	svc.SetLocationProvider(nil)

	// ownerIDValue 空指针 → ""
	if v := ownerIDValue(nil); v != "" {
		t.Fatalf("nil owner: %q", v)
	}
	owner := "u1"
	if v := ownerIDValue(&owner); v != "u1" {
		t.Fatalf("owner: %q", v)
	}

	// derefString
	if v := derefString(nil); v != "" {
		t.Fatalf("nil deref: %q", v)
	}
	s := "x"
	if v := derefString(&s); v != "x" {
		t.Fatalf("deref: %q", v)
	}

	// groupUserInfo：avatar 为空/非空
	info := groupUserInfo(UserInfo{UserID: "u1", Nickname: "n1"})
	if info["user_id"] != "u1" {
		t.Fatalf("user info: %#v", info)
	}
	avatar := "http://a"
	info = groupUserInfo(UserInfo{UserID: "u2", Nickname: "n2", AvatarURL: &avatar})
	if info["avatar_url"] != "http://a" {
		t.Fatalf("user info avatar: %#v", info)
	}
	emptyAvatar := ""
	info = groupUserInfo(UserInfo{UserID: "u3", Nickname: "n3", AvatarURL: &emptyAvatar})
	if _, has := info["avatar_url"]; has {
		t.Fatalf("empty avatar must be omitted: %#v", info)
	}
}

func TestMatchesSearchCursor(t *testing.T) {
	item := GroupItem{Group: GroupInfo{GroupID: "g1", MembersCount: 10}}
	meta := groupSearchMeta{Distance: 100, CreatedAt: mustParseTime(t, "2025-01-01T00:00:00Z")}
	// distance：不同距离比大小，相同距离比 ID
	cur := groupSearchCursor{SortBy: searchSortDistance, Distance: 50}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("distance greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortDistance, Distance: 100, GroupID: "g0"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("distance equal id greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortDistance, Distance: 100, GroupID: "g2"}
	if matchesSearchCursor(meta, item, cur) {
		t.Fatal("distance equal id lower must not match")
	}

	// newest：时间不同比时间，相同比 ID
	cur = groupSearchCursor{SortBy: searchSortNewest, CreatedAt: "2025-01-02T00:00:00Z"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("older must match")
	}
	cur = groupSearchCursor{SortBy: searchSortNewest, CreatedAt: "2025-01-01T00:00:00Z", GroupID: "g0"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("same time id greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortNewest, CreatedAt: "2025-01-01T00:00:00Z", GroupID: "g2"}
	if matchesSearchCursor(meta, item, cur) {
		t.Fatal("same time id lower must not match")
	}

	// popular：人数不同比人数，相同比 ID
	cur = groupSearchCursor{SortBy: searchSortPopular, MembersCount: 20}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("fewer members must match")
	}
	cur = groupSearchCursor{SortBy: searchSortPopular, MembersCount: 10, GroupID: "g0"}
	if !matchesSearchCursor(meta, item, cur) {
		t.Fatal("same count id greater must match")
	}
	cur = groupSearchCursor{SortBy: searchSortPopular, MembersCount: 10, GroupID: "g2"}
	if matchesSearchCursor(meta, item, cur) {
		t.Fatal("same count id lower must not match")
	}
}

type mockRoleResolver struct {
	roleFunc func(ctx context.Context, groupID string) (rbac.GroupRole, bool, error)
}

func (m mockRoleResolver) RoleOf(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
	if m.roleFunc != nil {
		return m.roleFunc(ctx, groupID)
	}
	return "", false, nil
}

func TestGroupAdminGuardMiddleware(t *testing.T) {
	orig := serviceGroupRoleResolver
	defer func() { SetGroupRoleResolver(orig) }()

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	// 1. 请求体缺少或错误的 group_id
	t.Run("Invalid Body Without Group ID", func(t *testing.T) {
		guard := groupAdminGuard(false)(nextHandler)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/update", strings.NewReader(`{}`))
		guard.ServeHTTP(rr, req)

		var resp httpx.ApiResponse[struct{}]
		_ = json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("expected 10000 invalid request, got %d", resp.Code)
		}
	})

	// 2. resolver 未注入 (fail-closed -> 10600)
	t.Run("Resolver Not Configured Fail-Closed", func(t *testing.T) {
		SetGroupRoleResolver(nil)
		guard := groupAdminGuard(false)(nextHandler)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/update", strings.NewReader(`{"group_id":"g_1"}`))
		guard.ServeHTTP(rr, req)

		var resp httpx.ApiResponse[struct{}]
		_ = json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Code != httpx.CodeInternalUnavailable {
			t.Fatalf("expected 10600 when resolver nil, got %d", resp.Code)
		}
	})

	// 3. resolver 返回 error (fail-closed -> 10600)
	t.Run("Resolver Error Fail-Closed", func(t *testing.T) {
		SetGroupRoleResolver(mockRoleResolver{
			roleFunc: func(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
				return "", false, errors.New("db error")
			},
		})
		guard := groupAdminGuard(false)(nextHandler)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/update", strings.NewReader(`{"group_id":"g_1"}`))
		guard.ServeHTTP(rr, req)

		var resp httpx.ApiResponse[struct{}]
		_ = json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Code != httpx.CodeInternalUnavailable {
			t.Fatalf("expected 10600 on resolver error, got %d", resp.Code)
		}
	})

	// 4. 非群成员 -> 10200 + ReasonNotGroupMember
	t.Run("Not Group Member", func(t *testing.T) {
		SetGroupRoleResolver(mockRoleResolver{
			roleFunc: func(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
				return "", false, nil
			},
		})
		guard := groupAdminGuard(false)(nextHandler)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/update", strings.NewReader(`{"group_id":"g_1"}`))
		guard.ServeHTTP(rr, req)

		var resp httpx.ApiResponse[httpx.ForbiddenErrorData]
		_ = json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Code != httpx.CodeForbidden || resp.Data == nil || resp.Data.ForbiddenReason != httpx.ReasonNotGroupMember {
			t.Fatalf("expected ReasonNotGroupMember, got %+v", resp)
		}
	})

	// 5. 普通群成员，非管理员 -> 10200 + ReasonNotGroupAdmin
	t.Run("Member Not Admin", func(t *testing.T) {
		SetGroupRoleResolver(mockRoleResolver{
			roleFunc: func(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
				return rbac.GroupRoleMember, true, nil
			},
		})
		guard := groupAdminGuard(false)(nextHandler)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/update", strings.NewReader(`{"group_id":"g_1"}`))
		guard.ServeHTTP(rr, req)

		var resp httpx.ApiResponse[httpx.ForbiddenErrorData]
		_ = json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Code != httpx.CodeForbidden || resp.Data == nil || resp.Data.ForbiddenReason != httpx.ReasonNotGroupAdmin {
			t.Fatalf("expected ReasonNotGroupAdmin, got %+v", resp)
		}
	})

	// 6. 管理员访问 ownerOnly 端点（如解散群） -> 10200 + ReasonNotGroupOwner
	t.Run("Admin On OwnerOnly Endpoint", func(t *testing.T) {
		SetGroupRoleResolver(mockRoleResolver{
			roleFunc: func(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
				return rbac.GroupRoleAdmin, true, nil
			},
		})
		guard := groupAdminGuard(true)(nextHandler) // ownerOnly = true
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/dismiss", strings.NewReader(`{"group_id":"g_1"}`))
		guard.ServeHTTP(rr, req)

		var resp httpx.ApiResponse[httpx.ForbiddenErrorData]
		_ = json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Code != httpx.CodeForbidden || resp.Data == nil || resp.Data.ForbiddenReason != httpx.ReasonNotGroupOwner {
			t.Fatalf("expected ReasonNotGroupOwner, got %+v", resp)
		}
	})

	// 7. 管理员正常访问普通管理端点（ownerOnly=false） -> 放行
	t.Run("Admin On Admin Endpoint Allowed", func(t *testing.T) {
		SetGroupRoleResolver(mockRoleResolver{
			roleFunc: func(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
				return rbac.GroupRoleAdmin, true, nil
			},
		})
		nextCalled = false
		guard := groupAdminGuard(false)(nextHandler)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(`{"group_id":"g_1"}`))
		guard.ServeHTTP(rr, req)

		if !nextCalled || rr.Code != http.StatusNoContent {
			t.Fatalf("expected next handler called with 204, got %d", rr.Code)
		}
	})

	// 8. 群主访问 ownerOnly 端点 -> 放行
	t.Run("Owner On OwnerOnly Endpoint Allowed", func(t *testing.T) {
		SetGroupRoleResolver(mockRoleResolver{
			roleFunc: func(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
				return rbac.GroupRoleOwner, true, nil
			},
		})
		nextCalled = false
		guard := groupAdminGuard(true)(nextHandler)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/dismiss", strings.NewReader(`{"group_id":"g_1"}`))
		guard.ServeHTTP(rr, req)

		if !nextCalled || rr.Code != http.StatusNoContent {
			t.Fatalf("expected next handler called with 204, got %d", rr.Code)
		}
	})
}
