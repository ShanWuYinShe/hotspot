package com.hotspot.app.util

/**
 * 聊天「正在输入」展示规则（对齐 harmonyapp features/chat/components/ChatTypingRules 与
 * spec/api_spec_websocket.md 的 PrivateTyping / GroupTyping 语义）。
 *
 * 发送侧节流：输入中每 2 秒最多发一次 is_typing=true；停止输入 3 秒后发 false。
 * 接收侧自动清除：6 秒内没有新的 is_typing=true 就认为已停止输入（防止丢包导致提示常驻）。
 */
object ChatTypingRules {
    const val TYPING_THROTTLE_MS = 2_000L
    const val TYPING_IDLE_STOP_MS = 3_000L
    const val TYPING_AUTO_CLEAR_MS = 6_000L
    const val GROUP_TYPING_NAME_PREVIEW_LIMIT = 3

    /** 输入中的成员名：优先最近消息里的昵称，查不到回退「成员+userId 末 4 位」。 */
    fun resolveTypingDisplayName(userId: String, resolveName: (String) -> String?): String {
        val nickname = resolveName(userId)?.trim().orEmpty()
        if (nickname.isNotEmpty()) return nickname
        val suffix = if (userId.length > 4) userId.takeLast(4) else userId
        return "成员" + suffix
    }

    /**
     * 群聊输入中提示：1 人「A 正在输入…」；≤3 人「A、B 正在输入…」；
     * 超过 3 人「A、B、C 等 N 人正在输入…」。
     */
    fun buildGroupTypingHint(userIds: List<String>, resolveName: (String) -> String?): String {
        val ids = userIds.filter { it.isNotBlank() }.distinct()
        if (ids.isEmpty()) return ""
        val previewCount = minOf(ids.size, GROUP_TYPING_NAME_PREVIEW_LIMIT)
        val names = (0 until previewCount).map { index ->
            resolveTypingDisplayName(ids[index], resolveName)
        }
        return when {
            ids.size == 1 -> names[0] + " 正在输入…"
            ids.size <= GROUP_TYPING_NAME_PREVIEW_LIMIT -> names.joinToString("、") + " 正在输入…"
            else -> names.joinToString("、") + " 等 " + ids.size + " 人正在输入…"
        }
    }
}
