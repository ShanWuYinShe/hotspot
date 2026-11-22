package com.hotspot.app.util

/** 输入框内「@」查询的解析结果：startIndex 指向 @ 字符，keyword 为 @ 后已输入的关键字。 */
data class MentionQuery(
    val startIndex: Int,
    val keyword: String,
)

/**
 * 群聊 @ 提及的纯规则（spec/api_spec_chat.md：group_messages/send 的 mentions 为 @UserIDs，
 * 服务端仅向群成员推送 GroupMessageMentioned）。
 *
 * 输入侧：解析光标前最后一个未以空白结束的 @keyword，用于弹出成员候选与插入「@昵称 」；
 * 发送侧：只提交仍出现在最终文案里的已选成员 ID，避免插入后手动删除名字仍误 @。
 */
object MentionRules {
    /** 光标位置前若存在「@ + 非空白关键字」则返回查询（@ 必须在行首或空白之后）。 */
    fun findActiveQuery(text: String, cursor: Int = text.length): MentionQuery? {
        if (cursor <= 0 || cursor > text.length) return null
        val before = text.substring(0, cursor)
        val at = before.lastIndexOf('@')
        if (at < 0) return null
        if (at > 0 && !before[at - 1].isWhitespace()) return null
        val keyword = before.substring(at + 1)
        if (keyword.any { it.isWhitespace() }) return null
        return MentionQuery(at, keyword)
    }

    /** 把当前 @keyword 替换为「@昵称 」，命中候选后光标自然落在空格后。 */
    fun insertMention(
        text: String,
        cursor: Int,
        query: MentionQuery,
        nickname: String,
    ): String {
        val safeCursor = cursor.coerceIn(query.startIndex, text.length)
        return text.replaceRange(query.startIndex, safeCursor, "@" + nickname + " ")
    }

    /**
     * 发送前收集有效 mentions：已选 ID ∩ 成员仍存在 ∩ 最终文案仍包含「@昵称」（后跟空白或结尾）。
     * 返回去重后的 user_id 列表；结果稳定按已选顺序，便于单测。
     */
    fun activeMentionIds(
        text: String,
        candidates: List<Pair<String, String>>,
        selectedIds: Set<String>,
    ): List<String> = selectedIds.filter { id ->
        val nickname = candidates.firstOrNull { it.first == id }?.second
        nickname != null && containsMention(text, nickname)
    }

    fun containsMention(text: String, nickname: String): Boolean {
        if (nickname.isBlank()) return false
        val pattern = Regex("@${Regex.escape(nickname)}(?=\\s|$)")
        return pattern.containsMatchIn(text)
    }
}
