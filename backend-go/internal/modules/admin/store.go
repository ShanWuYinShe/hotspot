package admin

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

const (
	permissionRbacManage    = "admin.rbac.manage"
	permissionAppConfigRead = "admin.app_config.read"
	wildcardPermission      = "*"
)

var permissionDictionary = []AdminPermissionItem{
	{Key: permissionRbacManage, Description: "管理员权限管理（授权/角色/权限）"},
	{Key: "admin.reports.read", Description: "查看举报列表"},
	{Key: "admin.reports.write", Description: "处理举报"},
	{Key: permissionAppConfigRead, Description: "查看应用配置（AppConfig）"},
	{Key: "admin.app_config.write", Description: "修改应用配置（AppConfig）"},
	{Key: "admin.tags.read", Description: "查看标签列表"},
	{Key: "admin.tags.write", Description: "删除/屏蔽标签"},
	{Key: "admin.announcements.read", Description: "查看公告列表"},
	{Key: "admin.announcements.write", Description: "发布/删除公告"},
	{Key: "admin.users.read", Description: "查看用户列表/详情"},
	{Key: "admin.users.write", Description: "封禁/解封用户"},
	{Key: "admin.groups.read", Description: "查看群组列表/详情"},
	{Key: "admin.groups.write", Description: "解散群组"},
	{Key: "admin.audit.read", Description: "查看审计日志（不含 ip_address 字段）"},
	{Key: "admin.audit.read_ip", Description: "查看审计日志 IP 地址"},
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

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 100:
		return 100
	default:
		return limit
	}
}

func encodeAdminCursor(cursor adminCursor) string {
	blob, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func decodeAdminCursor(raw string) (adminCursor, *httpx.APIError) {
	if strings.TrimSpace(raw) == "" {
		return adminCursor{}, nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return adminCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	var cursor adminCursor
	if err := json.Unmarshal(blob, &cursor); err != nil {
		return adminCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	if cursor.UserID == "" || cursor.GrantedAt.IsZero() {
		return adminCursor{}, httpx.InvalidRequest("pagination.cursor 无效")
	}
	return cursor, nil
}
