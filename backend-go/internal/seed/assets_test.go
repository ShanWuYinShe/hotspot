package seed

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assets_test.go — 外置素材（devdata/seed/media）的解析/加载/校验单测。
//
// 单测不依赖仓库里的真实素材目录：素材用临时目录现场生成 25 张固定尺寸 JPEG，
// 因此 `go test ./internal/seed/...` 在任意工作目录都能跑（CI 不需要额外准备
// devdata）。真实素材的存在性与尺寸由 devdata/seed/media 的提交内容保证，并在
// 开发环境 --dry-run 时实际加载校验（见 scripts/seed-dev.sh）。
//
// 替换素材时必须保持文件名与尺寸（avatar 400x400 / group 800x800 /
// moment 1200x800），否则加载会立即失败，避免写入错误比例的 media 记录。

// testAssetsDirPath 是包内共享的素材目录：TestMain 生成一次，避免每个用例
// 重复编码 JPEG（plan/execute 等用例经 testPlanOptions 复用同一目录）。
var testAssetsDirPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "hotspot-seed-assets-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建测试素材临时目录失败: %v\n", err)
		os.Exit(1)
	}
	if err := writeTestAssets(dir); err != nil {
		fmt.Fprintf(os.Stderr, "生成测试素材失败: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	testAssetsDirPath = dir
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// writeTestAssets 在 dir 下生成 assetManifest 要求的全部 JPEG（尺寸按
// expectedAssetSize，内容为纯色，仅用于校验解码/加载逻辑）。
func writeTestAssets(dir string) error {
	for _, name := range assetManifest {
		width, height := expectedAssetSize(name)
		if width <= 0 || height <= 0 {
			return fmt.Errorf("素材 %s 没有登记期望尺寸", name)
		}
		if err := writeTestJPEG(filepath.Join(dir, name), width, height); err != nil {
			return err
		}
	}
	return nil
}

// writeTestJPEG 生成指定尺寸的纯色 JPEG 文件。
func writeTestJPEG(path string, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fill := color.RGBA{R: 0x3C, G: 0x7A, B: 0xB4, A: 0xFF}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: fill}, image.Point{}, draw.Src)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建测试素材 %s 失败: %w", path, err)
	}
	if encodeErr := jpeg.Encode(file, img, &jpeg.Options{Quality: 70}); encodeErr != nil {
		_ = file.Close()
		return fmt.Errorf("编码测试素材 %s 失败: %w", path, encodeErr)
	}
	if closeErr := file.Close(); closeErr != nil {
		return fmt.Errorf("写入测试素材 %s 失败: %w", path, closeErr)
	}
	return nil
}

// TestAssetManifest 素材清单必须与文档口径一致：25 张，文件名唯一且可推导尺寸。
func TestAssetManifest(t *testing.T) {
	if len(assetManifest) != 25 {
		t.Fatalf("素材清单数量 = %d，期望 25（9 头像 + 4 群头像 + 12 瞬间图）", len(assetManifest))
	}
	seen := make(map[string]struct{}, len(assetManifest))
	for _, name := range assetManifest {
		if _, ok := seen[name]; ok {
			t.Errorf("素材清单重复: %s", name)
		}
		seen[name] = struct{}{}
		if !strings.HasSuffix(name, ".jpg") {
			t.Errorf("素材 %s 不是 .jpg", name)
		}
		width, height := expectedAssetSize(name)
		if width <= 0 || height <= 0 {
			t.Errorf("素材 %s 没有登记期望尺寸（检查文件名前缀）", name)
		}
	}
}

// TestLoadAllAssets 用临时目录生成的素材验证加载结果：数量、尺寸、大小、MD5、JPEG 魔数。
func TestLoadAllAssets(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestAssets(dir); err != nil {
		t.Fatalf("生成测试素材失败: %v", err)
	}
	assets, err := loadAllAssets(dir)
	if err != nil {
		t.Fatalf("加载素材失败: %v", err)
	}
	if len(assets) != len(assetManifest) {
		t.Fatalf("素材数量 = %d，期望 %d", len(assets), len(assetManifest))
	}
	for _, name := range assetManifest {
		asset, ok := assets[name]
		if !ok {
			t.Fatalf("缺少素材 %s", name)
		}
		wantWidth, wantHeight := expectedAssetSize(name)
		if asset.Width != wantWidth || asset.Height != wantHeight {
			t.Errorf("素材 %s 尺寸 %dx%d，期望 %dx%d", name, asset.Width, asset.Height, wantWidth, wantHeight)
		}
		if asset.Size != len(asset.Data) || asset.Size == 0 {
			t.Errorf("素材 %s 大小字段与内容不一致", name)
		}
		if len(asset.MD5) != 32 {
			t.Errorf("素材 %s md5 = %q（期望 32 位 hex）", name, asset.MD5)
		}
		if len(asset.Data) < 3 || asset.Data[0] != 0xFF || asset.Data[1] != 0xD8 {
			t.Errorf("素材 %s 不是 JPEG", name)
		}
	}
	// MD5 必须是内容摘要（media_upload_sessions.md5 与对象内容一致）。
	raw, err := os.ReadFile(filepath.Join(dir, "avatar_lumos.jpg"))
	if err != nil {
		t.Fatalf("读取素材失败: %v", err)
	}
	sum := md5.Sum(raw)
	if got, want := assets["avatar_lumos.jpg"].MD5, hex.EncodeToString(sum[:]); got != want {
		t.Errorf("素材 md5 = %q，期望 %q", got, want)
	}
	// 清单之外的额外文件不影响加载（允许开发者在目录里临时放草稿素材）。
	if err := os.WriteFile(filepath.Join(dir, "draft_ignore.jpg"), raw, 0o644); err != nil {
		t.Fatalf("写入额外文件失败: %v", err)
	}
	if _, err := loadAllAssets(dir); err != nil {
		t.Errorf("额外文件不应导致加载失败: %v", err)
	}
}

// TestLoadAllAssetsMissingFiles 缺文件必须明确报错：列出缺失文件、提示素材目录与 README。
func TestLoadAllAssetsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestAssets(dir); err != nil {
		t.Fatalf("生成测试素材失败: %v", err)
	}
	missing := []string{"avatar_lumos.jpg", "moment_12.jpg"}
	for _, name := range missing {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			t.Fatalf("删除测试素材 %s 失败: %v", name, err)
		}
	}
	_, err := loadAllAssets(dir)
	if err == nil {
		t.Fatal("缺少素材时应报错，不能静默跳过")
	}
	message := err.Error()
	for _, want := range append(missing, "devdata/seed/media", "README.md") {
		if !strings.Contains(message, want) {
			t.Errorf("错误信息缺少 %q: %s", want, message)
		}
	}
}

// TestLoadAllAssetsRejectsWrongSize 替换素材但尺寸不符必须失败（防止写入错误比例）。
func TestLoadAllAssetsRejectsWrongSize(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestAssets(dir); err != nil {
		t.Fatalf("生成测试素材失败: %v", err)
	}
	if err := writeTestJPEG(filepath.Join(dir, "group_1.jpg"), 400, 400); err != nil {
		t.Fatalf("覆盖素材失败: %v", err)
	}
	_, err := loadAllAssets(dir)
	if err == nil || !strings.Contains(err.Error(), "尺寸") {
		t.Fatalf("尺寸不符应报错，实际 err = %v", err)
	}
}

// TestLoadAllAssetsRejectsNonJPEG 非 JPEG 内容必须失败（魔数校验）。
func TestLoadAllAssetsRejectsNonJPEG(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestAssets(dir); err != nil {
		t.Fatalf("生成测试素材失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "avatar_admin.jpg"), []byte("not a jpeg"), 0o644); err != nil {
		t.Fatalf("覆盖素材失败: %v", err)
	}
	_, err := loadAllAssets(dir)
	if err == nil || !strings.Contains(err.Error(), "JPEG") {
		t.Fatalf("非 JPEG 应报错，实际 err = %v", err)
	}
}

// TestLoadAllAssetsMissingDir 目录不存在时给出含素材位置与 README 的明确错误。
func TestLoadAllAssetsMissingDir(t *testing.T) {
	_, err := loadAllAssets(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("素材目录不存在时应报错")
	}
	if !strings.Contains(err.Error(), "devdata/seed/media") || !strings.Contains(err.Error(), "README.md") {
		t.Errorf("错误信息应提示素材位置: %v", err)
	}
}

// TestResolveAssetsDirExplicit 显式路径（模拟 -assets-dir / SEED_ASSETS_DIR）优先且返回绝对路径。
func TestResolveAssetsDirExplicit(t *testing.T) {
	dir := t.TempDir()
	resolved, tried, err := resolveAssetsDir(dir)
	if err != nil {
		t.Fatalf("解析显式素材目录失败: %v", err)
	}
	if resolved != dir {
		t.Errorf("resolved = %q，期望 %q", resolved, dir)
	}
	if len(tried) != 1 || tried[0] != dir {
		t.Errorf("tried = %v，期望只尝试显式路径", tried)
	}
	missing := filepath.Join(dir, "missing")
	if _, _, err := resolveAssetsDir(missing); err == nil {
		t.Fatal("显式路径不存在时应报错")
	} else if !strings.Contains(err.Error(), missing) || !strings.Contains(err.Error(), "devdata/seed/media") {
		t.Errorf("错误信息应包含显式路径与默认素材位置: %v", err)
	}
}

// TestResolveAssetsDirDefaultsFromBackendGoDir 未显式指定时，从 backend-go 目录
// 运行（宿主机 go run / dev 容器 cwd=/src）应命中 ../devdata/seed/media。
func TestResolveAssetsDirDefaultsFromBackendGoDir(t *testing.T) {
	root := t.TempDir()
	backendDir := filepath.Join(root, "backend-go")
	mediaDir := filepath.Join(root, "devdata", "seed", "media")
	for _, dir := range []string{backendDir, mediaDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}
	t.Chdir(backendDir)
	resolved, tried, err := resolveAssetsDir("")
	if err != nil {
		t.Fatalf("默认解析应命中 ../devdata/seed/media: %v（尝试过: %v）", err, tried)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("读取当前目录失败: %v", err)
	}
	want := filepath.Join(filepath.Dir(cwd), "devdata", "seed", "media")
	if resolved != want {
		t.Errorf("resolved = %q，期望 %q", resolved, want)
	}
}

// TestResolveAssetsDirDefaultErrorListsAttempts 默认候选全部缺失时，错误信息必须
// 列出每个尝试过的路径（而不是只报一个相对路径）。
func TestResolveAssetsDirDefaultErrorListsAttempts(t *testing.T) {
	t.Chdir(t.TempDir())
	resolved, tried, err := resolveAssetsDir("")
	if err == nil {
		t.Fatalf("空目录应解析失败，实际命中 %q", resolved)
	}
	if len(tried) == 0 || len(tried) > len(defaultAssetsDirCandidates) {
		t.Errorf("尝试路径数 = %d，期望 1..%d（%v）", len(tried), len(defaultAssetsDirCandidates), tried)
	}
	seen := make(map[string]struct{}, len(tried))
	for _, attempted := range tried {
		if _, ok := seen[attempted]; ok {
			t.Errorf("尝试路径重复: %q（%v）", attempted, tried)
		}
		seen[attempted] = struct{}{}
	}
	message := err.Error()
	if !strings.Contains(message, "devdata/seed/media") {
		t.Errorf("错误信息应提示 devdata/seed/media: %s", message)
	}
	for _, attempted := range tried {
		if !strings.Contains(message, attempted) {
			t.Errorf("错误信息未列出尝试路径 %q: %s", attempted, message)
		}
	}
}

// TestExpectedAssetSize 文件名前缀 → 约定尺寸。
func TestExpectedAssetSize(t *testing.T) {
	cases := map[string][2]int{
		"avatar_lumos.jpg": {400, 400},
		"group_3.jpg":      {800, 800},
		"moment_12.jpg":    {1200, 800},
		"unknown.jpg":      {0, 0},
	}
	for name, want := range cases {
		width, height := expectedAssetSize(name)
		if width != want[0] || height != want[1] {
			t.Fatalf("expectedAssetSize(%q) = (%d, %d)，期望 (%d, %d)", name, width, height, want[0], want[1])
		}
	}
}
