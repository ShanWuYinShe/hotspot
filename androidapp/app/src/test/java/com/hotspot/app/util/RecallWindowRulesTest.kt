package com.hotspot.app.util

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class RecallWindowRulesTest {
    private val now = 1_700_000_000_000L

    @Test
    fun `窗口内可撤回`() {
        val sentAt = java.time.Instant.ofEpochMilli(now - 60_000).toString()
        assertTrue(RecallWindowRules.isWithin(sentAt, 120, now))
    }

    @Test
    fun `超出窗口不可撤回`() {
        val sentAt = java.time.Instant.ofEpochMilli(now - 180_000).toString()
        assertFalse(RecallWindowRules.isWithin(sentAt, 120, now))
    }

    @Test
    fun `0 表示不允许撤回`() {
        val sentAt = java.time.Instant.ofEpochMilli(now).toString()
        assertFalse(RecallWindowRules.isWithin(sentAt, 0, now))
    }

    @Test
    fun `空白或非法时间戳放行给服务端兜底`() {
        assertTrue(RecallWindowRules.isWithin("", 120, now))
        assertTrue(RecallWindowRules.isWithin("not-a-date", 120, now))
    }

    @Test
    fun `兜底窗口与后端默认一致`() {
        assert(RecallWindowRules.FALLBACK_WINDOW_SECONDS == 120)
    }
}
