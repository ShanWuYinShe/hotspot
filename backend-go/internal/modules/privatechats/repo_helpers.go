package privatechats

// key 会话键（双方 user_id 字典序拼接）：Postgres 与内存实现共用。
func key(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}
