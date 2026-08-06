package admin

import (
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/httpx"
)

const (
	permissionRbacManage         = "admin.rbac.manage"
	permissionAppConfigRead      = "admin.app_config.read"
	permissionAppConfigWrite     = "admin.app_config.write"
	permissionAuditRead          = "admin.audit.read"
	permissionAuditReadIP        = "admin.audit.read_ip"
	permissionTagsRead           = "admin.tags.read"
	permissionTagsWrite          = "admin.tags.write"
	permissionAnnouncementsRead  = "admin.announcements.read"
	permissionAnnouncementsWrite = "admin.announcements.write"
	permissionUsersRead          = "admin.users.read"
	permissionUsersWrite         = "admin.users.write"
	permissionGroupsRead         = "admin.groups.read"
	permissionGroupsWrite        = "admin.groups.write"
	permissionReportsRead        = "admin.reports.read"
	permissionReportsWrite       = "admin.reports.write"
	wildcardPermission           = "*"
)

var permissionDictionary = []AdminPermissionItem{
	{Key: permissionRbacManage, Description: "管理员权限管理（授权/角色/权限）"},
	{Key: "admin.reports.read", Description: "查看举报列表"},
	{Key: "admin.reports.write", Description: "处理举报"},
	{Key: permissionAppConfigRead, Description: "查看应用配置（AdminAppConfig）"},
	{Key: permissionAppConfigWrite, Description: "修改应用配置（AdminAppConfig）"},
	{Key: permissionTagsRead, Description: "查看标签列表"},
	{Key: permissionTagsWrite, Description: "删除/屏蔽标签"},
	{Key: permissionAnnouncementsRead, Description: "查看公告列表"},
	{Key: permissionAnnouncementsWrite, Description: "发布/删除公告"},
	{Key: "admin.users.read", Description: "查看用户列表/详情"},
	{Key: "admin.users.write", Description: "封禁/解封用户"},
	{Key: "admin.groups.read", Description: "查看群组列表/详情"},
	{Key: "admin.groups.write", Description: "解散群组"},
	{Key: permissionAuditRead, Description: "查看审计日志（不含 ip_address 字段）"},
	{Key: permissionAuditReadIP, Description: "查看审计日志 IP 地址"},
}

type AdminGrantUpsert struct {
	UserID              string
	GrantAllPermissions bool
	RoleIDs             []string
	GrantedAt           time.Time
}

func normalizeRoleName(name string) string {
	return strings.TrimSpace(name)
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := append([]string(nil), values...)
	return cloned
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortAndUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := set[value]; exists {
			continue
		}
		set[value] = struct{}{}
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func validPermissionSet(values []string) bool {
	if len(values) == 0 {
		return true
	}
	allowed := make(map[string]struct{}, len(permissionDictionary)+1)
	allowed[wildcardPermission] = struct{}{}
	for _, item := range permissionDictionary {
		allowed[item.Key] = struct{}{}
	}
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return false
		}
	}
	return true
}

func limitPageItems[T any](items []T, limit int) ([]T, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

// admin 各列表游标的 AES-GCM 加密命名空间（cursor 包：同 secret 下命名空间
// 隔离，跨列表游标无法互用）。
const (
	adminCursorNamespace       = "admin.admins"
	auditCursorNamespace       = "admin.audit_logs"
	adminUserCursorNamespace   = "admin.users"
	adminGroupCursorNamespace  = "admin.groups"
	adminReportCursorNamespace = "admin.reports"
)

func encodeCursor[T any](namespace string, c T) string {
	value, err := cursor.Encode(namespace, c)
	if err != nil {
		return ""
	}
	return value
}

func decodeCursor[T any](namespace, raw string) (T, *httpx.APIError) {
	var out T
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	if err := cursor.Decode(namespace, raw, &out); err != nil {
		return out, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return out, nil
}

func encodeAdminCursor(c adminCursor) string {
	return encodeCursor(adminCursorNamespace, c)
}

func decodeAdminCursor(raw string) (adminCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return adminCursor{}, nil
	}
	c, apiErr := decodeCursor[adminCursor](adminCursorNamespace, raw)
	if apiErr != nil {
		return c, apiErr
	}
	if c.UserID == "" || c.GrantedAt.IsZero() {
		return adminCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return c, nil
}

func encodeAuditCursor(c auditCursor) string {
	return encodeCursor(auditCursorNamespace, c)
}

func decodeAuditCursor(raw string) (auditCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return auditCursor{}, nil
	}
	c, apiErr := decodeCursor[auditCursor](auditCursorNamespace, raw)
	if apiErr != nil {
		return c, apiErr
	}
	if c.LogID == "" || c.CreatedAt.IsZero() {
		return auditCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return c, nil
}

func encodeAdminUserCursor(c adminUserCursor) string {
	return encodeCursor(adminUserCursorNamespace, c)
}

func decodeAdminUserCursor(raw string) (adminUserCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return adminUserCursor{}, nil
	}
	c, apiErr := decodeCursor[adminUserCursor](adminUserCursorNamespace, raw)
	if apiErr != nil {
		return c, apiErr
	}
	if c.UserID == "" || c.CreatedAt.IsZero() {
		return adminUserCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return c, nil
}

func encodeAdminGroupCursor(c adminGroupCursor) string {
	return encodeCursor(adminGroupCursorNamespace, c)
}

func decodeAdminGroupCursor(raw string) (adminGroupCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return adminGroupCursor{}, nil
	}
	c, apiErr := decodeCursor[adminGroupCursor](adminGroupCursorNamespace, raw)
	if apiErr != nil {
		return c, apiErr
	}
	if c.GroupID == "" || c.CreatedAt.IsZero() {
		return adminGroupCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return c, nil
}

func encodeAdminReportCursor(c adminReportCursor) string {
	return encodeCursor(adminReportCursorNamespace, c)
}

func decodeAdminReportCursor(raw string) (adminReportCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return adminReportCursor{}, nil
	}
	c, apiErr := decodeCursor[adminReportCursor](adminReportCursorNamespace, raw)
	if apiErr != nil {
		return c, apiErr
	}
	if c.ReportID == "" || c.CreatedAt.IsZero() {
		return adminReportCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return c, nil
}
