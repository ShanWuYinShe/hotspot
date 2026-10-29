package seed

import (
	"bytes"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg" // 注册 JPEG 解码器（image.DecodeConfig 校验素材尺寸）
	"io/fs"
	"path"
	"strings"
)

// assets.go — //go:embed 打包的开发素材。
//
// 素材由 Lead 统一提供（25 张 JPEG，尺寸固定）：
//   - avatar_*.jpg 400x400  用户头像（9 张）
//   - group_*.jpg  800x800  群头像（4 张）
//   - moment_*.jpg 1200x800 瞬间图 / 聊天图（12 张）
//
// seed 只读取不修改；尺寸在加载时校验（素材被替换但尺寸不符会立即失败，
// 避免把错误尺寸的 width/height 写进 media_upload_sessions 造成图片比例错乱）。
//
//go:embed assets/*.jpg
var embeddedAssets embed.FS

// assetDir 是 embed 内的素材目录前缀。
const assetDir = "assets"

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

// loadAllAssets 读取并校验全部打包素材，返回 文件名 → 素材 的映射。
// 只在真实执行（非 --dry-run）与素材单测中使用。
func loadAllAssets() (map[string]seedAsset, error) {
	names, err := fs.Glob(embeddedAssets, assetDir+"/*.jpg")
	if err != nil {
		return nil, fmt.Errorf("遍历打包素材失败: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("打包素材为空（//go:embed %s/*.jpg）", assetDir)
	}
	out := make(map[string]seedAsset, len(names))
	for _, fullPath := range names {
		name := path.Base(fullPath)
		data, err := embeddedAssets.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("读取素材 %s 失败: %w", name, err)
		}
		if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 || data[2] != 0xFF {
			return nil, fmt.Errorf("素材 %s 不是合法 JPEG（魔数不符）", name)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("解析素材 %s 尺寸失败: %w", name, err)
		}
		wantWidth, wantHeight := expectedAssetSize(name)
		if wantWidth > 0 && (cfg.Width != wantWidth || cfg.Height != wantHeight) {
			return nil, fmt.Errorf("素材 %s 尺寸为 %dx%d，期望 %dx%d", name, cfg.Width, cfg.Height, wantWidth, wantHeight)
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
	return out, nil
}
