package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// goToolchainVersionRe 匹配 workflow 里 setup-go 的 go-version（单/双引号都
// 接受：漏掉某种写法会让对应 setup-go 静默逃过校验）。
var goToolchainVersionRe = regexp.MustCompile("go-version:\\s*['\"]([0-9.]+)['\"]")

// TestGoToolchainVersionsAligned：go.mod 的 go 指令 patch 版本必须与全部 CI
// workflow 的 setup-go、Dockerfile/Compose 的 GO_VERSION 完全一致，防止「本地
// 能过、CI/镜像编译差异」的工具链漂移（审计 E6）。
func TestGoToolchainVersionsAligned(t *testing.T) {
	backendRootDir := backendRoot(t)
	repoRoot := filepath.Dir(backendRootDir)

	requireMatch := func(path string, re *regexp.Regexp) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		match := re.FindSubmatch(data)
		if len(match) < 2 {
			t.Fatalf("cannot parse Go version from %s (pattern %s)", path, re.String())
		}
		return string(match[1])
	}

	// requireAllMatches：与 requireMatch 不同，这里要求取出文件里**全部**匹配并
	// 逐一校验。一个 workflow 有多个 setup-go（如 ci.yml 的 backend /
	// backend-integration / gitleaks），只比对第一处会让其余 job 的版本漂移静默通过。
	requireAllMatches := func(path string, re *regexp.Regexp) []string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		matches := re.FindAllSubmatch(data, -1)
		if len(matches) == 0 {
			t.Fatalf("cannot parse Go version from %s (pattern %s)", path, re.String())
		}
		versions := make([]string, 0, len(matches))
		for _, match := range matches {
			versions = append(versions, string(match[1]))
		}
		return versions
	}

	goModVersion := requireMatch(filepath.Join(backendRootDir, "go.mod"), regexp.MustCompile("(?m)^go\\s+(\\d+\\.\\d+\\.\\d+)\\s*$"))
	dockerfileVersion := requireMatch(filepath.Join(backendRootDir, "Dockerfile"), regexp.MustCompile("(?m)^ARG GO_VERSION=(\\S+)\\s*$"))
	dockerfileDevVersion := requireMatch(filepath.Join(backendRootDir, "Dockerfile.dev"), regexp.MustCompile("(?m)^ARG GO_VERSION=(\\S+)\\s*$"))
	if goModVersion != dockerfileVersion || goModVersion != dockerfileDevVersion {
		t.Fatalf("Go toolchain drift: go.mod=%s Dockerfile=%s Dockerfile.dev=%s", goModVersion, dockerfileVersion, dockerfileDevVersion)
	}

	// 遍历**全部** workflow（不只 ci.yml）：夜间 fuzz.yml 也 pin 了 go-version，
	// 只查 ci.yml 时升级 Go 后夜间任务会静默跑在旧工具链上（编译失败才会暴露，
	// 且只是定时任务）。
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("read workflows dir: %v", err)
	}
	matchedWorkflows := 0
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml")) {
			continue
		}
		path := filepath.Join(workflowsDir, entry.Name())
		versions := requireAllMatchesIfAny(t, path)
		if len(versions) == 0 {
			continue // 未使用 setup-go 的 workflow（如纯前端 job）
		}
		matchedWorkflows++
		for i, version := range versions {
			if version != goModVersion {
				t.Fatalf("Go toolchain drift: .github/workflows/%s setup-go go-version[%d]=%s, go.mod=%s（每个 job 的 go-version 都必须等于 go.mod 的 go 指令）", entry.Name(), i, version, goModVersion)
			}
		}
	}
	if matchedWorkflows == 0 {
		t.Fatal("未在任何 workflow 中找到 go-version：setup-go 校验可能已失效")
	}
	// fuzz.yml 必须显式 pin Go 版本：若误删 go-version 行，上面的遍历会静默跳过
	// 该文件，夜间 fuzz 将跟随 runner 默认 Go。
	requireAllMatches(filepath.Join(workflowsDir, "fuzz.yml"), goToolchainVersionRe)
}

// requireAllMatchesIfAny 返回文件里全部 go-version 匹配；文件未使用 setup-go 时
// 返回空切片（不视为错误）。workflow 目录里存在非 Go job 的 workflow，不能要求
// 每个文件都有 go-version。
func requireAllMatchesIfAny(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	matches := goToolchainVersionRe.FindAllSubmatch(data, -1)
	versions := make([]string, 0, len(matches))
	for _, match := range matches {
		versions = append(versions, string(match[1]))
	}
	return versions
}
