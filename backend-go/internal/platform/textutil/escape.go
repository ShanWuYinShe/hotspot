package textutil

import "strings"

// EscapeLikePattern 转义 LIKE/ILIKE 的 \、%、_ 通配符。
// 转义后的值必须配 SQL 的 ESCAPE '\' 使用（见 moments/users/groups 各搜索查询），
// 使 SQL 端的包含匹配与 Go 端 strings.Contains 的字面子串语义一致：
// 用户输入的通配符字符按字面量处理，不改变匹配语义。
func EscapeLikePattern(input string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return replacer.Replace(input)
}
