package seed

import "testing"

// assets_test.go — 素材完整性校验：数量、尺寸、JPEG 魔数。
// 素材由 Lead 统一提供，seed 不应改动图片；本测试防止误删/替换导致
// media_upload_sessions 的 width/height 与实际对象不符（图片比例错乱）。

func TestLoadAllAssets(t *testing.T) {
	assets, err := loadAllAssets()
	if err != nil {
		t.Fatalf("加载素材失败: %v", err)
	}
	// 9 张头像 + 4 张群头像 + 12 张瞬间图 = 25 张。
	if len(assets) != 25 {
		t.Fatalf("素材数量 = %d，期望 25", len(assets))
	}
	for _, name := range []string{
		"avatar_lumos.jpg", "avatar_sophia.jpg", "avatar_marco.jpg", "avatar_mika.jpg",
		"avatar_chen.jpg", "avatar_lin.jpg", "avatar_xiaoman.jpg", "avatar_admin.jpg", "avatar_privacy.jpg",
		"group_1.jpg", "group_2.jpg", "group_3.jpg", "group_4.jpg",
		"moment_01.jpg", "moment_02.jpg", "moment_03.jpg", "moment_04.jpg",
		"moment_05.jpg", "moment_06.jpg", "moment_07.jpg", "moment_08.jpg",
		"moment_09.jpg", "moment_10.jpg", "moment_11.jpg", "moment_12.jpg",
	} {
		if _, ok := assets[name]; !ok {
			t.Fatalf("缺少素材 %s", name)
		}
	}
	for name, asset := range assets {
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
}

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
