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

	goModVersion := requireMatch(filepath.Join(backendRootDir, "go.mod"), regexp.MustCompile("(?m)^go\\s+(\\d+\\.\\d+\\.\\d+)\\s*$"))
	dockerfileVersion := requireMatch(filepath.Join(backendRootDir, "Dockerfile"), regexp.MustCompile("(?m)^ARG GO_VERSION=(\\S+)\\s*$"))
	dockerfileDevVersion := requireMatch(filepath.Join(backendRootDir, "Dockerfile.dev"), regexp.MustCompile("(?m)^ARG GO_VERSION=(\\S+)\\s*$"))
	ciVersion := requireMatch(filepath.Join(repoRoot, ".github", "workflows", "ci.yml"), regexp.MustCompile("go-version:\\s*'([0-9.]+)'"))
	if goModVersion != dockerfileVersion || goModVersion != dockerfileDevVersion || goModVersion != ciVersion {
		t.Fatalf("Go toolchain drift: go.mod=%s Dockerfile=%s Dockerfile.dev=%s CI=%s", goModVersion, dockerfileVersion, dockerfileDevVersion, ciVersion)
	}
}
