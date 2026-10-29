package seed

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // 注册 JPEG 解码器（image.DecodeConfig 校验素材尺寸）
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// assets.go — 外置开发素材（仓库根 devdata/seed/media/）的解析、加载与校验。
//
// 素材不再 //go:embed 打包进后端二进制：用户 2026-10-29 反馈测试数据设施不应
// 进入生产产物（3.6MB 测试图片、素材无法替换、不便维护）。素材因此外置到仓库根
// devdata/seed/media/，seed 运行时从文件系统读取；生产镜像构建上下文是
// backend-go/，天然不包含该目录，dev 容器由 docker-compose.dev.yml 挂载。
//
// 素材清单固定为 25 张 JPEG（尺寸在加载时校验；替换素材但尺寸不符会立即失败，
// 避免把错误尺寸的 width/height 写进 media_upload_sessions 造成图片比例错乱）：
//   - avatar_*.jpg 400x400  用户头像（9 张）
//   - group_*.jpg  800x800  群头像（4 张）
//   - moment_*.jpg 1200x800 瞬间图 / 聊天图（12 张）

// assetManifest 是素材目录必须包含的全部文件（固定文件名，排序稳定，便于
// 错误信息与文档保持一致）。新增/删除素材时同步修改这里、expectedAssetSize、
// specs_*/mediaplan.go 的引用与 assets_test.go。
var assetManifest = []string{
	"avatar_admin.jpg", "avatar_chen.jpg", "avatar_lin.jpg", "avatar_lumos.jpg", "avatar_marco.jpg",
	"avatar_mika.jpg", "avatar_privacy.jpg", "avatar_sophia.jpg", "avatar_xiaoman.jpg",
	"group_1.jpg", "group_2.jpg", "group_3.jpg", "group_4.jpg",
	"moment_01.jpg", "moment_02.jpg", "moment_03.jpg", "moment_04.jpg",
	"moment_05.jpg", "moment_06.jpg", "moment_07.jpg", "moment_08.jpg",
	"moment_09.jpg", "moment_10.jpg", "moment_11.jpg", "moment_12.jpg",
}

// defaultAssetsDirCandidates 是未显式指定素材目录时依次尝试的候选路径（相对
// 进程工作目录）。官方两种运行方式的 cwd 都是 backend-go 源码根：
//   - 宿主机：backend-go 下 go run ./cmd/seed → ../devdata/seed/media 命中仓库根素材；
//   - dev compose 容器：backend-go 挂载在 /src、仓库根 devdata/ 挂载在 /devdata
//     （见 docker-compose.dev.yml）→ ../devdata/seed/media 即 /devdata/seed/media。
//
// 末尾两个绝对路径兜底其它挂载布局（例如「仓库根整体挂载在 /src」时命中
// /src/devdata/seed/media）；scripts/seed-dev.sh 还会显式传 -assets-dir。
var defaultAssetsDirCandidates = []string{
	filepath.Join("..", "devdata", "seed", "media"),
	"/devdata/seed/media",
	"/src/devdata/seed/media",
}

// seedAsset 是一张已加载素材的元数据（含字节内容）。
type seedAsset struct {
	Name   string // 文件名（不含目录，如 avatar_lumos.jpg）
	Width  int
	Height int
	Size   int
	MD5    string // 内容 MD5（32 位 hex，写入 media_upload_sessions.md5）
	Data   []byte
}

// expectedAssetSize 返回素材文件名的约定尺寸。未登记的前缀返回 0,0（不校验）。
func expectedAssetSize(name string) (width, height int) {
	switch {
	case strings.HasPrefix(name, "avatar_"):
		return 400, 400
	case strings.HasPrefix(name, "group_"):
		return 800, 800
	case strings.HasPrefix(name, "moment_"):
		return 1200, 800
	default:
		return 0, 0
	}
}

// assetManifestSummary 描述素材目录应有的内容（错误信息与文档口径复用）。
func assetManifestSummary() string {
	return fmt.Sprintf("%d 张 JPEG（avatar_*.jpg 9 张 400x400 / group_*.jpg 4 张 800x800 / moment_*.jpg 12 张 1200x800）", len(assetManifest))
}

// resolveAssetsDir 解析素材目录：explicit 非空时只认该路径（-assets-dir /
// SEED_ASSETS_DIR），否则按 defaultAssetsDirCandidates 依次探测。
//
// 返回素材目录的绝对路径，以及按尝试顺序记录的「尝试过的绝对路径」
// （错误信息会全部列出，避免只报一个与当前运行方式无关的相对路径）。
func resolveAssetsDir(explicit string) (dir string, tried []string, err error) {
	candidates := defaultAssetsDirCandidates
	if explicit != "" {
		candidates = []string{explicit}
	}
	tried = make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		abs, absErr := filepath.Abs(candidate)
		if absErr != nil {
			abs = candidate
		}
		// 不同候选可能解析到同一绝对路径（如 cwd=/tmp 时 ../devdata/seed/media
		// 与 /devdata/seed/media 相同）：去重后再记录，错误信息不重复。
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		tried = append(tried, abs)
		if absErr != nil {
			continue
		}
		info, statErr := os.Stat(abs)
		if statErr == nil && info.IsDir() {
			return abs, tried, nil
		}
	}
	if explicit != "" {
		return "", tried, fmt.Errorf(
			"素材目录 %s 不存在或不是目录（来自 -assets-dir / SEED_ASSETS_DIR）。%s",
			tried[0], assetsDirHint(),
		)
	}
	return "", tried, fmt.Errorf(
		"未找到开发素材目录 devdata/seed/media（已尝试: %s）。%s",
		strings.Join(tried, "、"), assetsDirHint(),
	)
}

// assetsDirHint 是素材缺失时统一的修复提示（错误信息用）。
func assetsDirHint() string {
	return "开发素材外置在仓库根 devdata/seed/media，应有 " + assetManifestSummary() +
		"；不再打包进后端二进制。请确认仓库完整，或用 -assets-dir / SEED_ASSETS_DIR 显式指定。说明见 devdata/seed/README.md"
}

// loadAllAssets 读取并校验素材目录下的全部必备素材，返回 文件名 → 素材 的映射。
// 目录或任一文件缺失都明确报错（列出缺失清单），不做静默降级。
func loadAllAssets(dir string) (map[string]seedAsset, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("素材目录为空：请用 -assets-dir / SEED_ASSETS_DIR 指定。%s", assetsDirHint())
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("读取素材目录 %s 失败: %w。%s", dir, err, assetsDirHint())
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("素材路径 %s 不是目录。%s", dir, assetsDirHint())
	}
	out := make(map[string]seedAsset, len(assetManifest))
	missing := make([]string, 0, len(assetManifest))
	for _, name := range assetManifest {
		fullPath := filepath.Join(dir, name)
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			if errors.Is(readErr, fs.ErrNotExist) {
				missing = append(missing, name)
				continue
			}
			return nil, fmt.Errorf("读取素材 %s 失败: %w", fullPath, readErr)
		}
		if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 || data[2] != 0xFF {
			return nil, fmt.Errorf("素材 %s 不是合法 JPEG（魔数不符）：请放入真实的 .jpg 文件", fullPath)
		}
		cfg, _, decodeErr := image.DecodeConfig(bytes.NewReader(data))
		if decodeErr != nil {
			return nil, fmt.Errorf("解析素材 %s 尺寸失败: %w", fullPath, decodeErr)
		}
		wantWidth, wantHeight := expectedAssetSize(name)
		if wantWidth > 0 && (cfg.Width != wantWidth || cfg.Height != wantHeight) {
			return nil, fmt.Errorf("素材 %s 尺寸为 %dx%d，期望 %dx%d（替换素材时必须保持尺寸）", fullPath, cfg.Width, cfg.Height, wantWidth, wantHeight)
		}
		sum := md5.Sum(data)
		out[name] = seedAsset{
			Name:   name,
			Width:  cfg.Width,
			Height: cfg.Height,
			Size:   len(data),
			MD5:    hex.EncodeToString(sum[:]),
			Data:   data,
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"素材目录 %s 缺少 %d 个必需文件: %s。\n该目录应包含 %s。默认素材目录是仓库根 devdata/seed/media，可从仓库补全缺失文件，或用 -assets-dir / SEED_ASSETS_DIR 指向正确目录；说明见 devdata/seed/README.md",
			dir, len(missing), strings.Join(missing, "、"), assetManifestSummary(),
		)
	}
	return out, nil
}
