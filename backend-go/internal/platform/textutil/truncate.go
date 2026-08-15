// Package textutil 提供跨模块共用的文本处理工具。
package textutil

import "strings"

// TruncateRunes 按 rune 截断文本（先去首尾空白），避免多字节字符（中文等）
// 被字节截断产生非法 UTF-8 / U+FFFD 乱码。maxRunes 为最终保留的最大 rune 数。
func TruncateRunes(s string, maxRunes int) string {
	value := strings.TrimSpace(s)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
