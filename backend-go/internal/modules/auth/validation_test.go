package auth

import (
	"strings"
	"testing"
)

// 通用字段约束（spec api_spec_base.md「通用字段约束」）单测：
// password 8-128 字符且至少含字母和数字；nickname 1-30 字符（按 rune 计）。
func TestIsValidPassword(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"合规_字母数字", "secret12", true},
		{"下限8字符", "abcd1234", true},
		{"上限128字符", strings.Repeat("x", 126) + "a1", true},
		{"过短7字符", "pass123", false},
		{"纯字母", "abcdefgh", false},
		{"纯数字", "12345678", false},
		{"过长129字符", strings.Repeat("a", 64) + strings.Repeat("1", 65), false},
	}
	for _, tc := range cases {
		if got := isValidPassword(tc.in); got != tc.want {
			t.Fatalf("%s: isValidPassword(%q) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestIsValidNickname(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"空", "", false},
		{"单字符", "A", true},
		{"中文30字", strings.Repeat("热", 30), true},
		{"中文31字", strings.Repeat("热", 31), false},
		{"英文31字", strings.Repeat("a", 31), false},
	}
	for _, tc := range cases {
		if got := isValidNickname(tc.in); got != tc.want {
			t.Fatalf("%s: isValidNickname(%q) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}
