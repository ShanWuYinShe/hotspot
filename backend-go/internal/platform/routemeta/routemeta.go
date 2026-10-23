// Package routemeta 是路由级元数据的唯一来源（审查 P2-架构：此前限流读/写
// 分类（ratelimit.classifyRoute）与资源守卫规则（resourceguard.guardRules）
// 是两份互不相干的硬编码清单，新增端点时极易只改一处——历史上已出现
// /groups/update、/groups/dismiss 这类永不匹配的死规则，且限流分类漂移
// 属于安全影响（写端点被当读端点就跳过写限流桶）。
//
// 因此把两类元数据集中在本包声明：
//   - 限流分类：ClassifyPath（只读/管理端点分类）
//   - 资源守卫：GuardRules / MatchGuard（请求体资源 ID 存在性预检）
//
// 消费方（ratelimit、resourceguard）只做薄适配；路由与元数据的一致性由
// AuditRoutes 在启动时（bootstrap，Warn 日志）与 CI 测试
// （bootstrap/route_metadata_test.go）双向校验：
//   - 已挂载路由必须被分类覆盖（防止新端点漏限流分类）
//   - 守卫规则必须命中至少一条已挂载路由（防止死配置漂移）
package routemeta

import "strings"

// ResourceKind 资源类型（同时被 resourceguard.Checker 使用；resourceguard 以
// 类型别名 re-export，避免两处定义漂移）。
type ResourceKind string

const (
	ResourceUser   ResourceKind = "user"
	ResourceGroup  ResourceKind = "group"
	ResourceMoment ResourceKind = "moment"
)

// GuardRef 一条资源守卫引用：从请求体某字段取资源 ID 做存在性预检。
type GuardRef struct {
	Field string
	Kind  ResourceKind
}

// GuardRule 按路径后缀匹配的守卫规则。
type GuardRule struct {
	Suffix string
	Refs   []GuardRef
}

// guardRules 资源守卫规则表（原 resourceguard.guardRules，逐条迁移，语义不变）。
var guardRules = []GuardRule{
	{Suffix: "/api/v1/users/open_the_door", Refs: []GuardRef{{Field: "user_id", Kind: ResourceUser}}},
	{Suffix: "/api/v1/users/fetch_groups", Refs: []GuardRef{{Field: "user_id", Kind: ResourceUser}}},
	{Suffix: "/api/v1/users/fetch_moments", Refs: []GuardRef{{Field: "user_id", Kind: ResourceUser}}},
	{Suffix: "/api/v1/users/fetch_reacted_moments", Refs: []GuardRef{{Field: "user_id", Kind: ResourceUser}}},
	{Suffix: "/api/v1/groups/open_the_door", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/members/fetch", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/members/search", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	// 群管理端点实际挂在 /groups/admin/* 子树（modules/groups/module.go）；
	// 旧规则 /groups/update、/groups/dismiss 从未匹配（HasSuffix 不命中，死配置）。
	// 全量对齐 admin 子树携带 group_id 的端点：资源存在性预检 + 404 语义统一。
	{Suffix: "/api/v1/groups/admin/update", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/settings", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/invites/create", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/invites/fetch_sent", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/invites/revoke", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/members/fetch", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/members/set_role", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/members/set_mute", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/members/set_ban", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/members/fetch_banned", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/members/remove", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/applications/fetch", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/applications/approve", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/applications/reject", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/announcements/create", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/announcements/update", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/announcements/delete", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/messages/recall", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/dismiss", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/admin/transfer_ownership", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/groups/enter", Refs: []GuardRef{{Field: "group_id", Kind: ResourceGroup}}},
	{Suffix: "/api/v1/moments/open_the_door", Refs: []GuardRef{{Field: "moment_id", Kind: ResourceMoment}}},
	{Suffix: "/api/v1/moments/update", Refs: []GuardRef{{Field: "moment_id", Kind: ResourceMoment}}},
	{Suffix: "/api/v1/moments/delete", Refs: []GuardRef{{Field: "moment_id", Kind: ResourceMoment}}},
	{Suffix: "/api/v1/moments/set_favorite", Refs: []GuardRef{{Field: "moment_id", Kind: ResourceMoment}}},
	{Suffix: "/api/v1/comments/fetch", Refs: []GuardRef{{Field: "moment_id", Kind: ResourceMoment}}},
	{Suffix: "/api/v1/comments/create", Refs: []GuardRef{{Field: "moment_id", Kind: ResourceMoment}}},
}

// GuardRules 返回守卫规则表（只读用途；调用方不得修改返回切片）。
func GuardRules() []GuardRule { return guardRules }

// MatchGuard 按路径后缀匹配守卫规则（与原 resourceguard.matchGuardRule 同口径：
// 按声明顺序取首个 Suffix 命中项）。
func MatchGuard(path string) (GuardRule, bool) {
	for _, rule := range guardRules {
		if strings.HasSuffix(path, rule.Suffix) {
			return rule, true
		}
	}
	return GuardRule{}, false
}
