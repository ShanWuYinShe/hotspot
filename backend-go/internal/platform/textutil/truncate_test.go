package textutil

import "testing"

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
		{"empty", "", 3, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TruncateRunes(tc.in, tc.max); got != tc.want {
				t.Fatalf("TruncateRunes(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
			}
		})
	}
}
