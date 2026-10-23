package routemeta

import "strings"

// Audit 路由元数据一致性审计结果。
type Audit struct {
	// Unclassified 已挂载但限流分类未覆盖的路由（会落默认兜底桶；新增端点
	// 若属写端点将被漏判）。
	Unclassified []string
	// DeadGuardRules 声明了但没有任何已挂载路由与之匹配的守卫规则
	// （历史踩坑：/groups/update、/groups/dismiss 因路由实际在 /groups/admin
	// 子树下而永不命中，属死配置）。
	DeadGuardRules []string
}

// OK 审计是否无问题。
func (a Audit) OK() bool { return len(a.Unclassified) == 0 && len(a.DeadGuardRules) == 0 }

// AuditRoutes 对已挂载路由做双向一致性校验：
//  1. 业务路由（/api/v1/、/admin/api/v1/、/public/v1/、/api/v1/system 等）必须
//     被限流分类覆盖；
//  2. 每条守卫规则必须至少命中一条已挂载路由。
//
// routes 为 chi 路由枚举后的完整路径（如 /api/v1/moments/create）。
func AuditRoutes(routes []string) Audit {
	var audit Audit
	for _, route := range routes {
		if !isClassifiablePath(route) {
			continue
		}
		if _, ok := ClassifyPath(route); !ok {
			audit.Unclassified = append(audit.Unclassified, route)
		}
	}
	for _, rule := range guardRules {
		matched := false
		for _, route := range routes {
			if strings.HasSuffix(route, rule.Suffix) {
				matched = true
				break
			}
		}
		if !matched {
			audit.DeadGuardRules = append(audit.DeadGuardRules, rule.Suffix)
		}
	}
	return audit
}

// isClassifiablePath 只审计走限流中间件的业务路由；/metrics、/healthz、
// /share/{code}（爬虫落地页）等登记例外不参与。
func isClassifiablePath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/api/v1/"), strings.HasPrefix(path, "/admin/api/v1/"), strings.HasPrefix(path, "/public/v1/"):
		return true
	default:
		return false
	}
}
