package routemeta

import "strings"

// RouteClass 限流分类（原 ratelimit.routeClass，语义逐条迁移）。
//
// AdminOnly 表示管理后台端点（独立的管理维度限流桶）；ReadOnly 表示只读端点
// （走读限流桶，跳写桶）。未识别路径返回 ok=false，由调用方决定兜底策略。
type RouteClass struct {
	ReadOnly  bool
	AdminOnly bool
}

// ClassifyPath 按路径前缀/后缀分类限流桶（原 ratelimit.classifyRoute）。
// 行为与迁移前完全一致：同一路径必须得到相同的 (ReadOnly, AdminOnly)。
func ClassifyPath(path string) (RouteClass, bool) {
	switch {
	case strings.HasPrefix(path, "/public/v1/"):
		// 匿名节点（spec api_spec_public.md）：认证入口在 appendAuthRules 已按 IP
		// 处理并提前返回；其余端点全部只读，按 IP 维度限流（无 principal）。
		return RouteClass{ReadOnly: true}, true
	case strings.HasPrefix(path, "/share/"):
		// 分享链接爬虫落地页（GET /share/{code}，登记例外见 share/og.go）：匿名只读。
		// 归入读分类后由 ratelimit 统一按 IP 计量（见 ratelimit 的 probe 桶），
		// 避免随机 share code 枚举放大数据库查询。
		return RouteClass{ReadOnly: true}, true
	case strings.HasPrefix(path, "/admin/api/v1/"):
		return RouteClass{AdminOnly: true, ReadOnly: isAdminReadPath(path)}, true
	case strings.HasPrefix(path, "/api/v1/system/"):
		return RouteClass{ReadOnly: true}, true
	case strings.HasPrefix(path, "/api/v1/search/"):
		return RouteClass{ReadOnly: true}, true
	case strings.HasPrefix(path, "/api/v1/auth/"):
		return RouteClass{ReadOnly: isAuthReadPath(path)}, true
	case strings.HasPrefix(path, "/api/v1/media/"):
		return RouteClass{ReadOnly: false}, true
	case strings.HasPrefix(path, "/api/v1/reactions/"):
		// 审计补漏（2026-10-24 路由一致性审计首次运行即发现）：reactions
		// 模块此前从未出现在限流分类里，classifyRules 对未识别路径返回 nil
		// 规则集——/api/v1/reactions/fetch 与 /set 实际完全没有限流。
		return RouteClass{ReadOnly: strings.HasSuffix(path, "/fetch")}, true
	case strings.HasPrefix(path, "/api/v1/reports/"):
		return RouteClass{ReadOnly: strings.HasSuffix(path, "/fetch")}, true
	case strings.HasPrefix(path, "/api/v1/moments/"):
		return RouteClass{ReadOnly: !strings.HasSuffix(path, "/create")}, true
	case strings.HasPrefix(path, "/api/v1/comments/"):
		return RouteClass{ReadOnly: !strings.HasSuffix(path, "/create")}, true
	case strings.HasPrefix(path, "/api/v1/groups/"):
		return RouteClass{ReadOnly: strings.HasSuffix(path, "/search") || strings.HasSuffix(path, "/open_the_door") || strings.HasSuffix(path, "/fetch_joined") || strings.HasSuffix(path, "/fetch_submitted") || strings.HasSuffix(path, "/fetch_received") || strings.HasSuffix(path, "/fetch_sent") || strings.HasSuffix(path, "/fetch_banned")}, true
	case strings.HasPrefix(path, "/api/v1/users/"):
		return RouteClass{ReadOnly: isUsersReadPath(path)}, true
	case strings.HasPrefix(path, "/api/v1/private_chats/"):
		return RouteClass{ReadOnly: true}, true
	case strings.HasPrefix(path, "/api/v1/private_messages/"):
		return RouteClass{ReadOnly: strings.HasSuffix(path, "/fetch")}, true
	case strings.HasPrefix(path, "/api/v1/group_messages/"):
		return RouteClass{ReadOnly: strings.HasSuffix(path, "/fetch")}, true
	case strings.HasPrefix(path, "/api/v1/notify/"):
		return RouteClass{ReadOnly: strings.HasSuffix(path, "/fetch") || strings.HasSuffix(path, "/unread_summary")}, true
	case strings.HasPrefix(path, "/api/v1/share/"):
		return RouteClass{ReadOnly: strings.HasSuffix(path, "/fetch_created")}, true
	case strings.HasPrefix(path, "/api/v1/map/"):
		return RouteClass{ReadOnly: true}, true
	default:
		return RouteClass{}, false
	}
}

func isAuthReadPath(path string) bool {
	switch path {
	case "/api/v1/auth/session", "/api/v1/auth/fetch_sessions":
		return true
	default:
		return false
	}
}

func isAdminReadPath(path string) bool {
	return strings.Contains(path, "/get") || strings.Contains(path, "/list") || strings.Contains(path, "/search") || strings.Contains(path, "/open_the_door")
}

func isUsersReadPath(path string) bool {
	switch path {
	case "/api/v1/users/search", "/api/v1/users/open_the_door", "/api/v1/users/account", "/api/v1/users/settings", "/api/v1/users/fetch_blocked":
		return true
	default:
		return false
	}
}
