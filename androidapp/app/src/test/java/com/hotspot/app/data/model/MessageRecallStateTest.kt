package com.hotspot.app.data.model

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** 服务端未撤回时 recalled_at 下发空字符串，端上必须按「空串=未撤回」处理。 */
class MessageRecallStateTest {
    @Test
    fun privateMessageBlankOrNullRecalledAtIsNotRecalled() {
        assertFalse(PrivateMessageItem(recalledAt = null).isRecalled)
        assertFalse(PrivateMessageItem(recalledAt = "").isRecalled)
        assertTrue(PrivateMessageItem(recalledAt = "2026-11-22T00:00:00Z").isRecalled)
    }

    @Test
    fun groupMessageBlankOrNullRecalledAtIsNotRecalled() {
        assertFalse(GroupMessageItem(recalledAt = null).isRecalled)
        assertFalse(GroupMessageItem(recalledAt = "").isRecalled)
        assertTrue(GroupMessageItem(recalledAt = "2026-11-22T00:00:00Z").isRecalled)
    }
}
