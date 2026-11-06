// Package archtest 是模块边界的可执行约束：把「哪些跨模块 import / 跨模块写表
// 是既有债务」固化成 allowlist。新增边会立即失败，必须显式登记并在设计文档
// 记录原因——把「跨模块改动是必然」变成「跨模块改动是可检测、可审计的」。
//
// 基线来自架构审计（docs/backend-audit.md / .workwork/audit/modules-architecture.md）。
// 清理债务时直接从 allowlist 删除条目（测试仍会通过），因此此文件同时是债务清单。
package archtest

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ── 模块 import 允许边 ─────────────────────────────────────────────

// allowedModuleImports 是基线：模块 -> 允许 import 的其它模块集合。全部为
// DTO/服务类型/适配器用途（无 repo 直接复用）。新增条目需说明为何不能通过
// platform 接口/适配器解耦。
var allowedModuleImports = map[string][]string{
	"admin":           {"groups", "notify", "reports", "system", "users"},
	"comments":        {"notify"},
	"groupmessages":   {"users"},
	"groups":          {"notify"},
	"moments":         {"comments", "notify"},
	"notify":          {"system"},
	"privatechats":    {"privatemessages", "users"},
	"privatemessages": {"users"},
	"reports":         {"groupmessages", "privatemessages"},
	"users":           {"groups", "moments", "notify"},
}

// ── 表归属（数据所有权） ───────────────────────────────────────────

// tableOwners 是逻辑归属（Repository 所在模块）。与 schema_manifest.json 的
// 表集合必须一一对应：新增表必须在此登记归属，否则架构测试失败。
var tableOwners = map[string]string{
	"admin_audit_logs":                 "admin",
	"admin_blocked_tags":               "admin",
	"admin_role_permissions":           "admin",
	"admin_roles":                      "admin",
	"admin_user_roles":                 "admin",
	"admin_users":                      "admin",
	"app_configs":                      "platform",
	"auth_accounts":                    "auth",
	"auth_password_reset_requests":     "auth",
	"auth_refresh_tokens":              "auth",
	"auth_register_verification_codes": "auth",
	"auth_sessions":                    "auth",
	"comments":                         "comments",
	"group_announcements":              "groups",
	"group_applications":               "groups",
	"group_bans":                       "groups",
	"group_invites":                    "groups",
	"group_members":                    "groups",
	"group_message_read_state":         "groupmessages",
	"group_messages":                   "groupmessages",
	"groups":                           "groups",
	"media_upload_sessions":            "media",
	"moment_favorites":                 "moments",
	"moment_images":                    "moments",
	"moment_mentions":                  "moments",
	"moment_tags":                      "moments",
	"moments":                          "moments",
	"notifications":                    "notify",
	"private_chat_read_state":          "privatemessages",
	"private_chat_user_state":          "privatechats",
	"private_chats":                    "privatechats",
	"private_messages":                 "privatemessages",
	"reactions":                        "reactions",
	"reports":                          "reports",
	"shares":                           "share",
	"system_announcements":             "system",
	"user_profile_tags":                "users",
	"user_relations":                   "users",
	"user_settings":                    "users",
	"users":                            "users",
}

// crossModuleWritesAllowlist 是既有跨模块写表债务（架构审计 P0-4）。键为
// "模块:表"。新增写其他模块表必须显式登记；删除条目是纯收益。
var crossModuleWritesAllowlist = map[string]string{
	// admin：审核/管理动作需要直接改多域状态（后续应由各模块暴露管理接口）。
	"admin:auth_sessions":      "封禁用户时逐条撤销会话",
	"admin:comments":           "管理员删除评论",
	"admin:group_applications": "管理员处理入群申请",
	"admin:group_invites":      "管理员清理群邀请",
	"admin:group_messages":     "管理员撤回群消息",
	"admin:groups":             "管理员解散/封禁群",
	"admin:moment_tags":        "管理员删除瞬间标签",
	"admin:moments":            "管理员删除瞬间",
	"admin:private_messages":   "管理员撤回私聊消息",
	"admin:reports":            "管理员处理举报",
	"admin:users":              "管理员封禁/封号",
	// auth：注册/设置联动（auth 需在同一流程写 users/user_settings）。
	"auth:user_settings": "注册默认设置",
	"auth:users":         "注册建用户/改用户类型",
	// users：账号升级/注销/隐私设置联动其它模块状态。
	"users:auth_accounts":           "账号升级写登录凭证",
	"users:auth_sessions":           "注销/封禁撤销会话",
	"users:group_members":           "注销时退出群组",
	"users:media_upload_sessions":   "注销时清理未完成上传",
	"users:moment_favorites":        "注销时清理收藏",
	"users:notifications":           "注销时清理通知",
	"users:private_chat_read_state": "注销时清理已读位点",
	"users:private_chat_user_state": "注销时清理会话状态",
	// privatemessages：发消息时更新会话摘要/未读数（私聊会话表归属 privatechats）。
	"privatemessages:private_chats": "发消息更新会话最后消息/未读",
}

// ── 扫描实现 ─────────────────────────────────────────────────────

var (
	writeTableRe = regexp.MustCompile(`\b(?:INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+([a-z_][a-z0-9_]*)`)
	sqlStartRe   = regexp.MustCompile(`(?i)^\s*(?:--[^\n]*\n\s*)*(?:SELECT|INSERT|UPDATE|DELETE|WITH|MERGE|ALTER|CREATE|DROP|TRUNCATE)\b`)
)

// backendRoot 返回 backend-go 根目录（测试运行时 cwd = 本包目录）。
func backendRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve backend root: %v", err)
	}
	return root
}

// moduleFiles 遍历 internal/modules 下所有非测试 .go 文件，回调模块名与路径。
func moduleFiles(t *testing.T, root string, visit func(module, path string)) {
	t.Helper()
	modRoot := filepath.Join(root, "internal", "modules")
	entries, err := os.ReadDir(modRoot)
	if err != nil {
		t.Fatalf("read modules dir: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		module := entry.Name()
		err := filepath.Walk(filepath.Join(modRoot, module), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			visit(module, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk module %s: %v", module, err)
		}
	}
}

func parseGoFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}

// sqlLiteralBody 去掉 Go 字符串字面量的引号/反引号，还原 SQL 文本用于判断。
func sqlLiteralBody(raw string) string {
	body := strings.Trim(raw, "`")
	return strings.Trim(body, "\"")
}

func importEdgeAllowed(module, target string) bool {
	for _, allowed := range allowedModuleImports[module] {
		if allowed == target {
			return true
		}
	}
	return false
}

// TestModuleImportBoundaries 冻结模块间 import 图：新增/未登记边立即失败。
func TestModuleImportBoundaries(t *testing.T) {
	root := backendRoot(t)
	edges := make(map[string]bool) // "from -> to"
	moduleFiles(t, root, func(module, path string) {
		file := parseGoFile(t, path)
		for _, imp := range file.Imports {
			impPath := strings.Trim(imp.Path.Value, "\"")
			const prefix = "hotspot/backend-go/internal/modules/"
			if !strings.HasPrefix(impPath, prefix) {
				continue
			}
			target := strings.Split(strings.TrimPrefix(impPath, prefix), "/")[0]
			if target == module {
				continue
			}
			edges[module+" -> "+target] = true
			if !importEdgeAllowed(module, target) {
				t.Errorf("未登记的跨模块 import：%s -> %s（文件 %s）。"+
					"请优先通过 platform 接口/适配器解耦；确属必要则在 archtest.allowedModuleImports 登记并说明原因。",
					module, target, strings.TrimPrefix(path, root+"/"))
			}
		}
	})
	// 反向漂移：allowlist 中列出但实际已不存在的边会导致「债务清单」失真，
	// 这里只提示（t.Logf），避免清理途中的中途提交被阻断。
	for from, targets := range allowedModuleImports {
		for _, to := range targets {
			if !edges[from+" -> "+to] {
				t.Logf("allowlist 中的边已不存在，可从 allowedModuleImports 删除：%s -> %s", from, to)
			}
		}
	}
}

// TestTableOwnershipRegistryCoversSchema 表归属登记必须覆盖实际 schema：
// 新增表/迁移后忘记登记归属会让跨模块写检测失效。
func TestTableOwnershipRegistryCoversSchema(t *testing.T) {
	root := backendRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "migrations", "schema_manifest.json"))
	if err != nil {
		t.Fatalf("read schema manifest: %v", err)
	}
	var manifest struct {
		Tables []string `json:"tables"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse schema manifest: %v", err)
	}
	for _, table := range manifest.Tables {
		if _, ok := tableOwners[table]; !ok {
			t.Errorf("表 %s 未在 archtest.tableOwners 登记归属模块（新增表必须登记）", table)
		}
	}
	for table := range tableOwners {
		found := false
		for _, actual := range manifest.Tables {
			if actual == table {
				found = true
				break
			}
		}
		if !found {
			t.Logf("tableOwners 中的表已不存在，可删除：%s", table)
		}
	}
}

// ── 跨模块仓储构造约束 ─────────────────────────────────────────────

var repositoryConstructors = map[string]bool{
	"NewPostgresRepository": true,
	"NewInMemoryRepository": true,
}

// importedModuleAliases 返回文件中「本地包名 -> 归属模块」映射（只含 modules 包）。
func importedModuleAliases(file *ast.File) map[string]string {
	aliases := make(map[string]string)
	const prefix = "hotspot/backend-go/internal/modules/"
	for _, imp := range file.Imports {
		impPath := strings.Trim(imp.Path.Value, "\"")
		if !strings.HasPrefix(impPath, prefix) {
			continue
		}
		target := strings.Split(strings.TrimPrefix(impPath, prefix), "/")[0]
		local := target
		if imp.Name != nil {
			local = imp.Name.Name
		}
		aliases[local] = target
	}
	return aliases
}

// crossModuleRepoConstructions 返回模块内对其它模块仓储构造函数的调用点描述。
// 生产模块只能消费对方暴露的 Service/最小接口，不得自建对方 Repository
// （架构审计 P0-1 隐藏副本；如 admin 曾自建 system.Repository）。
func crossModuleRepoConstructions(file *ast.File, module string) []string {
	aliases := importedModuleAliases(file)
	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !repositoryConstructors[sel.Sel.Name] {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		target, ok := aliases[ident.Name]
		if !ok || target == module {
			return true
		}
		found = append(found, target+"."+sel.Sel.Name)
		return true
	})
	return found
}

// TestModulesDoNotConstructOtherModulesRepositories 模块生产代码不得构造其它模块
// 的 Repository（仓储归属对方模块，只能经其导出的 Service/最小接口消费）。
func TestModulesDoNotConstructOtherModulesRepositories(t *testing.T) {
	root := backendRoot(t)
	moduleFiles(t, root, func(module, path string) {
		for _, call := range crossModuleRepoConstructions(parseGoFile(t, path), module) {
			t.Errorf("跨模块仓储构造：%s 中调用 %s（文件 %s）。"+
				"请改为消费归属模块导出的 Service/最小接口（如 system.AnnouncementAdminStore）。",
				module, call, strings.TrimPrefix(path, root+"/"))
		}
	})
}

// TestCrossModuleRepoConstructionDetection 用最小样例验证检测器本身有效
// （防止扫描逻辑空转导致门禁形同虚设）。
func TestCrossModuleRepoConstructionDetection(t *testing.T) {
	src := `package sample

import (
	systemmodule "hotspot/backend-go/internal/modules/system"
	users "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/modules/admin"
)

var _ = systemmodule.NewPostgresRepository(nil)
var _ = users.NewInMemoryRepository()
var _ = admin.NewPostgresRepository(nil)
`
	file, err := parser.ParseFile(token.NewFileSet(), "sample.go", src, 0)
	if err != nil {
		t.Fatalf("parse sample: %v", err)
	}
	got := crossModuleRepoConstructions(file, "admin")
	want := map[string]bool{
		"system.NewPostgresRepository": true,
		"users.NewInMemoryRepository":  true,
	}
	if len(got) != len(want) {
		t.Fatalf("detected = %v, want keys of %v", got, want)
	}
	for _, call := range got {
		if !want[call] {
			t.Fatalf("unexpected detection: %s (all=%v)", call, got)
		}
	}
}

// TestCrossModuleSQLWritesAreAllowlisted 扫描模块 SQL 字面量中的写表动作，
// 除归属模块自身外必须命中有原因的 allowlist。
func TestCrossModuleSQLWritesAreAllowlisted(t *testing.T) {
	root := backendRoot(t)
	seen := make(map[string]bool)
	moduleFiles(t, root, func(module, path string) {
		file := parseGoFile(t, path)
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING || !sqlStartRe.MatchString(sqlLiteralBody(lit.Value)) {
				return true
			}
			for _, match := range writeTableRe.FindAllStringSubmatch(lit.Value, -1) {
				table := strings.ToLower(match[1])
				if table == "set" {
					continue // "DO UPDATE SET" 的误匹配
				}
				owner, known := tableOwners[table]
				if !known {
					t.Errorf("未登记归属的表写入：%s 写 %s（文件 %s）——请在 archtest.tableOwners 登记归属",
						module, table, strings.TrimPrefix(path, root+"/"))
					continue
				}
				if owner == module || owner == "platform" {
					continue
				}
				edge := module + ":" + table
				seen[edge] = true
				if _, allowed := crossModuleWritesAllowlist[edge]; !allowed {
					t.Errorf("未登记的跨模块写表：%s 写 %s（归属 %s，文件 %s）。"+
						"请改走归属模块的管理接口；确属必要则在 archtest.crossModuleWritesAllowlist 登记原因。",
						module, table, owner, strings.TrimPrefix(path, root+"/"))
				}
			}
			return true
		})
	})
	var allowlisted []string
	for edge := range crossModuleWritesAllowlist {
		if !seen[edge] {
			allowlisted = append(allowlisted, edge)
		}
	}
	sort.Strings(allowlisted)
	if len(allowlisted) > 0 {
		t.Logf("以下跨模块写债务已不存在（可从 allowlist 删除）：%v", allowlisted)
	}
}
