package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// 架构守卫（C2/C5 收口）：这两条规则把「已经清掉的进程级全局」和
// 「多写仓储方法必须显式事务」固化到 CI，防止后续改动回潮。

// mutableGlobalAllowlist：仍保留的进程级可变全局（组合根基础设施单例）。
// 新增条目必须在 PR/commit 说明理由——C2 的目标是逐轮收敛而非永久豁免。
var mutableGlobalAllowlist = map[string]string{
	"internal/platform/observability/logger.go:defaultLogger":        "组合根/后台兜底：模块已全部改走 observability.FromContext(ctx)，仅 platform 启动期与无 ctx 后台任务使用 L()",
	"internal/platform/realtime/global.go:defaultHub":                "测试/后台 fallback：生产 NewApp 持有应用级 Hub 实例并显式注入（context/WS），全局仅未注入时兜底",
	"internal/platform/requestmeta/requestmeta.go:trustedProxyCIDRs": "启动期一次写入的代理 CIDR 配置（只读快照）",
}

// TestNoNewMutablePackageGlobals 扫描 platform/modules 下包级 var 的 atomic/sync.Map
// 类型，只允许 allowlist 中的条目。新增全局会直接失败并打印文件:变量。
func TestNoNewMutablePackageGlobals(t *testing.T) {
	root := backendRoot(t)
	var found []string
	for _, dir := range []string{"internal/platform", "internal/modules"} {
		err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.VAR {
					continue
				}
				for _, spec := range gen.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					typeText := exprText(value.Type)
					if !strings.Contains(typeText, "atomic.Pointer") && !strings.Contains(typeText, "atomic.Value") && !strings.Contains(typeText, "sync.Map") {
						continue
					}
					for _, name := range value.Names {
						rel, _ := filepath.Rel(root, path)
						found = append(found, rel+":"+name.Name+" type="+typeText)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	sort.Strings(found)
	var unexpected []string
	for _, item := range found {
		key := strings.SplitN(item, " type=", 2)[0]
		if _, ok := mutableGlobalAllowlist[key]; !ok {
			unexpected = append(unexpected, item)
		}
	}
	if len(unexpected) > 0 {
		t.Fatalf("发现未登记的可变包级全局（请改成显式依赖注入，或在 allowlist 说明理由）:\n  %s", strings.Join(unexpected, "\n  "))
	}
}

// repoMultiWriteAllowlist：多写仓储方法的显式豁免（调用方事务/单语句幂等）。
// key 为 文件:函数名；value 说明为什么不需要自身 RunInTransaction。
var repoMultiWriteAllowlist = map[string]string{
	"internal/modules/groupmessages/repo_postgres.go:MarkAsRead":           "if/else 互斥分支各一条写，不存在顺序多写",
	"internal/modules/users/repo_postgres.go:execInTx":                     "事务内/池连接二选一的辅助函数（互斥分支）",
	"internal/modules/users/repo_postgres_profile_writer.go:ProvisionUser": "调用方事务契约：由 auth.Register 的 RunInTransaction 包裹（同一 txCtx）",
}

// TestRepoMultiWriteMethodsUseTransaction：扫描 modules 下 repo_postgres*.go，
// 同一函数内直接执行 >=2 次写语句（.Exec(）时必须出现 RunInTransaction/
// WithTx，或进入 allowlist（例如 ProvisionUser/EnsureChat 由调用方事务包裹）。
func TestRepoMultiWriteMethodsUseTransaction(t *testing.T) {
	root := backendRoot(t)
	var violations []string
	err := filepath.Walk(filepath.Join(root, "internal/modules"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		baseName := filepath.Base(path)
		if !strings.HasPrefix(baseName, "repo_postgres") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			execs := 0
			hasTx := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if sel.Sel.Name == "Exec" {
					execs++
				}
				if sel.Sel.Name == "RunInTransaction" || sel.Sel.Name == "WithTx" {
					hasTx = true
				}
				return true
			})
			if execs < 2 || hasTx {
				continue
			}
			key := rel + ":" + fn.Name.Name
			if _, ok := repoMultiWriteAllowlist[key]; ok {
				continue
			}
			violations = append(violations, key+" execs="+itoa(execs))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk modules: %v", err)
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("多写仓储方法缺少显式事务（或用 allowlist 记录调用方事务契约）:\n  %s", strings.Join(violations, "\n  "))
	}
}

func exprText(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.IndexExpr:
		return exprText(e.X) + "[" + exprText(e.Index) + "]"
	case *ast.IndexListExpr:
		parts := make([]string, 0, len(e.Indices))
		for _, idx := range e.Indices {
			parts = append(parts, exprText(idx))
		}
		return exprText(e.X) + "[" + strings.Join(parts, ", ") + "]"
	case *ast.SelectorExpr:
		return exprText(e.X) + "." + e.Sel.Name
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprText(e.X)
	case *ast.ArrayType:
		return "[]" + exprText(e.Elt)
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// TestModulesUseContextHubNotGlobal：模块内禁止直接取进程级 realtime.DefaultHub，
// 统一 realtime.HubFromContext(ctx)（由 bootstrap 中间件显式注入）。新增直取
// 全局会在同进程多 App/并行测试下串台，直接失败。
func TestModulesUseContextHubNotGlobal(t *testing.T) {
	root := backendRoot(t)
	var violations []string
	moduleFiles(t, root, func(module, path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(data), "realtime.DefaultHub") || strings.Contains(string(data), "realtime.SetDefaultHub") {
			violations = append(violations, path)
		}
	})
	if len(violations) > 0 {
		t.Fatalf("模块内不得直接使用进程级 realtime Hub，请改用 realtime.HubFromContext(ctx):\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestModulesUseContextLoggerNotGlobal：模块内禁止直接取进程级 observability.L()，
// 统一 observability.FromContext(ctx)（由 bootstrap 中间件显式注入；后台无 ctx
// 的 platform 层代码仍可用 L() 兜底）。
func TestModulesUseContextLoggerNotGlobal(t *testing.T) {
	root := backendRoot(t)
	var violations []string
	moduleFiles(t, root, func(module, path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(data), "observability.L()") {
			violations = append(violations, path)
		}
	})
	if len(violations) > 0 {
		t.Fatalf("模块内不得直接使用进程级 logger，请改用 observability.FromContext(ctx):\n  %s", strings.Join(violations, "\n  "))
	}
}
