package seed

import (
	"crypto/md5"
	"fmt"
	"testing"
)

// naming_test.go — 锁定 file_id / storage_key / chat_key 规则。
//
// media 模块的 buildFileID / storageKeyFor 是未导出的，seed 侧按规格 §4 复写
// 同式。本测试用「独立计算的期望值 + 黄金值」双重锁定：既验证公式一致，
// 也在公式被无意改动时立刻失败（若 media 侧改规则，这里需要同步修改）。

func TestBuildFileIDMatchesMediaRule(t *testing.T) {
	cases := []struct {
		userID   string
		uploadID string
		golden   string // 独立算出的 md5 hex（见注释中的计算式）
	}{
		// md5("user_seed_lumos:upload_seed_avatar_lumos")
		{"user_seed_lumos", "upload_seed_avatar_lumos", "81446bd7ea873d954a00d5afdfd4df2b"},
		// md5("user_seed_mika:upload_seed_09_img2")
		{"user_seed_mika", "upload_seed_09_img2", "cc70b7c53087d72b04ac6162e51287de"},
		// md5("user_seed_lumos:upload_seed_moment_01_img1")
		{"user_seed_lumos", "upload_seed_moment_01_img1", "fe87feb265284b0d71f642d57f9de2bf"},
	}
	for _, testCase := range cases {
		got := buildFileID(testCase.userID, testCase.uploadID)
		if want := "file_" + testCase.golden; got != want {
			t.Fatalf("buildFileID(%q, %q) = %q，期望 %q", testCase.userID, testCase.uploadID, got, want)
		}
		// 与 media.buildFileID 同式：file_ + md5(userID + ":" + uploadID) 的 hex。
		sum := md5.Sum([]byte(testCase.userID + ":" + testCase.uploadID))
		if want := fmt.Sprintf("file_%x", sum[:]); got != want {
			t.Fatalf("buildFileID(%q, %q) = %q，与 media 公式 %q 不一致", testCase.userID, testCase.uploadID, got, want)
		}
	}
}

func TestStorageKeyMatchesMediaRule(t *testing.T) {
	cases := []struct {
		name     string
		userID   string
		uploadID string
		fileName string
		want     string
	}{
		{
			name: "jpg 小写扩展名", userID: "user_seed_lumos", uploadID: "upload_seed_avatar_lumos",
			fileName: "avatar_lumos.jpg", want: "uploads/user_seed_lumos/upload_seed_avatar_lumos.jpg",
		},
		{
			name: "大写扩展名归一为小写", userID: "user_seed_mika", uploadID: "upload_seed_10_img1",
			fileName: "moment_10.JPG", want: "uploads/user_seed_mika/upload_seed_10_img1.jpg",
		},
		{
			name: "无扩展名回退 bin", userID: "user_seed_admin", uploadID: "upload_seed_raw",
			fileName: "noext", want: "uploads/user_seed_admin/upload_seed_raw.bin",
		},
		{
			name: "多点文件名取最后一段扩展名", userID: "u", uploadID: "upload_x",
			fileName: "a.b.c.jpeg", want: "uploads/u/upload_x.jpeg",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := storageKeyFor(testCase.userID, testCase.uploadID, testCase.fileName); got != testCase.want {
				t.Fatalf("storageKeyFor = %q，期望 %q", got, testCase.want)
			}
		})
	}
}

func TestChatKeyUsesSortedUserIDs(t *testing.T) {
	// privatechats 的口径：两个 user_id 字典序用 ":" 连接（与传参顺序无关）。
	key := chatKey("user_seed_lumos", "user_seed_sophia")
	if want := "user_seed_lumos:user_seed_sophia"; key != want {
		t.Fatalf("chatKey = %q，期望 %q", key, want)
	}
	if reversed := chatKey("user_seed_sophia", "user_seed_lumos"); reversed != key {
		t.Fatalf("chatKey 应与传参顺序无关，得到 %q 与 %q", key, reversed)
	}
	userA, userB := chatUsers("user_seed_sophia", "user_seed_lumos")
	if userA != "user_seed_lumos" || userB != "user_seed_sophia" {
		t.Fatalf("chatUsers = (%q, %q)，期望按字典序 (user_seed_lumos, user_seed_sophia)", userA, userB)
	}
}

func TestAvatarURLUsesPublicProxyPath(t *testing.T) {
	// 与 media.avatarPublicPathPrefix 一致：/public/v1/media/avatars/<file_id>。
	got := avatarURL("file_abc123")
	want := "/public/v1/media/avatars/file_abc123"
	if got != want {
		t.Fatalf("avatarURL = %q，期望 %q", got, want)
	}
}

func TestTwoDigits(t *testing.T) {
	if got := twoDigits(1); got != "01" {
		t.Fatalf("twoDigits(1) = %q", got)
	}
	if got := twoDigits(24); got != "24" {
		t.Fatalf("twoDigits(24) = %q", got)
	}
}
