package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"short passthrough", "hello", 10, "hello"},
		{"trims space first", "  hello  ", 10, "hello"},
		{"ascii cut", "hello world", 5, "hello"},
		{"中文按字截断", "你好世界朋友", 4, "你好世界"},
		{"中文不被字节截断", "你好世界朋友", 2, "你好"},
		{"emoji 单rune", "👍👍👍👍", 2, "👍👍"},
		{"混合中英文", "你好abc世界", 6, "你好abc世"},
		{"maxRunes 为 0", "hello", 0, ""},
		{"首尾空白+中文", "  你好世界  ", 4, "你好世界"},
		{"empty", "", 3, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateRunes(tc.in, tc.max)
			if got != tc.want {
				t.Fatalf("TruncateRunes(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
			}
			// 截断结果必须是合法 UTF-8（不允许出现 U+FFFD 替换字符）
			if !utf8.ValidString(got) {
				t.Fatalf("result is not valid UTF-8: %q", got)
			}
			if strings.ContainsRune(got, utf8.RuneError) {
				t.Fatalf("result contains U+FFFD replacement char: %q", got)
			}
		})
	}
}
