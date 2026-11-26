package com.hotspot.app.util

/** 消息长按操作（私聊/群聊共用）：点赞、复制、撤回（自己的）、举报（他人的）。 */
enum class MessageAction { LIKE, COPY, RECALL, REPORT }

object MessageActionRules {
    /**
     * 根据消息归属/类型/撤回状态计算可用操作：
     * - 复制：仅未撤回的文本消息且内容非空；
     * - 撤回：自己发送的未撤回非系统消息（时间窗由服务端校验，超时返回 10400）；
     * - 举报：他人发送的未撤回非系统消息。
     */
    fun availableActions(
        isMine: Boolean,
        isRecalled: Boolean,
        type: String,
        content: String?,
    ): List<MessageAction> = buildList {
        if (isRecalled || type == "system") return@buildList
        add(MessageAction.LIKE)
        if (type == "text" && !content.isNullOrBlank()) add(MessageAction.COPY)
        if (isMine) add(MessageAction.RECALL) else add(MessageAction.REPORT)
    }
}
