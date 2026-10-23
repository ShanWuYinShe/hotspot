package routemeta

import "testing"

// TestClassifyPath 迁移自 ratelimit 分类逻辑的行为契约：同一路径必须得到
// 与迁移前完全一致的 (ReadOnly, AdminOnly)。这些都是限流桶选择依据，
// 分类漂移等于写端点跑进读桶（安全影响），故逐条固化。
func TestClassifyPath(t *testing.T) {
	cases := []struct {
		path      string
		readOnly  bool
		adminOnly bool
		known     bool
	}{
		{"/public/v1/auth/guest", true, false, true},
		{"/public/v1/system/announcements/fetch", true, false, true},
		{"/admin/api/v1/users/search", true, true, true},
		{"/admin/api/v1/users/set_ban", false, true, true},
		{"/admin/api/v1/reports/open_the_door", true, true, true},
		{"/api/v1/system/health", true, false, true},
		{"/api/v1/search/suggest", true, false, true},
		{"/api/v1/auth/session", true, false, true},
		{"/api/v1/auth/fetch_sessions", true, false, true},
		{"/api/v1/auth/register", false, false, true},
		{"/api/v1/media/presign_upload", false, false, true},
		{"/api/v1/reports/fetch", true, false, true},
		{"/api/v1/reports/create", false, false, true},
		{"/api/v1/moments/fetch_feed", true, false, true},
		{"/api/v1/moments/create", false, false, true},
		{"/api/v1/comments/fetch", true, false, true},
		{"/api/v1/comments/create", false, false, true},
		{"/api/v1/groups/search", true, false, true},
		{"/api/v1/groups/open_the_door", true, false, true},
		{"/api/v1/groups/fetch_joined", true, false, true},
		{"/api/v1/groups/admin/create", false, false, true},
		{"/api/v1/users/search", true, false, true},
		{"/api/v1/users/account", true, false, true},
		{"/api/v1/users/update_profile", false, false, true},
		{"/api/v1/private_chats/fetch", true, false, true},
		{"/api/v1/private_messages/fetch", true, false, true},
		{"/api/v1/private_messages/send", false, false, true},
		{"/api/v1/group_messages/fetch", true, false, true},
		{"/api/v1/group_messages/send", false, false, true},
		{"/api/v1/notify/fetch", true, false, true},
		{"/api/v1/notify/unread_summary", true, false, true},
		{"/api/v1/notify/mark_read", false, false, true},
		{"/api/v1/share/fetch_created", true, false, true},
		{"/api/v1/share/create", false, false, true},
		{"/api/v1/map/markers", true, false, true},
		// 审计补漏项：此前未分类 → 无限流（429 桶缺失）
		{"/api/v1/reactions/fetch", true, false, true},
		{"/api/v1/reactions/set", false, false, true},
		{"/healthz", false, false, false},
		{"/metrics", false, false, false},
	}
	for _, tc := range cases {
		got, ok := ClassifyPath(tc.path)
		if ok != tc.known {
			t.Fatalf("%s: known=%v want %v", tc.path, ok, tc.known)
		}
		if ok && (got.ReadOnly != tc.readOnly || got.AdminOnly != tc.adminOnly) {
			t.Fatalf("%s: class=%+v want readOnly=%v adminOnly=%v", tc.path, got, tc.readOnly, tc.adminOnly)
		}
	}
}

func TestMatchGuard(t *testing.T) {
	cases := []struct {
		path  string
		field string
		kind  ResourceKind
		ok    bool
	}{
		{"/api/v1/groups/admin/members/set_role", "group_id", ResourceGroup, true},
		{"/api/v1/moments/open_the_door", "moment_id", ResourceMoment, true},
		{"/api/v1/users/fetch_moments", "user_id", ResourceUser, true},
		{"/api/v1/groups/create", "", "", false},
		{"/api/v1/auth/session", "", "", false},
	}
	for _, tc := range cases {
		rule, ok := MatchGuard(tc.path)
		if ok != tc.ok {
			t.Fatalf("%s: ok=%v want %v", tc.path, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if len(rule.Refs) != 1 || rule.Refs[0].Field != tc.field || rule.Refs[0].Kind != tc.kind {
			t.Fatalf("%s: refs=%+v want field=%s kind=%s", tc.path, rule.Refs, tc.field, tc.kind)
		}
	}
}

func TestAuditRoutes(t *testing.T) {
	// 覆盖齐全 + 死规则：用最小路由集合验证两个方向都能报出来。
	audit := AuditRoutes([]string{"/api/v1/moments/create", "/healthz", "/api/v1/unknown/future_endpoint"})
	if len(audit.Unclassified) != 1 || audit.Unclassified[0] != "/api/v1/unknown/future_endpoint" {
		t.Fatalf("unexpected unclassified: %v", audit.Unclassified)
	}
	if len(audit.DeadGuardRules) == 0 {
		t.Fatal("expected dead guard rules reported")
	}
	if audit.OK() {
		t.Fatal("audit must not be OK")
	}
	// 全量规则命中时应无审计问题。
	var routes []string
	for _, rule := range GuardRules() {
		routes = append(routes, rule.Suffix)
	}
	clean := AuditRoutes(routes)
	if !clean.OK() {
		t.Fatalf("expected clean audit, got %+v", clean)
	}
}

func TestGuardRulesAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(GuardRules()))
	for _, rule := range GuardRules() {
		if seen[rule.Suffix] {
			t.Fatalf("duplicate guard rule suffix: %s", rule.Suffix)
		}
		seen[rule.Suffix] = true
		if len(rule.Refs) == 0 {
			t.Fatalf("guard rule without refs: %s", rule.Suffix)
		}
	}
	if len(seen) != len(GuardRules()) {
		t.Fatalf("rules=%d unique=%d", len(GuardRules()), len(seen))
	}
}
