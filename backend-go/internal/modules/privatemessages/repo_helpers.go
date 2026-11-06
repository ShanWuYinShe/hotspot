package privatemessages

import "strings"

// conversationKey 会话键（双方 user_id 字典序拼接）：PG 与内存实现共用。
func conversationKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// otherUserID 从会话键解析「对方 user_id」：PG 与内存实现共用。
func otherUserID(chatKey, userID string) string {
	parts := strings.Split(chatKey, ":")
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == userID {
		return parts[1]
	}
	return parts[0]
}
