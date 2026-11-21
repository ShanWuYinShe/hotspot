package com.hotspot.app.util

/**
 * 举报枚举文案（spec/api_spec_types.md → ReportReason / ReportTargetType / ReportStatus）：
 * 我的举报列表与他人主页举报弹窗共用一份映射，避免两端文案漂移。
 */
object ReportCatalog {
    data class Reason(val key: String, val label: String)

    val reasons: List<Reason> = listOf(
        Reason("spam", "垃圾信息"),
        Reason("harassment", "骚扰/欺凌"),
        Reason("violence", "暴力/威胁"),
        Reason("hate_speech", "仇恨言论"),
        Reason("sexual_content", "色情/裸露内容"),
        Reason("misinformation", "虚假信息"),
        Reason("impersonation", "冒充他人"),
        Reason("self_harm", "自伤/自杀相关"),
        Reason("other", "其他"),
    )

    fun reasonLabel(key: String): String = reasons.firstOrNull { it.key == key }?.label ?: key

    fun targetTypeLabel(type: String): String = when (type) {
        "user" -> "用户"
        "moment" -> "瞬间"
        "comment" -> "评论"
        "message" -> "消息"
        "group" -> "群组"
        else -> type
    }
}
