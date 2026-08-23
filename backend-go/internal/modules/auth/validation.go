package auth

import (
	"unicode"
	"unicode/utf8"
)

// 密码与昵称的通用字段约束（spec api_spec_base.md「通用字段约束」）：
// - password：8-128 字符，至少含一个字母和一个数字；
// - nickname：1-30 字符（按 rune 计，中文昵称一字一符）。

// isValidPassword 校验密码复杂度。
func isValidPassword(p string) bool {
	if n := utf8.RuneCountInString(p); n < 8 || n > 128 {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range p {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

// isValidNickname 校验昵称长度约束（1-30 字符）。
func isValidNickname(n string) bool {
	count := utf8.RuneCountInString(n)
	return count >= 1 && count <= 30
}
