package com.hotspot.app.util

import java.time.Instant

/**
 * 撤回时间窗（spec limits.message_recall_window_seconds，0 = 不允许撤回）。
 * 与 Web（lib/chat.svelte resolveRecallWindowMs）/ 鸿蒙（ChatMessageRules）同口径：
 * 空白或无法解析的时间戳按「可撤回」放行，由服务端 10400 兜底。
 */
object RecallWindowRules {
    const val FALLBACK_WINDOW_SECONDS: Int = 120

    fun isWithin(sentAt: String, windowSeconds: Int, nowMs: Long = System.currentTimeMillis()): Boolean {
        if (windowSeconds <= 0) return false
        if (sentAt.isBlank()) return true
        val sentMs = runCatching { Instant.parse(sentAt).toEpochMilli() }.getOrNull() ?: return true
        return nowMs - sentMs < windowSeconds * 1000L
    }
}
