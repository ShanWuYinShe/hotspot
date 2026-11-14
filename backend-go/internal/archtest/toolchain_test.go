package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestGoToolchainVersionsAligned：go.mod 的 go 指令 patch 版本必须与 CI
// setup-go、Dockerfile/Compose 的 GO_VERSION 完全一致，防止「本地能过、
// CI/镜像编译差异」的工具链漂移（审计 E6）。
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
	// 逐一校验。ci.yml 有多个 setup-go（backend / backend-integration / gitleaks），
	// 只比对第一处会让其余 job 的版本漂移静默通过。
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
	// 单引号/双引号都接受：漏掉某种写法会让对应 setup-go 静默逃过校验。
	ciVersions := requireAllMatches(filepath.Join(repoRoot, ".github", "workflows", "ci.yml"), regexp.MustCompile("go-version:\\s*['\"]([0-9.]+)['\"]"))
	for i, ciVersion := range ciVersions {
		if ciVersion != goModVersion {
			t.Fatalf("Go toolchain drift: ci.yml setup-go go-version[%d]=%s, go.mod=%s（每个 job 的 go-version 都必须等于 go.mod 的 go 指令）", i, ciVersion, goModVersion)
		}
	}
}
