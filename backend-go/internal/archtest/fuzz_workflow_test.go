package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	// fuzzFuncRe 匹配 _test.go 中的 fuzz 入口：Fuzz 前缀 + *testing.F 参数是
	// Go 原生 fuzz 目标的固定形态（普通测试/基准不会以 Fuzz 命名）。要求参数
	// 类型也能避免把恰好以 Fuzz 命名的普通测试误判成 fuzz 目标。
	fuzzFuncRe = regexp.MustCompile(`(?m)^func (Fuzz\w+)\(\s*\w+\s+\*testing\.F\s*\)`)
	// fuzzTargetsBlockRe 提取 fuzz.yml 里 shell 数组 targets=( ... ) 的内容。
	fuzzTargetsBlockRe = regexp.MustCompile(`(?s)\n\s*targets=\((.*?)\n\s*\)`)
	// fuzzTargetEntryRe 提取数组中的每个 "包:目标" 条目。
	fuzzTargetEntryRe = regexp.MustCompile(`"([^"]+)"`)
)

// goFuzzTargets 扫描 backend-go 下全部 _test.go，返回 "包目录:Fuzz 函数名"
// 集合（包目录为相对 backend-go 根的 "./xxx/" 形式，与 workflow 条目同口径）。
func goFuzzTargets(t *testing.T, backendRootDir string) map[string]bool {
	t.Helper()
	targets := make(map[string]bool)
	err := filepath.WalkDir(backendRootDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			// 跳过版本库/构建产物/测试语料目录，避免扫描到非源码副本。
			if path != backendRootDir && (strings.HasPrefix(name, ".") || name == "tmp" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := fuzzFuncRe.FindAllSubmatch(data, -1)
		if len(matches) == 0 {
			return nil
		}
		rel, err := filepath.Rel(backendRootDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkg := "./" + filepath.ToSlash(rel) + "/"
		for _, match := range matches {
			targets[pkg+":"+string(match[1])] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend-go for fuzz targets: %v", err)
	}
	return targets
}

// parseFuzzWorkflowTargets 解析 fuzz.yml 的 targets 数组，返回规范化的
// "包:目标" 集合；重复条目/无法解析的条目直接判失败（重复会让同一目标跑两次）。
func parseFuzzWorkflowTargets(t *testing.T, path string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	block := fuzzTargetsBlockRe.FindSubmatch(data)
	if len(block) < 2 {
		t.Fatalf("cannot parse fuzz targets array from %s（期望 shell 数组 targets=( ... )）", path)
	}
	targets := make(map[string]bool)
	for _, line := range strings.Split(string(block[1]), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entry := fuzzTargetEntryRe.FindStringSubmatch(line)
		if len(entry) < 2 {
			t.Fatalf("无法解析 fuzz 目标条目 %q（%s）", line, path)
		}
		target := normalizeFuzzTarget(t, entry[1])
		if targets[target] {
			t.Errorf("fuzz 目标重复登记：%s（%s）", target, path)
			continue
		}
		targets[target] = true
	}
	if len(targets) == 0 {
		t.Fatalf("%s 的 fuzz 目标清单为空", path)
	}
	return targets
}

// normalizeFuzzTarget 把 workflow 条目规范化为 "包目录:目标名"：包目录统一为
// "./xxx/" 形式（接受 ./ 前缀缺失 / 尾斜杠缺失的写法），目标名取最后一个冒号
// 之后的部分。
func normalizeFuzzTarget(t *testing.T, raw string) string {
	t.Helper()
	idx := strings.LastIndex(raw, ":")
	if idx <= 0 || idx == len(raw)-1 {
		t.Fatalf("fuzz 目标条目 %q 不符合 \"包目录:Fuzz目标名\" 格式", raw)
	}
	pkg := raw[:idx]
	name := raw[idx+1:]
	if !strings.HasPrefix(pkg, "./") {
		pkg = "./" + pkg
	}
	if !strings.HasSuffix(pkg, "/") {
		pkg += "/"
	}
	return pkg + ":" + name
}

// TestFuzzWorkflowTargetsMatchCode：夜间 fuzz workflow 的目标清单必须与代码中
// 全部 `func Fuzz*` 目标完全一致。fuzz 只在定时任务里跑（PR 门禁只回放种子），
// 新增/改名的目标一旦漏登记就永久不进 CI 且无任何红灯；多登记的条目会让夜间
// 任务报「找不到目标」失败。两向差集都报出来，便于一次改完。
func TestFuzzWorkflowTargetsMatchCode(t *testing.T) {
	backendRootDir := backendRoot(t)
	workflowPath := filepath.Join(filepath.Dir(backendRootDir), ".github", "workflows", "fuzz.yml")

	codeTargets := goFuzzTargets(t, backendRootDir)
	if len(codeTargets) == 0 {
		// 检测器失效（正则/遍历写坏）时不能静默通过：否则门禁形同虚设。
		t.Fatal("未扫描到任何 Fuzz 目标：检测器可能已失效")
	}
	workflowTargets := parseFuzzWorkflowTargets(t, workflowPath)

	var missing, stale []string
	for target := range codeTargets {
		if !workflowTargets[target] {
			missing = append(missing, target)
		}
	}
	for target := range workflowTargets {
		if !codeTargets[target] {
			stale = append(stale, target)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	if len(missing) > 0 {
		t.Errorf("fuzz workflow 漏登记代码中的目标：%v（新增 Fuzz 目标必须同步 .github/workflows/fuzz.yml 的 targets 清单）", missing)
	}
	if len(stale) > 0 {
		t.Errorf("fuzz workflow 登记了代码中不存在的目标：%v（改名/删除 Fuzz 目标后必须同步清单）", stale)
	}
}
